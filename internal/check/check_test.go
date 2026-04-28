package check_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Fel1xKan/axle/internal/check"
)

func TestCheckPositiveFixture(t *testing.T) {
	result := check.Run("../../testdata/fixtures/single/descriptor.axle.json", "../..")
	if result.Status != "ok" {
		t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics)
	}
}

func TestCheckNegativeFixtures(t *testing.T) {
	cases := []struct {
		name       string
		descriptor string
		root       string
		want       string
	}{
		{"controller db", "", "../../testdata/fixtures/negative/controller-db", "AXLE_BOUNDARY_CONTROLLER_DB"},
		{"service http", "", "../../testdata/fixtures/negative/service-http", "AXLE_BOUNDARY_SERVICE_HTTP"},
		{"missing bindings", "../../testdata/fixtures/negative/missing-bindings/descriptor.axle.json", "", "AXLE_OPERATION_REQUEST"},
		{"public import", "", "../../testdata/fixtures/negative/public-import", "AXLE_PUBLIC_IMPORT_INTERNAL"},
		{"multi db", "", "../../testdata/fixtures/negative/multidb", "AXLE_MULTIDB_ABSTRACTION"},
		{"reflection", "", "../../testdata/fixtures/negative/reflection", "AXLE_RUNTIME_DISCOVERY"},
		{"public api bloat", "", "../../testdata/fixtures/negative/public-api-bloat", "AXLE_PUBLIC_API_BLOAT"},
		{"public api record", "", "../../testdata/fixtures/negative/public-api-record", "AXLE_PUBLIC_API_BLOAT"},
		{"manual crud routing", "", "../../testdata/fixtures/negative/manual-crud-routing", "AXLE_MANUAL_CRUD_ROUTING"},
		{"manual crud routing task", "", "../../testdata/fixtures/negative/manual-crud-routing-task", "AXLE_MANUAL_CRUD_ROUTING"},
		{"typed orm", "", "../../testdata/fixtures/negative/typed-orm", "AXLE_TYPED_ORM_CREEP"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := check.Run(tc.descriptor, tc.root)
			codes := make([]string, 0, len(result.Diagnostics))
			for _, diagnostic := range result.Diagnostics {
				codes = append(codes, diagnostic.Code)
			}
			if !slices.Contains(codes, tc.want) {
				t.Fatalf("missing %s in %#v", tc.want, result.Diagnostics)
			}
		})
	}
}

func TestManualCRUDAllowsAppOwnedEdgeWrapper(t *testing.T) {
	root := t.TempDir()
	server := `package edge

import "net/http"

const apiPrefix = "/api/v1"

func Serve(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/healthz", "/routes", apiPrefix:
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
`
	if err := os.WriteFile(filepath.Join(root, "server.go"), []byte(server), 0o644); err != nil {
		t.Fatal(err)
	}
	result := check.Run("", root)
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "AXLE_MANUAL_CRUD_ROUTING" {
			t.Fatalf("edge wrapper should not be manual CRUD: %#v", result.Diagnostics)
		}
	}
}

func TestCheckDetectsStaleGeneratedCatalog(t *testing.T) {
	root := t.TempDir()
	catalogDir := filepath.Join(root, "catalog")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "package": "catalog",
  "resources": [
    {"alias": "resources", "import": "example.com/app/descriptors/resources/generated"},
    {"alias": "policies", "import": "example.com/app/descriptors/policies/generated"}
  ]
}
`
	if err := os.WriteFile(filepath.Join(catalogDir, "axle.catalog.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catalogDir, "catalog.gen.go"), []byte("// stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := check.Run("", root)
	codes := make([]string, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	if !slices.Contains(codes, "AXLE_GENERATED_STALE") {
		t.Fatalf("expected stale catalog diagnostic, got %#v", result.Diagnostics)
	}
}
