package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

type registryKind int

const (
	registryAbsent registryKind = iota
	registryCurrent
	registryLegacyEmpty
	registryLegacyPopulated
)

type MigrationRecord struct {
	Component  string
	Version    int
	Name       string
	Checksum   string
	AppliedAt  string
	DurationMs int64
}

func EnsureRegistry(ctx context.Context, db *sql.DB) error {
	kind, count, err := inspectRegistry(ctx, db)
	if err != nil {
		return err
	}
	switch kind {
	case registryCurrent:
		return nil
	case registryLegacyPopulated:
		return legacyRegistryError(count)
	case registryLegacyEmpty:
		return bootstrapLegacyRegistry(ctx, db)
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS _axle_migrations (
		component TEXT NOT NULL,
		version INTEGER NOT NULL,
		name TEXT NOT NULL,
		checksum TEXT NOT NULL,
		applied_at TEXT NOT NULL,
		duration_ms INTEGER NOT NULL,
		PRIMARY KEY (component, version)
	)`)
	return err
}

func inspectRegistry(ctx context.Context, db *sql.DB) (registryKind, int, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(_axle_migrations)`)
	if err != nil {
		return registryAbsent, 0, err
	}

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			rows.Close()
			return registryAbsent, 0, err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return registryAbsent, 0, err
	}
	rows.Close()
	if len(columns) == 0 {
		return registryAbsent, 0, nil
	}
	currentColumns := []string{"component", "version", "name", "checksum", "applied_at", "duration_ms"}
	if hasAllColumns(columns, currentColumns) {
		return registryCurrent, 0, nil
	}
	legacyColumns := []string{"version", "name", "applied_at"}
	if !hasAllColumns(columns, legacyColumns) || columns["component"] || columns["checksum"] || columns["duration_ms"] {
		return registryAbsent, 0, malformedRegistryError(columns)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM _axle_migrations`).Scan(&count); err != nil {
		return registryAbsent, 0, err
	}
	if count > 0 {
		return registryLegacyPopulated, count, nil
	}
	return registryLegacyEmpty, 0, nil
}

func hasAllColumns(columns map[string]bool, required []string) bool {
	for _, column := range required {
		if !columns[column] {
			return false
		}
	}
	return true
}

func malformedRegistryError(columns map[string]bool) error {
	names := make([]string, 0, len(columns))
	for name := range columns {
		names = append(names, name)
	}
	sort.Strings(names)
	return &MigrationError{
		Kind:    StatusInvalidSet,
		Message: "malformed _axle_migrations schema with columns " + strings.Join(names, ","),
	}
}

func legacyRegistryError(count int) error {
	return &MigrationError{
		Kind:    StatusInvalidSet,
		Message: fmt.Sprintf("legacy _axle_migrations contains %d rows without component/checksum; manual intervention required", count),
	}
}

func bootstrapLegacyRegistry(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `ALTER TABLE _axle_migrations RENAME TO _axle_migrations_legacy_tmp`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE _axle_migrations (
		component TEXT NOT NULL,
		version INTEGER NOT NULL,
		name TEXT NOT NULL,
		checksum TEXT NOT NULL,
		applied_at TEXT NOT NULL,
		duration_ms INTEGER NOT NULL,
		PRIMARY KEY (component, version)
	)`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE _axle_migrations_legacy_tmp`); err != nil {
		return err
	}
	return tx.Commit()
}

func LoadRecords(ctx context.Context, db *sql.DB, component string) ([]MigrationRecord, error) {
	kind, count, err := inspectRegistry(ctx, db)
	if err != nil {
		return nil, err
	}
	switch kind {
	case registryAbsent, registryLegacyEmpty:
		return nil, nil
	case registryLegacyPopulated:
		return nil, legacyRegistryError(count)
	}
	rows, err := db.QueryContext(ctx,
		`SELECT component, version, name, checksum, applied_at, duration_ms FROM _axle_migrations WHERE component = ? ORDER BY version`,
		component,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []MigrationRecord
	for rows.Next() {
		var r MigrationRecord
		if err := rows.Scan(&r.Component, &r.Version, &r.Name, &r.Checksum, &r.AppliedAt, &r.DurationMs); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func insertRecord(ctx context.Context, tx *sql.Tx, r MigrationRecord) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO _axle_migrations (component, version, name, checksum, applied_at, duration_ms) VALUES (?, ?, ?, ?, ?, ?)`,
		r.Component, r.Version, r.Name, r.Checksum, r.AppliedAt, r.DurationMs,
	)
	return err
}

func MaxMigrationVersion(records []MigrationRecord) int {
	max := 0
	for _, r := range records {
		if r.Version > max {
			max = r.Version
		}
	}
	return max
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }
