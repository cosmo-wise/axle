package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cosmo-wise/axle/internal/schema"
	"github.com/cosmo-wise/axle/pkg/axle"
)

type CatalogMigrationConfig struct {
	Component string
	Version   int
	Name      string
	Checksum  string
	Catalog   axle.Catalog
}

func CatalogMigrationSet(cfg CatalogMigrationConfig) []MigrationDef {
	if cfg.Component == "" {
		cfg.Component = "axle.catalog"
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Name == "" {
		cfg.Name = "catalog_baseline"
	}
	if cfg.Checksum == "" {
		cfg.Checksum = "0000000000000000000000000000000000000000000000000000000000000001"
	}
	return []MigrationDef{
		{
			Component: cfg.Component,
			Version:   cfg.Version,
			Name:      cfg.Name,
			Checksum:  cfg.Checksum,
			Up: func(ctx context.Context, tx *sql.Tx) error {
				return reconcileCatalog(ctx, tx, cfg.Catalog)
			},
			Verify: func(ctx context.Context, tx *sql.Tx) error {
				return verifyCatalog(ctx, tx, cfg.Catalog)
			},
		},
	}
}

func reconcileCatalog(ctx context.Context, tx *sql.Tx, catalog axle.Catalog) error {
	seen := map[string]bool{}
	for _, registry := range catalog.Resources {
		res := registry.Resource
		key := strings.ToLower(strings.TrimSpace(res.Table))
		if seen[key] {
			continue
		}
		seen[key] = true

		table, err := schema.QuoteIdent(res.Table)
		if err != nil {
			return err
		}
		columns := make([]string, 0, len(res.Fields))
		for _, field := range res.Fields {
			column, err := schema.ColumnDefinition(res, field, true)
			if err != nil {
				return err
			}
			columns = append(columns, column)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table, strings.Join(columns, ", "))); err != nil {
			return err
		}
		if err := addMissingColumnsTx(ctx, tx, res, table); err != nil {
			return err
		}
		indexes, err := schema.IndexStatements(res)
		if err != nil {
			return err
		}
		for _, statement := range indexes {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
	}
	return nil
}

func addMissingColumnsTx(ctx context.Context, tx *sql.Tx, res axle.ResourceDescriptor, table string) error {
	rows, err := tx.QueryContext(ctx, "PRAGMA table_info("+schema.MustQuoteIdent(res.Table)+")")
	if err != nil {
		return err
	}
	defer rows.Close()
	existing := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, field := range res.Fields {
		if existing[field.Name] {
			continue
		}
		column, err := schema.ColumnDefinition(res, field, false)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, column)); err != nil {
			return err
		}
	}
	return nil
}

func verifyCatalog(ctx context.Context, tx *sql.Tx, catalog axle.Catalog) error {
	seen := map[string]bool{}
	for _, registry := range catalog.Resources {
		res := registry.Resource
		key := strings.ToLower(strings.TrimSpace(res.Table))
		if seen[key] {
			continue
		}
		seen[key] = true

		rows, err := tx.QueryContext(ctx, "PRAGMA table_info("+schema.MustQuoteIdent(res.Table)+")")
		if err != nil {
			return fmt.Errorf("verify table %s: %w", res.Table, err)
		}
		existing := map[string]bool{}
		for rows.Next() {
			var cid int
			var name, typ string
			var notNull, pk int
			var defaultValue any
			if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
				rows.Close()
				return err
			}
			existing[name] = true
		}
		rows.Close()
		for _, field := range res.Fields {
			if !existing[field.Name] {
				return fmt.Errorf("verify table %s: missing column %s", res.Table, field.Name)
			}
		}
	}
	return nil
}
