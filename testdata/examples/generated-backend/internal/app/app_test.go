package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appcatalog "github.com/Fel1xKan/axle/testdata/examples/generated-backend/catalog"
	"github.com/Fel1xKan/axle/testdata/examples/generated-backend/internal/app"
	axlesqlite "github.com/Fel1xKan/axle/pkg/axle/sqlite"
)

func TestGeneratedBackendEdgeCRUDAndNestedAction(t *testing.T) {
	handler := newTestHandler(t)

	mustStatus(t, handler, http.MethodGet, "/healthz", "", http.StatusOK)
	routes := mustStatus(t, handler, http.MethodGet, "/routes", "", http.StatusOK)
	var routePayload struct{ Count int }
	if err := json.Unmarshal(routes, &routePayload); err != nil {
		t.Fatal(err)
	}
	if routePayload.Count < 10 {
		t.Fatalf("expected generated routes, got %d", routePayload.Count)
	}

	mustStatus(t, handler, http.MethodPost, "/policies", `{"id":"p1","name":"Gold","level":1}`, http.StatusOK)
	mustStatus(t, handler, http.MethodGet, "/api/v1/policies/p1", "", http.StatusOK)
	mustStatus(t, handler, http.MethodPost, "/resources", `{"id":"r1","name":"First","policy_id":"p1"}`, http.StatusOK)
	mustStatus(t, handler, http.MethodGet, "/resources", "", http.StatusOK)
	mustStatus(t, handler, http.MethodPost, "/resources/r1/update", `{"name":"Renamed","policy_id":"p1"}`, http.StatusOK)
	payload := mustStatus(t, handler, http.MethodPost, "/resources/r1/policy/p1/upgrade", `{"reason":"paid"}`, http.StatusOK)
	var action struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(payload, &action); err != nil {
		t.Fatal(err)
	}
	if action.Data["policy_id"] != "p1" || action.Data["upgraded"] != true {
		t.Fatalf("unexpected action payload: %s", payload)
	}
	mustStatus(t, handler, http.MethodPost, "/resources/r1/delete", "", http.StatusOK)
	mustStatus(t, handler, http.MethodGet, "/resources/r1", "", http.StatusNotFound)
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	ctx := context.Background()
	db, err := axlesqlite.Open(ctx, ":memory:", appcatalog.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return app.New(db)
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
