package codegen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fel1xKan/axle/internal/codegen"
	"github.com/Fel1xKan/axle/internal/descriptor"
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
