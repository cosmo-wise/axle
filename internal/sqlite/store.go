package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/cosmo-wise/axle/pkg/axle"
)

var ErrNotFound = errors.New("record not found")
var ErrUnknownField = errors.New("unknown descriptor field")
var ErrImmutableField = errors.New("immutable descriptor field")

func Open(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite", dsn)
}

// OpenDatabaseHandles returns a regular application handle plus a handle whose
// transactions use BEGIN IMMEDIATE exclusively for migration execution.
func OpenDatabaseHandles(dsn string) (application, migrations *sql.DB, err error) {
	normalized, err := immediateTransactionDSN(dsn)
	if err != nil {
		return nil, nil, err
	}
	if connectionLocalDSN(dsn) {
		db, err := sql.Open("sqlite", normalized)
		return db, db, err
	}
	application, err = sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, err
	}
	migrations, err = sql.Open("sqlite", normalized)
	if err != nil {
		_ = application.Close()
		return nil, nil, err
	}
	return application, migrations, nil
}

// Axle migration callbacks receive *sql.Tx, so a migration-only driver handle
// is the only way to guarantee BEGIN IMMEDIATE without replacing database/sql's
// transaction type or changing application transaction semantics.
func immediateTransactionDSN(dsn string) (string, error) {
	base, rawQuery, _ := strings.Cut(dsn, "?")
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", fmt.Errorf("parse SQLite DSN query: %w", err)
	}
	query.Set("_txlock", "immediate")
	return base + "?" + query.Encode(), nil
}

func connectionLocalDSN(dsn string) bool {
	base, rawQuery, _ := strings.Cut(dsn, "?")
	if base == "" || base == ":memory:" {
		return true
	}
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return true
	}
	isMemory := strings.Contains(base, ":memory:") || query.Get("mode") == "memory"
	return isMemory && query.Get("cache") != "shared"
}

type Store struct {
	db       *sql.DB
	resource axle.ResourceDescriptor
}

func NewStore(db *sql.DB, resource axle.ResourceDescriptor) Store {
	return Store{db: db, resource: resource}
}
