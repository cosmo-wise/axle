package codegen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cosmo-wise/axle/internal/codegen"
	"github.com/cosmo-wise/axle/internal/descriptor"
	"github.com/cosmo-wise/axle/pkg/axle"
)

func TestRoutesPreserveSemanticsAndNestedParams(t *testing.T) {
	desc, diagnostics := descriptor.Load("../../testdata/fixtures/nested/descriptor.axle.json")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	routes := codegen.BuildRoutes(desc)
	var foundRename, foundPolicy bool
	for _, route := range routes {
		if route.Name == "DeleteResource" && (route.TransportMethod != "POST" || route.Kind != "delete") {
			t.Fatalf("delete semantics not preserved: %#v", route)
		}
		if route.Path == "/resources/{id}/rename" && route.TransportMethod == "POST" {
			foundRename = true
		}
		if route.Path == "/resources/{id}/policy/{policy_id}/upgrade" {
			foundPolicy = true
			if strings.Join(route.Params, ",") != "id,policy_id" {
				t.Fatalf("unexpected nested params: %#v", route.Params)
			}
		}
	}
	if !foundRename || !foundPolicy {
		t.Fatalf("missing nested action routes: %#v", routes)
	}
}

func TestGenerateMatchesGoldenAndDetectsStale(t *testing.T) {
	desc, diagnostics := descriptor.Load("../../testdata/fixtures/single/descriptor.axle.json")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	files, genDiagnostics := codegen.Generate(desc)
	if len(genDiagnostics) != 0 {
		t.Fatalf("unexpected gen diagnostics: %#v", genDiagnostics)
	}
	var sawTypes, sawCatalog bool
	for _, file := range files {
		if file.Path == "types.gen.go" && strings.Contains(file.Content, "type Resource struct") {
			sawTypes = true
		}
		if file.Path == "registry.gen.go" && strings.Contains(file.Content, "var Catalog = axle.Catalog") {
			sawCatalog = true
		}
	}
	if !sawTypes || !sawCatalog {
		t.Fatalf("generated typed edge/catalog missing: types=%t catalog=%t files=%#v", sawTypes, sawCatalog, files)
	}
	if diags := codegen.Check("../../testdata/fixtures/single/generated", files); len(diags) != 0 {
		t.Fatalf("golden output is stale: %#v", diags)
	}
	tmp := t.TempDir()
	if err := codegen.Write(tmp, files); err != nil {
		t.Fatal(err)
	}
	if diags := codegen.Check(tmp, files); len(diags) != 0 {
		t.Fatalf("fresh output reported stale: %#v", diags)
	}
	path := filepath.Join(tmp, "routes.gen.go")
	if err := os.WriteFile(path, []byte("// stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diags := codegen.Check(tmp, files)
	if len(diags) != 1 || diags[0].Code != "AXLE_GENERATED_STALE" {
		t.Fatalf("expected stale diagnostic, got %#v", diags)
	}
}

func TestGenerateCatalogCombinesResourcesDeterministically(t *testing.T) {
	files, diagnostics := codegen.GenerateCatalog(codegen.CatalogDescriptor{
		Package: "catalog",
		Resources: []codegen.CatalogResource{
			{Alias: "resources", ImportPath: "example.com/app/descriptors/resources/generated"},
			{Alias: "policies", ImportPath: "example.com/app/descriptors/policies/generated"},
		},
	})
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(files) != 1 || files[0].Path != "catalog.gen.go" {
		t.Fatalf("unexpected catalog files: %#v", files)
	}
	policyIndex := strings.Index(files[0].Content, "policies.ResourceRegistry")
	resourceIndex := strings.Index(files[0].Content, "resources.ResourceRegistry")
	if policyIndex < 0 || resourceIndex < 0 || policyIndex > resourceIndex {
		t.Fatalf("catalog order is not deterministic: %s", files[0].Content)
	}
	tmp := t.TempDir()
	if err := codegen.Write(tmp, files); err != nil {
		t.Fatal(err)
	}
	if diags := codegen.Check(tmp, files); len(diags) != 0 {
		t.Fatalf("fresh catalog reported stale: %#v", diags)
	}
}

func TestGeneratePreservesSchemaMetadataInRegistryAndMigration(t *testing.T) {
	notNullable := false
	desc := axle.Descriptor{
		Schema: descriptor.SchemaV1,
		Resource: axle.ResourceDescriptor{
			Name:  "Widget",
			Path:  "widgets",
			Table: "widgets",
			ID:    "id",
			Fields: []axle.FieldDescriptor{
				{Name: "id", Type: "text", Auto: "uuid"},
				{Name: "owner_id", Type: "text", Mutable: true, Nullable: &notNullable, Index: true, References: &axle.ReferenceDescriptor{Resource: "User", Table: "users", Field: "id", OnDelete: "cascade"}},
				{Name: "slug", Type: "text", Mutable: true, Unique: true, Default: "'new'"},
			},
			Operations: []axle.OperationDescriptor{
				{Name: "ListWidgets", Kind: "list", Request: "ListWidgetsRequest", Response: "ListWidgetsResponse", Policy: "list", Handler: "ListWidgets"},
				{Name: "GetWidget", Kind: "get", Request: "GetWidgetRequest", Response: "GetWidgetResponse", Policy: "get", Handler: "GetWidget"},
				{Name: "CreateWidget", Kind: "create", Request: "CreateWidgetRequest", Response: "CreateWidgetResponse", Policy: "create", Handler: "CreateWidget"},
				{Name: "UpdateWidget", Kind: "update", Request: "UpdateWidgetRequest", Response: "UpdateWidgetResponse", Policy: "update", Handler: "UpdateWidget"},
				{Name: "DeleteWidget", Kind: "delete", Request: "DeleteWidgetRequest", Response: "DeleteWidgetResponse", Policy: "delete", Handler: "DeleteWidget"},
			},
		},
		Generated: axle.GeneratedTarget{Package: "generated"},
	}
	files, diagnostics := codegen.Generate(desc)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	var registry string
	var migration string
	for _, file := range files {
		if file.Path == "registry.gen.go" {
			registry = file.Content
		}
		if strings.HasPrefix(file.Path, "migrations/") {
			migration = file.Content
		}
	}
	for _, want := range []string{"Nullable: boolPtr(false)", "Index: true", "Unique: true", "Auto: \"uuid\"", "Default: \"'new'\"", "References: &axle.ReferenceDescriptor{Resource: \"User\", Table: \"users\", Field: \"id\", OnDelete: \"cascade\"}"} {
		if !strings.Contains(registry, want) {
			t.Fatalf("registry missing %s in:\n%s", want, registry)
		}
	}
	for _, want := range []string{"CREATE TABLE IF NOT EXISTS \"widgets\"", "\"owner_id\" TEXT NOT NULL REFERENCES \"users\"(\"id\") ON DELETE CASCADE", "\"slug\" TEXT UNIQUE DEFAULT 'new'", "CREATE INDEX IF NOT EXISTS \"idx_widgets_owner_id\" ON \"widgets\" (\"owner_id\")"} {
		if !strings.Contains(migration, want) {
			t.Fatalf("migration missing %s in:\n%s", want, migration)
		}
	}
}
