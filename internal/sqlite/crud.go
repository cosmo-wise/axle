package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosmo-wise/axle/internal/schema"
)

func (s Store) Create(ctx context.Context, values map[string]any) error {
	filtered, err := s.createValues(values)
	if err != nil {
		return err
	}
	keys := sortedKeys(filtered)
	placeholders := make([]string, len(keys))
	columns := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, key := range keys {
		quoted, err := schema.QuoteIdent(key)
		if err != nil {
			return err
		}
		columns[i] = quoted
		placeholders[i] = "?"
		args[i] = filtered[key]
	}
	table, err := schema.QuoteIdent(s.resource.Table)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s Store) List(ctx context.Context) ([]map[string]any, error) {
	columns := s.quotedFieldNames()
	table, err := schema.QuoteIdent(s.resource.Table)
	if err != nil {
		return nil, err
	}
	id, err := schema.QuoteIdent(s.resource.ID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %s ORDER BY %s", strings.Join(columns, ", "), table, id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows, s.fieldNames())
}

func (s Store) Get(ctx context.Context, id any) (map[string]any, error) {
	columns := s.quotedFieldNames()
	table, err := schema.QuoteIdent(s.resource.Table)
	if err != nil {
		return nil, err
	}
	idColumn, err := schema.QuoteIdent(s.resource.ID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", strings.Join(columns, ", "), table, idColumn), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanRows(rows, s.fieldNames())
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items[0], nil
}

func (s Store) Update(ctx context.Context, id any, values map[string]any) error {
	filtered, err := s.updateValues(values)
	if err != nil {
		return err
	}
	keys := sortedKeys(filtered)
	if len(keys) == 0 {
		_, err := s.Get(ctx, id)
		return err
	}
	sets := make([]string, len(keys))
	args := make([]any, 0, len(keys)+1)
	for i, key := range keys {
		quoted, err := schema.QuoteIdent(key)
		if err != nil {
			return err
		}
		sets[i] = quoted + " = ?"
		args = append(args, filtered[key])
	}
	args = append(args, id)
	table, err := schema.QuoteIdent(s.resource.Table)
	if err != nil {
		return err
	}
	idColumn, err := schema.QuoteIdent(s.resource.ID)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", table, strings.Join(sets, ", "), idColumn), args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return ErrNotFound
	}
	return nil
}

func (s Store) Delete(ctx context.Context, id any) error {
	table, err := schema.QuoteIdent(s.resource.Table)
	if err != nil {
		return err
	}
	idColumn, err := schema.QuoteIdent(s.resource.ID)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table, idColumn), id)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return ErrNotFound
	}
	return nil
}
