package sqlite

import (
	"database/sql"
	"errors"

	_ "modernc.org/sqlite"

	"github.com/cosmo-wise/axle/pkg/axle"
)

var ErrNotFound = errors.New("record not found")
var ErrUnknownField = errors.New("unknown descriptor field")
var ErrImmutableField = errors.New("immutable descriptor field")

func Open(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite", dsn)
}

type Store struct {
	db       *sql.DB
	resource axle.ResourceDescriptor
}

func NewStore(db *sql.DB, resource axle.ResourceDescriptor) Store {
	return Store{db: db, resource: resource}
}
