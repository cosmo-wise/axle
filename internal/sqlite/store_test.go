package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cosmo-wise/axle/internal/descriptor"
	"github.com/cosmo-wise/axle/internal/sqlite"
	"github.com/cosmo-wise/axle/pkg/axle"
)

func TestStoreCRUD(t *testing.T) {
	ctx := context.Background()
	desc, diagnostics := descriptor.Load("../../testdata/fixtures/single/descriptor.axle.json")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := sqlite.NewStore(db, desc.Resource)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, map[string]any{"id": "r1", "name": "First", "status": "new"}); err != nil {
		t.Fatal(err)
	}
	items, err := store.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("list failed items=%#v err=%v", items, err)
	}
	item, err := store.Get(ctx, "r1")
	if err != nil || item["name"] != "First" {
		t.Fatalf("get failed item=%#v err=%v", item, err)
	}
	if err := store.Update(ctx, "r1", map[string]any{"name": "Renamed"}); err != nil {
		t.Fatal(err)
	}
	item, err = store.Get(ctx, "r1")
	if err != nil || item["name"] != "Renamed" {
		t.Fatalf("update failed item=%#v err=%v", item, err)
	}
	if err := store.Delete(ctx, "r1"); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(ctx)
	if err != nil || len(items) != 0 {
		t.Fatalf("delete failed items=%#v err=%v", items, err)
	}
}

func TestStoreRejectsUnknownAndImmutableFieldsAndAutoUUID(t *testing.T) {
	ctx := context.Background()
	resource := secureTestResource()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := sqlite.NewStore(db, resource)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	values := map[string]any{"name": "First", "status": "new"}
	if err := store.Create(ctx, values); err != nil {
		t.Fatal(err)
	}
	if values["id"] == "" {
		t.Fatalf("auto uuid field was not populated: %#v", values)
	}
	if err := store.Create(ctx, map[string]any{"id": "bad", "name); DROP TABLE resources; --": "x"}); !errors.Is(err, sqlite.ErrUnknownField) {
		t.Fatalf("expected unknown field rejection for injected field, got %v", err)
	}
	if err := store.Update(ctx, values["id"], map[string]any{"id": "changed"}); !errors.Is(err, sqlite.ErrImmutableField) {
		t.Fatalf("expected immutable field rejection, got %v", err)
	}
	if err := store.Update(ctx, values["id"], map[string]any{"unknown": "changed"}); !errors.Is(err, sqlite.ErrUnknownField) {
		t.Fatalf("expected unknown field rejection, got %v", err)
	}
}

func TestStoreMigrationAddsColumnsAndIndexes(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	initial := accountsResource(false)
	if err := sqlite.NewStore(db, initial).Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	upgraded := accountsResource(true)
	store := sqlite.NewStore(db, upgraded)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	columns := map[string]bool{}
	rows, err := db.QueryContext(ctx, `PRAGMA table_info("accounts")`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		columns[name] = true
	}
	if !columns["status"] {
		t.Fatalf("migration did not add status column: %#v", columns)
	}
	var indexName string
	if err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'index' AND name = 'idx_accounts_email'`).Scan(&indexName); err != nil {
		t.Fatalf("expected deterministic email index: %v", err)
	}
	if err := store.Create(ctx, map[string]any{"id": "a1", "email": "a@example.test", "status": "new"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, map[string]any{"id": "a2", "email": "a@example.test", "status": "new"}); err == nil {
		t.Fatalf("expected unique email constraint to reject duplicate")
	}
}

func secureTestResource() axle.ResourceDescriptor {
	return axle.ResourceDescriptor{
		Name:  "Resource",
		Path:  "resources",
		Table: "resources",
		ID:    "id",
		Fields: []axle.FieldDescriptor{
			{Name: "id", Type: "text", Auto: "uuid"},
			{Name: "name", Type: "text", Mutable: true},
			{Name: "status", Type: "text", Mutable: true},
		},
	}
}

func accountsResource(includeStatus bool) axle.ResourceDescriptor {
	notNullable := false
	fields := []axle.FieldDescriptor{
		{Name: "id", Type: "text"},
		{Name: "email", Type: "text", Mutable: true, Nullable: &notNullable, Unique: true, Index: true},
	}
	if includeStatus {
		fields = append(fields, axle.FieldDescriptor{Name: "status", Type: "text", Mutable: true, Nullable: &notNullable, Default: "'new'"})
	}
	return axle.ResourceDescriptor{Name: "Account", Path: "accounts", Table: "accounts", ID: "id", Fields: fields}
}
