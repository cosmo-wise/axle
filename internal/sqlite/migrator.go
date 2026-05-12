package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosmo-wise/axle/internal/schema"
)

func (s Store) Migrate(ctx context.Context) error {
	table, err := schema.QuoteIdent(s.resource.Table)
	if err != nil {
		return err
	}
	columns := make([]string, 0, len(s.resource.Fields))
	for _, field := range s.resource.Fields {
		column, err := schema.ColumnDefinition(s.resource, field, true)
		if err != nil {
			return err
		}
		columns = append(columns, column)
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table, strings.Join(columns, ", "))); err != nil {
		return err
	}
	if err := s.addMissingColumns(ctx, table); err != nil {
		return err
	}
	indexes, err := schema.IndexStatements(s.resource)
	if err != nil {
		return err
	}
	for _, statement := range indexes {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) addMissingColumns(ctx context.Context, table string) error {
	existing, err := s.existingColumns(ctx)
	if err != nil {
		return err
	}
	for _, field := range s.resource.Fields {
		if existing[field.Name] {
			continue
		}
		column, err := schema.ColumnDefinition(s.resource, field, false)
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, column)); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) existingColumns(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, "PRAGMA table_info("+schema.MustQuoteIdent(s.resource.Table)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}
