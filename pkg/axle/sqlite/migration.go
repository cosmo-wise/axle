package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	internalsqlite "github.com/cosmo-wise/axle/internal/sqlite"
	"github.com/cosmo-wise/axle/pkg/axle"
)

type Migration struct {
	Component string
	Version   int
	Name      string
	Checksum  string
	Up        func(context.Context, *sql.Tx) error
	Verify    func(context.Context, *sql.Tx) error
}

type MigrationSet struct {
	Component     string
	Migrations    []Migration
	MinCompatible int
	TargetVersion int
}

type PlanEntry struct {
	Component string
	Version   int
	Name      string
	Checksum  string
}

type Plan struct {
	Components []ComponentPlan
	Entries    []PlanEntry
}

type ComponentPlan struct {
	Component      string
	CurrentVersion int
	TargetVersion  int
	Status         StatusKind
}

type StatusKind = internalsqlite.StatusKind

const (
	StatusCurrent        = internalsqlite.StatusCurrent
	StatusPending        = internalsqlite.StatusPending
	StatusChecksumDrift  = internalsqlite.StatusChecksumDrift
	StatusDatabaseTooOld = internalsqlite.StatusDatabaseTooOld
	StatusDatabaseTooNew = internalsqlite.StatusDatabaseTooNew
	StatusInvalidSet     = internalsqlite.StatusInvalidSet
	StatusVerifyFailed   = internalsqlite.StatusVerifyFailed
)

type MigrationError = internalsqlite.MigrationError

type ComponentStatus struct {
	Component      string
	CurrentVersion int
	TargetVersion  int
	Status         StatusKind
}

type Status struct {
	Components []ComponentStatus
}

type CatalogMigrationConfig = internalsqlite.CatalogMigrationConfig

func CatalogMigrationSet(cfg CatalogMigrationConfig) []Migration {
	defs := internalsqlite.CatalogMigrationSet(cfg)
	migrations := make([]Migration, 0, len(defs))
	for _, d := range defs {
		migrations = append(migrations, Migration{
			Component: d.Component,
			Version:   d.Version,
			Name:      d.Name,
			Checksum:  d.Checksum,
			Up:        d.Up,
			Verify:    d.Verify,
		})
	}
	return migrations
}

// CatalogSet returns the versioned migration set represented by a generated catalog.
// Legacy catalogs without explicit metadata receive a stable component derived from
// their table names and schema version 1.
func CatalogSet(catalog axle.Catalog) MigrationSet {
	component := strings.TrimSpace(catalog.Component)
	if component == "" {
		tables := make([]string, 0, len(catalog.Resources))
		for _, registry := range catalog.Resources {
			tables = append(tables, strings.ToLower(strings.TrimSpace(registry.Resource.Table)))
		}
		sort.Strings(tables)
		sum := sha256.Sum256([]byte(strings.Join(tables, "\x00")))
		component = "axle.catalog:legacy-" + hex.EncodeToString(sum[:8])
	}
	version := catalog.Version
	if version == 0 {
		version = 1
	}
	payload, _ := json.Marshal(catalog.Resources)
	checksumBytes := sha256.Sum256(payload)
	checksum := hex.EncodeToString(checksumBytes[:])
	migrations := CatalogMigrationSet(CatalogMigrationConfig{
		Component: component,
		Version:   version,
		Name:      "catalog_schema",
		Checksum:  checksum,
		Catalog:   catalog,
	})
	return MigrationSet{
		Component:     component,
		Migrations:    migrations,
		MinCompatible: 1,
		TargetVersion: version,
	}
}
