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

func TestValidateRejectsUnsafeIdentifiersAndUnknownFieldMetadata(t *testing.T) {
	desc := axle.Descriptor{
		Schema: descriptor.SchemaV1,
		Resource: axle.ResourceDescriptor{
			Name:  "Broken",
			Path:  "broken",
			Table: "broken;drop",
			ID:    "id",
			Fields: []axle.FieldDescriptor{
				{Name: "id", Type: "text"},
				{Name: "bad-name", Type: "json", Auto: "slug"},
				{Name: "owner_id", Type: "text", References: &axle.ReferenceDescriptor{Table: "users;drop", Field: "id"}},
			},
			Operations: validOps(),
		},
		Generated: axle.GeneratedTarget{Package: "generated"},
	}
	diagnostics := descriptor.Validate(desc, "broken.json")
	codes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	for _, want := range []string{"AXLE_RESOURCE_TABLE_IDENTIFIER", "AXLE_FIELD_IDENTIFIER", "AXLE_FIELD_TYPE", "AXLE_FIELD_AUTO", "AXLE_FIELD_REFERENCE_TABLE"} {
		if !slices.Contains(codes, want) {
			t.Fatalf("missing diagnostic %s in %#v", want, codes)
		}
	}
}

func validOps() []axle.OperationDescriptor {
	return []axle.OperationDescriptor{
		{Name: "ListBroken", Kind: "list", Request: "ListBrokenRequest", Response: "ListBrokenResponse", Policy: "list", Handler: "ListBroken"},
		{Name: "GetBroken", Kind: "get", Request: "GetBrokenRequest", Response: "GetBrokenResponse", Policy: "get", Handler: "GetBroken"},
		{Name: "CreateBroken", Kind: "create", Request: "CreateBrokenRequest", Response: "CreateBrokenResponse", Policy: "create", Handler: "CreateBroken"},
		{Name: "UpdateBroken", Kind: "update", Request: "UpdateBrokenRequest", Response: "UpdateBrokenResponse", Policy: "update", Handler: "UpdateBroken"},
		{Name: "DeleteBroken", Kind: "delete", Request: "DeleteBrokenRequest", Response: "DeleteBrokenResponse", Policy: "delete", Handler: "DeleteBroken"},
	}
}
