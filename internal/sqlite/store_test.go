package sqlite_test

import (
	"context"
	"testing"

	"github.com/Fel1xKan/axle/internal/descriptor"
	"github.com/Fel1xKan/axle/internal/sqlite"
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
