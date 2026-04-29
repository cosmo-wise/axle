package openapi

import (
	"encoding/json"

	"github.com/Fel1xKan/axle/pkg/axle"
)

type document struct {
	OpenAPI    string                   `json:"openapi"`
	Info       info                     `json:"info"`
	Paths      map[string]map[string]op `json:"paths"`
	Components components               `json:"components"`
}

type info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type components struct {
	Schemas map[string]schemaObject `json:"schemas"`
}

type schemaObject struct {
	Ref        string                  `json:"$ref,omitempty"`
	Type       string                  `json:"type,omitempty"`
	Properties map[string]schemaObject `json:"properties,omitempty"`
	Items      *schemaObject           `json:"items,omitempty"`
	Required   []string                `json:"required,omitempty"`
	Format     string                  `json:"format,omitempty"`
	Nullable   bool                    `json:"nullable,omitempty"`
}

type op struct {
	OperationID      string              `json:"operationId"`
	Parameters       []parameter         `json:"parameters,omitempty"`
	RequestBody      *requestBody        `json:"requestBody,omitempty"`
	Responses        map[string]response `json:"responses"`
	AxleKind         string              `json:"x-axle-kind"`
	TransportMethod  string              `json:"x-transport-method"`
	AxleResource     string              `json:"x-axle-resource"`
	AxleAction       string              `json:"x-axle-action,omitempty"`
	AxlePathParams   []string            `json:"x-axle-path-params,omitempty"`
	AxleRequestType  string              `json:"x-axle-request"`
	AxleResponseType string              `json:"x-axle-response"`
}

type parameter struct {
	Name     string       `json:"name"`
	In       string       `json:"in"`
	Required bool         `json:"required"`
	Schema   schemaObject `json:"schema"`
}

type requestBody struct {
	Required bool                 `json:"required"`
	Content  map[string]mediaType `json:"content"`
}

type mediaType struct {
	Schema schemaObject `json:"schema"`
}

type response struct {
	Description string               `json:"description"`
	Content     map[string]mediaType `json:"content,omitempty"`
}

// Build renders a deterministic OpenAPI metadata document.
func Build(desc axle.Descriptor, routes []axle.RouteDescriptor) (string, error) {
	paths := map[string]map[string]op{}
	resourceSchemaName := desc.Resource.Name
	components := components{Schemas: map[string]schemaObject{
		resourceSchemaName: resourceSchema(desc.Resource),
		resourceSchemaName + "ListResponse": {
			Type:       "object",
			Properties: map[string]schemaObject{"data": {Type: "array", Items: ref(resourceSchemaName)}},
		},
		resourceSchemaName + "Response": {
			Type:       "object",
			Properties: map[string]schemaObject{"data": *ref(resourceSchemaName)},
		},
		"DeleteResponse": {
			Type:       "object",
			Properties: map[string]schemaObject{"deleted": {Type: "boolean"}, desc.Resource.ID: {Type: "string"}},
		},
	}}
	for _, route := range routes {
		method := lower(route.TransportMethod)
		if paths[route.Path] == nil {
			paths[route.Path] = map[string]op{}
		}
		action := ""
		if route.Kind == "action" {
			action = route.Name
		}
		paths[route.Path][method] = op{
			OperationID:      route.Name,
			Parameters:       pathParameters(route.Params),
			RequestBody:      bodyForRoute(route, resourceSchemaName),
			Responses:        responsesForRoute(route, resourceSchemaName),
			AxleKind:         route.Kind,
			TransportMethod:  route.TransportMethod,
			AxleResource:     desc.Resource.Name,
			AxleAction:       action,
			AxlePathParams:   route.Params,
			AxleRequestType:  route.Request,
			AxleResponseType: route.Response,
		}
	}
	payload, err := json.MarshalIndent(document{OpenAPI: "3.1.0", Info: info{Title: desc.Resource.Name + " API", Version: "0.1.0"}, Paths: paths, Components: components}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload) + "\n", nil
}

func resourceSchema(resource axle.ResourceDescriptor) schemaObject {
	props := map[string]schemaObject{}
	var required []string
	for _, field := range resource.Fields {
		prop := schemaObject{Type: openAPIType(field.Type)}
		if field.Auto == "uuid" {
			prop.Format = "uuid"
		}
		if field.Nullable != nil && *field.Nullable {
			prop.Nullable = true
		}
		props[field.Name] = prop
		if field.Name == resource.ID || (field.Nullable != nil && !*field.Nullable) {
			required = append(required, field.Name)
		}
	}
	return schemaObject{Type: "object", Properties: props, Required: required}
}

func pathParameters(params []string) []parameter {
	out := make([]parameter, 0, len(params))
	for _, param := range params {
		out = append(out, parameter{Name: param, In: "path", Required: true, Schema: schemaObject{Type: "string"}})
	}
	return out
}

func bodyForRoute(route axle.RouteDescriptor, resourceSchemaName string) *requestBody {
	if route.Kind != "create" && route.Kind != "update" && route.Kind != "action" {
		return nil
	}
	return &requestBody{Required: true, Content: map[string]mediaType{"application/json": {Schema: *ref(resourceSchemaName)}}}
}

func responsesForRoute(route axle.RouteDescriptor, resourceSchemaName string) map[string]response {
	schema := *ref(resourceSchemaName + "Response")
	if route.Kind == "list" {
		schema = *ref(resourceSchemaName + "ListResponse")
	}
	if route.Kind == "delete" {
		schema = *ref("DeleteResponse")
	}
	return map[string]response{
		"200": {Description: route.Response, Content: map[string]mediaType{"application/json": {Schema: schema}}},
		"400": {Description: "Bad request"},
		"404": {Description: "Not found"},
	}
}

func ref(name string) *schemaObject {
	return &schemaObject{Ref: "#/components/schemas/" + name}
}

func openAPIType(kind string) string {
	switch kind {
	case "integer":
		return "integer"
	case "boolean":
		return "boolean"
	case "real":
		return "number"
	default:
		return "string"
	}
}

func lower(value string) string {
	out := []byte(value)
	for i, char := range out {
		if char >= 'A' && char <= 'Z' {
			out[i] = char + 32
		}
	}
	return string(out)
}
