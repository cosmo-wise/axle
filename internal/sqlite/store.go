package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/Fel1xKan/axle/internal/schema"
	"github.com/Fel1xKan/axle/pkg/axle"
)

// ErrNotFound is returned when a descriptor-backed row does not exist.
var ErrNotFound = errors.New("record not found")

// ErrUnknownField is returned when caller input contains fields outside the descriptor.
var ErrUnknownField = errors.New("unknown descriptor field")

// ErrImmutableField is returned when caller input attempts to patch an immutable field.
var ErrImmutableField = errors.New("immutable descriptor field")

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

func (s Store) createValues(values map[string]any) (map[string]any, error) {
	fields := s.fieldsByName()
	filtered := map[string]any{}
	for key, value := range values {
		if _, ok := fields[key]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownField, key)
		}
		filtered[key] = value
	}
	for _, field := range s.resource.Fields {
		if field.Auto == "uuid" && isEmpty(filtered[field.Name]) {
			id, err := newUUID()
			if err != nil {
				return nil, err
			}
			filtered[field.Name] = id
			values[field.Name] = id
		}
	}
	return filtered, nil
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

func (s Store) updateValues(values map[string]any) (map[string]any, error) {
	fields := s.fieldsByName()
	filtered := map[string]any{}
	for key, value := range values {
		field, ok := fields[key]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownField, key)
		}
		if !field.Mutable {
			return nil, fmt.Errorf("%w: %s", ErrImmutableField, key)
		}
		filtered[key] = value
	}
	return filtered, nil
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

func (s Store) fieldNames() []string {
	names := make([]string, 0, len(s.resource.Fields))
	for _, field := range s.resource.Fields {
		names = append(names, field.Name)
	}
	return names
}

func (s Store) quotedFieldNames() []string {
	names := make([]string, 0, len(s.resource.Fields))
	for _, field := range s.resource.Fields {
		names = append(names, schema.MustQuoteIdent(field.Name))
	}
	return names
}

func (s Store) fieldsByName() map[string]axle.FieldDescriptor {
	fields := map[string]axle.FieldDescriptor{}
	for _, field := range s.resource.Fields {
		fields[field.Name] = field
	}
	return fields
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

func isEmpty(value any) bool {
	if value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	return false
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	buf := make([]byte, 36)
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], b[10:])
	return string(buf), nil
}
