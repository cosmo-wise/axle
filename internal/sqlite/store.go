package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/Fel1xKan/axle/pkg/axle"
)

// ErrNotFound is returned when a descriptor-backed row does not exist.
var ErrNotFound = errors.New("record not found")

// Open opens a concrete SQLite database/sql handle. V1 intentionally supports only SQLite.
func Open(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite", dsn)
}

// Store is a small SQLite CRUD helper for descriptor-backed fixtures.
type Store struct {
	db       *sql.DB
	resource axle.ResourceDescriptor
}

func NewStore(db *sql.DB, resource axle.ResourceDescriptor) Store {
	return Store{db: db, resource: resource}
}

func (s Store) Migrate(ctx context.Context) error {
	columns := make([]string, 0, len(s.resource.Fields))
	for _, field := range s.resource.Fields {
		column := field.Name + " " + sqliteType(field.Type)
		if field.Name == s.resource.ID {
			column += " PRIMARY KEY"
		}
		columns = append(columns, column)
	}
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", s.resource.Table, strings.Join(columns, ", ")))
	return err
}

func (s Store) Create(ctx context.Context, values map[string]any) error {
	keys := sortedKeys(values)
	placeholders := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, key := range keys {
		placeholders[i] = "?"
		args[i] = values[key]
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", s.resource.Table, strings.Join(keys, ", "), strings.Join(placeholders, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s Store) List(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %s ORDER BY %s", strings.Join(s.fieldNames(), ", "), s.resource.Table, s.resource.ID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows, s.fieldNames())
}

func (s Store) Get(ctx context.Context, id any) (map[string]any, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", strings.Join(s.fieldNames(), ", "), s.resource.Table, s.resource.ID), id)
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
	keys := sortedKeys(values)
	if len(keys) == 0 {
		_, err := s.Get(ctx, id)
		return err
	}
	sets := make([]string, len(keys))
	args := make([]any, 0, len(keys)+1)
	for i, key := range keys {
		sets[i] = key + " = ?"
		args = append(args, values[key])
	}
	args = append(args, id)
	result, err := s.db.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", s.resource.Table, strings.Join(sets, ", "), s.resource.ID), args...)
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
	result, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = ?", s.resource.Table, s.resource.ID), id)
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

func (s Store) fieldNames() []string {
	names := make([]string, 0, len(s.resource.Fields))
	for _, field := range s.resource.Fields {
		names = append(names, field.Name)
	}
	return names
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func scanRows(rows *sql.Rows, columns []string) ([]map[string]any, error) {
	var items []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		item := map[string]any{}
		for i, column := range columns {
			item[column] = values[i]
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func sqliteType(kind string) string {
	switch kind {
	case "integer":
		return "INTEGER"
	case "boolean":
		return "INTEGER"
	case "real":
		return "REAL"
	default:
		return "TEXT"
	}
}
