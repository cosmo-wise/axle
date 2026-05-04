package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	internalsqlite "github.com/cosmo-wise/axle/internal/sqlite"
	"github.com/cosmo-wise/axle/pkg/axle"
)

// ErrNotFound is returned when a generated runtime route addresses a missing row.
var ErrNotFound = internalsqlite.ErrNotFound

// Record is the concrete SQLite row representation used by the SQLite facade.
type Record map[string]any

// Database is Axle's public SQLite-only CRUD facade.
//
// It intentionally exposes no driver, dialect, repository, query builder, or ORM
// abstraction: callers pass a generated Catalog and use descriptor resource names
// or paths to address CRUD operations.
type Database struct {
	db       *sql.DB
	catalog  axle.Catalog
	stores   map[string]internalsqlite.Store
	resource map[string]axle.ResourceDescriptor
}

// Open creates a SQLite CRUD facade from a generated catalog.
func Open(ctx context.Context, dsn string, catalog axle.Catalog) (*Database, error) {
	db, err := internalsqlite.Open(dsn)
	if err != nil {
		return nil, err
	}
	facade := &Database{
		db:       db,
		catalog:  catalog,
		stores:   map[string]internalsqlite.Store{},
		resource: map[string]axle.ResourceDescriptor{},
	}
	for _, registry := range catalog.Resources {
		res := registry.Resource
		store := internalsqlite.NewStore(db, res)
		for _, key := range resourceKeys(res) {
			normalized := normalizeResourceKey(key)
			facade.stores[normalized] = store
			facade.resource[normalized] = res
		}
	}
	if err := ctx.Err(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return facade, nil
}

// Close closes the underlying SQLite handle.
func (d *Database) Close() error { return d.db.Close() }

// SQLDB returns the concrete database/sql handle for tests and migrations that
// need SQLite-specific escape hatches. It is not a driver-neutral abstraction.
func (d *Database) SQLDB() *sql.DB { return d.db }

// Catalog returns the generated catalog used to construct this facade.
func (d *Database) Catalog() axle.Catalog { return d.catalog }

// Migrate creates all catalog tables if they do not exist.
func (d *Database) Migrate(ctx context.Context) error {
	seen := map[string]bool{}
	for _, registry := range d.catalog.Resources {
		key := normalizeResourceKey(registry.Resource.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		store := internalsqlite.NewStore(d.db, registry.Resource)
		if err := store.Migrate(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Resource returns descriptor metadata for a generated resource key.
func (d *Database) Resource(resource string) (axle.ResourceDescriptor, error) {
	res, ok := d.resource[normalizeResourceKey(resource)]
	if !ok {
		return axle.ResourceDescriptor{}, fmt.Errorf("unknown resource %q", resource)
	}
	return res, nil
}

// List returns records ordered by descriptor ID.
func (d *Database) List(ctx context.Context, resource string) ([]Record, error) {
	store, err := d.store(resource)
	if err != nil {
		return nil, err
	}
	rows, err := store.List(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, Record(row))
	}
	return records, nil
}

// Get returns one record by descriptor ID.
func (d *Database) Get(ctx context.Context, resource string, id any) (Record, error) {
	store, err := d.store(resource)
	if err != nil {
		return nil, err
	}
	row, err := store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return Record(row), nil
}

// Create inserts a record and returns the stored row when the descriptor ID is present.
func (d *Database) Create(ctx context.Context, resource string, values Record) (Record, error) {
	store, err := d.store(resource)
	if err != nil {
		return nil, err
	}
	if err := store.Create(ctx, map[string]any(values)); err != nil {
		return nil, err
	}
	res, err := d.Resource(resource)
	if err != nil {
		return nil, err
	}
	id, ok := values[res.ID]
	if !ok {
		return values, nil
	}
	return d.Get(ctx, resource, id)
}

// Update patches mutable fields and returns the stored row.
func (d *Database) Update(ctx context.Context, resource string, id any, values Record) (Record, error) {
	store, err := d.store(resource)
	if err != nil {
		return nil, err
	}
	if err := store.Update(ctx, id, map[string]any(values)); err != nil {
		return nil, err
	}
	return d.Get(ctx, resource, id)
}

// Delete removes one record by descriptor ID.
func (d *Database) Delete(ctx context.Context, resource string, id any) error {
	store, err := d.store(resource)
	if err != nil {
		return err
	}
	return store.Delete(ctx, id)
}

func (d *Database) store(resource string) (internalsqlite.Store, error) {
	store, ok := d.stores[normalizeResourceKey(resource)]
	if !ok {
		return internalsqlite.Store{}, fmt.Errorf("unknown resource %q", resource)
	}
	return store, nil
}

func resourceKeys(res axle.ResourceDescriptor) []string {
	return []string{res.Name, res.Path, "/" + strings.Trim(res.Path, "/")}
}

func normalizeResourceKey(resource string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(resource), "/"))
}

// IsNotFound reports whether err is Axle's concrete SQLite not-found error.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
