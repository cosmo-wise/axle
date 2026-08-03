package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type StatusKind string

const (
	StatusCurrent        StatusKind = "current"
	StatusPending        StatusKind = "pending"
	StatusChecksumDrift  StatusKind = "checksum_drift"
	StatusDatabaseTooOld StatusKind = "database_too_old"
	StatusDatabaseTooNew StatusKind = "database_too_new"
	StatusInvalidSet     StatusKind = "invalid_set"
	StatusVerifyFailed   StatusKind = "verify_failed"
)

type MigrationError struct {
	Kind    StatusKind
	Message string
	Cause   error
}

func (e *MigrationError) Error() string { return fmt.Sprintf("migration %s: %s", e.Kind, e.Message) }
func (e *MigrationError) Unwrap() error { return e.Cause }

var checksumPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type MigrationDef struct {
	Component string
	Version   int
	Name      string
	Checksum  string
	Up        func(context.Context, *sql.Tx) error
	Verify    func(context.Context, *sql.Tx) error
}

func ValidateSet(component string, migrations []MigrationDef, targetVersion int) error {
	if strings.TrimSpace(component) == "" {
		return &MigrationError{Kind: StatusInvalidSet, Message: "empty component"}
	}
	if targetVersion < 0 {
		return &MigrationError{Kind: StatusInvalidSet, Message: "target version cannot be negative"}
	}
	if len(migrations) == 0 && targetVersion != 0 {
		return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("target version %d requires migrations", targetVersion)}
	}
	seen := map[int]bool{}
	for i, m := range migrations {
		if m.Component != component {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("migration %d component %q != set component %q", m.Version, m.Component, component)}
		}
		if m.Version < 1 {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("migration version %d must be >= 1", m.Version)}
		}
		if seen[m.Version] {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("duplicate version %d", m.Version)}
		}
		seen[m.Version] = true
		if strings.TrimSpace(m.Name) == "" {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("version %d has empty name", m.Version)}
		}
		if !checksumPattern.MatchString(m.Checksum) {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("version %d checksum must be 64 lowercase hexadecimal characters", m.Version)}
		}
		if m.Up == nil {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("version %d has nil Up", m.Version)}
		}
		if i == 0 && m.Version != 1 {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("first migration version %d must be 1", m.Version)}
		}
		if i > 0 && m.Version != migrations[i-1].Version+1 {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("version gap: %d after %d", m.Version, migrations[i-1].Version)}
		}
	}
	if len(migrations) > 0 && migrations[len(migrations)-1].Version != targetVersion {
		return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("target version %d != last migration version %d", targetVersion, migrations[len(migrations)-1].Version)}
	}
	return nil
}

func ApplyMigrations(ctx context.Context, db *sql.DB, component string, migrations []MigrationDef, minCompatible, targetVersion int) error {
	if err := EnsureRegistry(ctx, db); err != nil {
		return err
	}
	if err := ValidateSet(component, migrations, targetVersion); err != nil {
		return err
	}

	records, err := LoadRecords(ctx, db, component)
	if err != nil {
		return err
	}
	current := MaxMigrationVersion(records)

	if current > targetVersion {
		return &MigrationError{Kind: StatusDatabaseTooNew, Message: fmt.Sprintf("component %s: database version %d > binary target %d", component, current, targetVersion)}
	}
	if current > 0 && current < minCompatible {
		return &MigrationError{Kind: StatusDatabaseTooOld, Message: fmt.Sprintf("component %s: database version %d < min compatible %d", component, current, minCompatible)}
	}

	recordMap := map[int]MigrationRecord{}
	for _, r := range records {
		recordMap[r.Version] = r
	}
	for _, m := range migrations {
		if m.Version > current {
			continue
		}
		existing, ok := recordMap[m.Version]
		if !ok {
			return &MigrationError{Kind: StatusChecksumDrift, Message: fmt.Sprintf("component %s version %d: applied but not in registry", component, m.Version)}
		}
		if existing.Checksum != m.Checksum {
			return &MigrationError{Kind: StatusChecksumDrift, Message: fmt.Sprintf("component %s version %d: checksum %s != binary %s", component, m.Version, existing.Checksum, m.Checksum)}
		}
	}

	for _, m := range migrations {
		if m.Version <= current {
			continue
		}
		if err := applyOne(ctx, db, m); err != nil {
			return err
		}
	}

	if targetVersion > 0 {
		for _, m := range migrations {
			if m.Version == targetVersion && m.Verify != nil {
				tx, err := db.BeginTx(ctx, nil)
				if err != nil {
					return err
				}
				defer tx.Rollback()
				if err := m.Verify(ctx, tx); err != nil {
					return &MigrationError{Kind: StatusVerifyFailed, Message: fmt.Sprintf("component %s target verify: %v", component, err)}
				}
				if err := tx.Commit(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func applyOne(ctx context.Context, db *sql.DB, m MigrationDef) error {
	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		err := applyOneAttempt(ctx, db, m)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isSQLiteBusy(err) {
			return err
		}
		delay := time.Duration(attempt+1) * 25 * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

func applyOneAttempt(ctx context.Context, db *sql.DB, m MigrationDef) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingChecksum string
	err = tx.QueryRowContext(ctx, `SELECT checksum FROM _axle_migrations WHERE component = ? AND version = ?`, m.Component, m.Version).Scan(&existingChecksum)
	if err == nil {
		if existingChecksum == m.Checksum {
			return tx.Commit()
		}
		return &MigrationError{Kind: StatusChecksumDrift, Message: fmt.Sprintf("component %s version %d: concurrent checksum mismatch", m.Component, m.Version)}
	}
	if err != sql.ErrNoRows {
		return err
	}

	start := time.Now()
	if err := m.Up(ctx, tx); err != nil {
		return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("component %s version %d up failed: %v", m.Component, m.Version, err), Cause: err}
	}
	if m.Verify != nil {
		if err := m.Verify(ctx, tx); err != nil {
			return &MigrationError{Kind: StatusVerifyFailed, Message: fmt.Sprintf("component %s version %d verify failed: %v", m.Component, m.Version, err)}
		}
	}
	duration := time.Since(start).Milliseconds()

	if err := insertRecord(ctx, tx, MigrationRecord{
		Component:  m.Component,
		Version:    m.Version,
		Name:       m.Name,
		Checksum:   m.Checksum,
		AppliedAt:  nowRFC3339(),
		DurationMs: duration,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "sqlite_busy") ||
		strings.Contains(message, "sqlite_locked")
}
