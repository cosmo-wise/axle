package codegen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/cosmo-wise/axle/pkg/axle"
)

func renderTypes(packageName string, desc axle.Descriptor) string {
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("package " + packageName + "\n")
	if len(desc.Resource.Actions) > 0 {
		b.WriteString("\nimport (\n")
		b.WriteString("\t\"context\"\n\n")
		b.WriteString("\taxleruntime \"github.com/cosmo-wise/axle/pkg/axle/runtime\"\n")
		b.WriteString(")\n")
	}
	b.WriteString("\n")
	resourceType := goIdent(desc.Resource.Name)
	b.WriteString("type " + resourceType + " struct {\n")
	for _, field := range desc.Resource.Fields {
		b.WriteString(fmt.Sprintf("\t%s %s `json:%q`\n", goIdent(field.Name), goType(field.Type), field.Name))
	}
	b.WriteString("}\n\n")
	for _, op := range desc.Resource.Operations {
		renderCRUDTypes(&b, resourceType, desc.Resource.ID, op)
	}
	for _, op := range desc.Resource.Actions {
		renderActionTypes(&b, op)
	}
	return b.String()
}

func renderCRUDTypes(b *strings.Builder, resourceType string, idField string, op axle.OperationDescriptor) {
	switch op.Kind {
	case "list":
		b.WriteString("type " + op.Request + " struct{}\n\n")
		b.WriteString("type " + op.Response + " struct {\n")
		b.WriteString("\tData []" + resourceType + " `json:\"data\"`\n")
		b.WriteString("}\n\n")
	case "get":
		writeIDRequest(b, op.Request, idField)
		writeDataResponse(b, op.Response, resourceType)
	case "create":
		b.WriteString("type " + op.Request + " struct {\n")
		b.WriteString("\tData " + resourceType + " `json:\"data\"`\n")
		b.WriteString("}\n\n")
		writeDataResponse(b, op.Response, resourceType)
	case "update":
		b.WriteString("type " + op.Request + " struct {\n")
		b.WriteString("\t" + goIdent(idField) + " string `json:\"" + idField + "\"`\n")
		b.WriteString("\tData " + resourceType + " `json:\"data\"`\n")
		b.WriteString("}\n\n")
		writeDataResponse(b, op.Response, resourceType)
	case "delete":
		writeIDRequest(b, op.Request, idField)
		b.WriteString("type " + op.Response + " struct {\n")
		b.WriteString("\tDeleted bool `json:\"deleted\"`\n")
		b.WriteString("\t" + goIdent(idField) + " string `json:\"" + idField + "\"`\n")
		b.WriteString("}\n\n")
	default:
		b.WriteString("type " + op.Request + " struct{}\n\n")
		b.WriteString("type " + op.Response + " struct{}\n\n")
	}
}

func writeIDRequest(b *strings.Builder, name string, idField string) {
	b.WriteString("type " + name + " struct {\n")
	b.WriteString("\t" + goIdent(idField) + " string `json:\"" + idField + "\"`\n")
	b.WriteString("}\n\n")
}

func writeDataResponse(b *strings.Builder, name string, resourceType string) {
	b.WriteString("type " + name + " struct {\n")
	b.WriteString("\tData " + resourceType + " `json:\"data\"`\n")
	b.WriteString("}\n\n")
}

func renderActionTypes(b *strings.Builder, op axle.OperationDescriptor) {
	b.WriteString(fmt.Sprintf("const Handler%s = %q\n\n", op.Name, op.Handler))
	b.WriteString("type " + op.Request + " struct {\n")
	b.WriteString("\tID string `json:\"id,omitempty\"`\n")
	b.WriteString("\tParams map[string]string `json:\"params,omitempty\"`\n")
	b.WriteString("\tBody map[string]any `json:\"body,omitempty\"`\n")
	b.WriteString("}\n\n")
	b.WriteString("type " + op.Response + " struct {\n")
	b.WriteString("\tData map[string]any `json:\"data,omitempty\"`\n")
	b.WriteString("}\n\n")
	b.WriteString("func Bind" + op.Name + "(handler func(context.Context, " + op.Request + ") (" + op.Response + ", error)) axleruntime.ActionHandler {\n")
	b.WriteString("\treturn func(ctx context.Context, request axleruntime.ActionRequest) (any, error) {\n")
	b.WriteString("\t\treturn handler(ctx, " + op.Request + "{ID: request.ID, Params: request.Params, Body: request.Body})\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
}

func goType(kind string) string {
	switch kind {
	case "integer":
		return "int"
	case "boolean":
		return "bool"
	case "real":
		return "float64"
	default:
		return "string"
	}
}

func goIdent(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '_' || r == '-' || r == ' ' || r == '/' })
	if len(parts) == 0 {
		return "Value"
	}
	for i, part := range parts {
		lower := strings.ToLower(part)
		if lower == "id" {
			parts[i] = "ID"
			continue
		}
		runes := []rune(lower)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}
