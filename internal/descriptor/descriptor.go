package descriptor

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cosmo-wise/axle/pkg/axle"
)

const SchemaV1 = "axle.resource.v1"

var identRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// Load reads a JSON Axle descriptor from disk.
func Load(path string) (axle.Descriptor, []axle.Diagnostic) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return axle.Descriptor{}, []axle.Diagnostic{{Code: "AXLE_DESCRIPTOR_READ", Path: path, Message: err.Error(), SuggestedFix: "Make the descriptor path readable."}}
	}
	var desc axle.Descriptor
	if err := json.Unmarshal(payload, &desc); err != nil {
		return axle.Descriptor{}, []axle.Diagnostic{{Code: "AXLE_DESCRIPTOR_JSON", Path: path, Message: err.Error(), SuggestedFix: "Fix descriptor JSON syntax."}}
	}
	return desc, Validate(desc, path)
}

// Validate returns deterministic descriptor diagnostics.
func Validate(desc axle.Descriptor, sourcePath string) []axle.Diagnostic {
	var diagnostics []axle.Diagnostic
	add := func(code, path, message, fix string) {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: code, Path: path, Message: message, SuggestedFix: fix})
	}
	if desc.Schema != SchemaV1 {
		add("AXLE_DESCRIPTOR_SCHEMA", sourcePath+"#/schema", fmt.Sprintf("schema must be %q", SchemaV1), "Set schema to axle.resource.v1.")
	}
	res := desc.Resource
	if strings.TrimSpace(res.Name) == "" || !identRE.MatchString(res.Name) {
		add("AXLE_RESOURCE_NAME", sourcePath+"#/resource/name", "resource name must be a Go-style identifier", "Use a stable singular identifier such as Resource.")
	}
	if strings.TrimSpace(res.Path) == "" {
		add("AXLE_RESOURCE_PATH", sourcePath+"#/resource/path", "resource path segment is required", "Set resource.path to a plural URL segment.")
	}
	if strings.TrimSpace(res.Table) == "" {
		add("AXLE_RESOURCE_TABLE", sourcePath+"#/resource/table", "resource table is required", "Set resource.table to the SQLite table name.")
	} else if !identRE.MatchString(res.Table) {
		add("AXLE_RESOURCE_TABLE_IDENTIFIER", sourcePath+"#/resource/table", "resource table must be a safe SQLite identifier", "Use letters, numbers, and underscores, starting with a letter.")
	}
	if strings.TrimSpace(res.ID) == "" {
		add("AXLE_RESOURCE_ID", sourcePath+"#/resource/id", "resource id field is required", "Set resource.id to the primary key field name.")
	}
	if len(res.Fields) == 0 {
		add("AXLE_RESOURCE_FIELDS", sourcePath+"#/resource/fields", "at least one field is required", "Add persisted fields for the resource.")
	}
	fieldNames := map[string]bool{}
	for i, field := range res.Fields {
		path := fmt.Sprintf("%s#/resource/fields/%d", sourcePath, i)
		if strings.TrimSpace(field.Name) == "" {
			add("AXLE_FIELD_NAME", path+"/name", "field name is required", "Give every field a stable name.")
		} else if !identRE.MatchString(field.Name) {
			add("AXLE_FIELD_IDENTIFIER", path+"/name", "field name must be a safe SQLite identifier", "Use letters, numbers, and underscores, starting with a letter.")
		}
		switch strings.TrimSpace(field.Type) {
		case "text", "integer", "boolean", "real":
		case "":
			add("AXLE_FIELD_TYPE", path+"/type", "field type is required", "Use one of text, integer, boolean, or real.")
		default:
			add("AXLE_FIELD_TYPE", path+"/type", "field type is unsupported", "Use one of text, integer, boolean, or real.")
		}
		if strings.TrimSpace(field.Auto) != "" && field.Auto != "uuid" {
			add("AXLE_FIELD_AUTO", path+"/auto", "field auto value is unsupported", "Use auto: uuid or omit auto generation.")
		}
		if field.References != nil {
			if strings.TrimSpace(field.References.Table) == "" || !identRE.MatchString(field.References.Table) {
				add("AXLE_FIELD_REFERENCE_TABLE", path+"/references/table", "reference table must be a safe SQLite identifier", "Set references.table to the target table name.")
			}
			if strings.TrimSpace(field.References.Field) != "" && !identRE.MatchString(field.References.Field) {
				add("AXLE_FIELD_REFERENCE_FIELD", path+"/references/field", "reference field must be a safe SQLite identifier", "Use a target field such as id.")
			}
		}
		if fieldNames[field.Name] {
			add("AXLE_FIELD_DUPLICATE", path+"/name", "field name is duplicated", "Use unique persisted field names.")
		}
		fieldNames[field.Name] = true
	}
	if res.ID != "" && !fieldNames[res.ID] {
		add("AXLE_RESOURCE_ID_FIELD", sourcePath+"#/resource/id", "id field must also appear in resource.fields", "Add the ID field to resource.fields.")
	}
	if strings.TrimSpace(desc.Generated.Package) == "" {
		add("AXLE_GENERATED_PACKAGE", sourcePath+"#/generated/package", "generated package is required", "Set generated.package, for example generated.")
	}
	seenKinds := map[string]bool{}
	for i, op := range res.Operations {
		validateOperation(op, fmt.Sprintf("%s#/resource/operations/%d", sourcePath, i), add)
		seenKinds[op.Kind] = true
	}
	for _, kind := range []string{"list", "get", "create", "update", "delete"} {
		if !seenKinds[kind] {
			add("AXLE_OPERATION_REQUIRED", sourcePath+"#/resource/operations", "missing CRUD operation kind "+kind, "Declare all CRUD operation kinds: list, get, create, update, delete.")
		}
	}
	for i, op := range res.Actions {
		validateOperation(op, fmt.Sprintf("%s#/resource/actions/%d", sourcePath, i), add)
		if strings.TrimSpace(op.Path) == "" {
			add("AXLE_ACTION_PATH", fmt.Sprintf("%s#/resource/actions/%d/path", sourcePath, i), "action path is required", "Set a relative action path such as rename or policy/{policy_id}/upgrade.")
		}
	}
	return diagnostics
}

func validateOperation(op axle.OperationDescriptor, path string, add func(string, string, string, string)) {
	if strings.TrimSpace(op.Name) == "" {
		add("AXLE_OPERATION_NAME", path+"/name", "operation name is required", "Add a stable operation name.")
	}
	if strings.TrimSpace(op.Kind) == "" {
		add("AXLE_OPERATION_KIND", path+"/kind", "operation kind is required", "Set operation kind to list/get/create/update/delete/action.")
	}
	if strings.TrimSpace(op.Request) == "" {
		add("AXLE_OPERATION_REQUEST", path+"/request", "operation request binding is required", "Bind this operation to an explicit request type.")
	}
	if strings.TrimSpace(op.Response) == "" {
		add("AXLE_OPERATION_RESPONSE", path+"/response", "operation response binding is required", "Bind this operation to an explicit response type.")
	}
	if strings.TrimSpace(op.Policy) == "" {
		add("AXLE_OPERATION_POLICY", path+"/policy", "operation policy binding is required", "Bind this operation to an explicit policy.")
	}
	if strings.TrimSpace(op.Handler) == "" {
		add("AXLE_OPERATION_HANDLER", path+"/handler", "operation handler binding is required", "Bind this operation to a service/handler function.")
	}
}
