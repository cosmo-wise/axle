package openapi_test

import (
	"strings"
	"testing"

	"github.com/Fel1xKan/axle/internal/codegen"
	"github.com/Fel1xKan/axle/internal/descriptor"
	"github.com/Fel1xKan/axle/internal/openapi"
)

func TestOpenAPIExtensionsPreserveSemantics(t *testing.T) {
	desc, diagnostics := descriptor.Load("../../testdata/fixtures/nested/descriptor.axle.json")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	doc, err := openapi.Build(desc, codegen.BuildRoutes(desc))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"\"x-axle-kind\": \"delete\"", "\"x-transport-method\": \"POST\"", "\"x-axle-action\": \"UpgradeResourcePolicy\"", "/resources/{id}/policy/{policy_id}/upgrade", "\"components\"", "\"$ref\": \"#/components/schemas/Resource\"", "\"requestBody\"", "\"parameters\""} {
		if !strings.Contains(doc, want) {
			t.Fatalf("OpenAPI missing %s in:\n%s", want, doc)
		}
	}
}
