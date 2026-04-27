package descriptor_test

import (
	"slices"
	"testing"

	"github.com/Fel1xKan/axle/internal/descriptor"
	"github.com/Fel1xKan/axle/pkg/axle"
)

func TestValidateRejectsMissingSchemaAndBindings(t *testing.T) {
	desc := axle.Descriptor{Resource: axle.ResourceDescriptor{Name: "Broken", Path: "broken", Table: "broken", ID: "id", Fields: []axle.FieldDescriptor{{Name: "id", Type: "text"}}, Operations: []axle.OperationDescriptor{{Name: "ListBroken", Kind: "list"}}}, Generated: axle.GeneratedTarget{Package: "generated"}}
	diagnostics := descriptor.Validate(desc, "broken.json")
	codes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	for _, want := range []string{"AXLE_DESCRIPTOR_SCHEMA", "AXLE_OPERATION_REQUEST", "AXLE_OPERATION_RESPONSE", "AXLE_OPERATION_POLICY", "AXLE_OPERATION_HANDLER"} {
		if !slices.Contains(codes, want) {
			t.Fatalf("missing diagnostic %s in %#v", want, codes)
		}
	}
}

func TestLoadValidFixture(t *testing.T) {
	desc, diagnostics := descriptor.Load("../../testdata/fixtures/single/descriptor.axle.json")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if desc.Resource.Path != "resources" || desc.Resource.ID != "id" {
		t.Fatalf("unexpected descriptor: %#v", desc.Resource)
	}
}
