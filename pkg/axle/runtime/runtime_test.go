package runtime_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Fel1xKan/axle/internal/codegen"
	"github.com/Fel1xKan/axle/internal/descriptor"
	"github.com/Fel1xKan/axle/pkg/axle"
	axleruntime "github.com/Fel1xKan/axle/pkg/axle/runtime"
	axlesqlite "github.com/Fel1xKan/axle/pkg/axle/sqlite"
)

func TestRuntimeMountsCatalogCRUDAndNestedAction(t *testing.T) {
	desc, diagnostics := descriptor.Load("../../../testdata/fixtures/nested/descriptor.axle.json")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	catalog := axle.Catalog{Resources: []axle.ResourceRegistry{{Resource: desc.Resource, Routes: codegen.BuildRoutes(desc)}}}
	ctx := context.Background()
	db, err := axlesqlite.Open(ctx, ":memory:", catalog)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	handler := axleruntime.New(catalog, db, axleruntime.ActionHandlers{
		"UpgradeResourcePolicy": func(ctx context.Context, request axleruntime.ActionRequest) (any, error) {
			return map[string]any{"data": map[string]any{"id": request.ID, "policy_id": request.Params["policy_id"], "upgraded": true}}, nil
		},
	})

	mustStatus(t, handler, http.MethodPost, "/resources", `{"id":"r1","name":"First","policy_id":"p1"}`, http.StatusOK)
	list := mustStatus(t, handler, http.MethodGet, "/resources", "", http.StatusOK)
	var listed struct{ Data []map[string]any }
	if err := json.Unmarshal(list, &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Data) != 1 || listed.Data[0]["name"] != "First" {
		t.Fatalf("unexpected list payload: %s", list)
	}
	mustStatus(t, handler, http.MethodPost, "/resources/r1/update", `{"name":"Renamed","policy_id":"p2"}`, http.StatusOK)
	payload := mustStatus(t, handler, http.MethodPost, "/resources/r1/policy/p2/upgrade", `{"reason":"paid"}`, http.StatusOK)
	var action struct{ Data map[string]any }
	if err := json.Unmarshal(payload, &action); err != nil {
		t.Fatal(err)
	}
	if action.Data["policy_id"] != "p2" || action.Data["upgraded"] != true {
		t.Fatalf("unexpected action payload: %s", payload)
	}
	mustStatus(t, handler, http.MethodPost, "/resources/r1/delete", "", http.StatusOK)
	mustStatus(t, handler, http.MethodGet, "/resources/r1", "", http.StatusNotFound)
}

func TestEdgeProvidesHealthRoutesPrefixAndCORS(t *testing.T) {
	desc, diagnostics := descriptor.Load("../../../testdata/fixtures/nested/descriptor.axle.json")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	catalog := axle.Catalog{Resources: []axle.ResourceRegistry{{Resource: desc.Resource, Routes: codegen.BuildRoutes(desc)}}}
	ctx := context.Background()
	db, err := axlesqlite.Open(ctx, ":memory:", catalog)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	handler := axleruntime.NewEdge(catalog, db, axleruntime.ActionHandlers{}, axleruntime.EdgeOptions{
		Name:      "Example Edge",
		APIPrefix: "/api/v1",
		CORS:      true,
		Health:    map[string]any{"backend": "example"},
	})

	health := mustStatus(t, handler, http.MethodGet, "/healthz", "", http.StatusOK)
	var healthPayload map[string]any
	if err := json.Unmarshal(health, &healthPayload); err != nil {
		t.Fatal(err)
	}
	if healthPayload["backend"] != "example" || healthPayload["framework"] != "axle" {
		t.Fatalf("unexpected health payload: %s", health)
	}

	routes := mustStatus(t, handler, http.MethodGet, "/routes", "", http.StatusOK)
	var routesPayload struct{ Count int }
	if err := json.Unmarshal(routes, &routesPayload); err != nil {
		t.Fatal(err)
	}
	if routesPayload.Count == 0 {
		t.Fatalf("expected route catalog, got %s", routes)
	}

	mustStatus(t, handler, http.MethodPost, "/api/v1/resources", `{"id":"r1","name":"First","policy_id":"p1"}`, http.StatusOK)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/api/v1/resources", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("unexpected CORS methods %q", got)
	}
}

func mustStatus(t *testing.T, handler http.Handler, method string, path string, body string, want int) []byte {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, rec.Code, want, rec.Body.String())
	}
	return rec.Body.Bytes()
}
