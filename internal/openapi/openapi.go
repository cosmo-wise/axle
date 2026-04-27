package openapi

import (
	"encoding/json"

	"github.com/Fel1xKan/axle/pkg/axle"
)

type document struct {
	OpenAPI string                   `json:"openapi"`
	Info    info                     `json:"info"`
	Paths   map[string]map[string]op `json:"paths"`
}

type info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type op struct {
	OperationID      string            `json:"operationId"`
	Responses        map[string]string `json:"responses"`
	AxleKind         string            `json:"x-axle-kind"`
	TransportMethod  string            `json:"x-transport-method"`
	AxleResource     string            `json:"x-axle-resource"`
	AxleAction       string            `json:"x-axle-action,omitempty"`
	AxlePathParams   []string          `json:"x-axle-path-params,omitempty"`
	AxleRequestType  string            `json:"x-axle-request"`
	AxleResponseType string            `json:"x-axle-response"`
}

// Build renders a deterministic OpenAPI metadata document.
func Build(desc axle.Descriptor, routes []axle.RouteDescriptor) (string, error) {
	paths := map[string]map[string]op{}
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
			Responses:        map[string]string{"200": route.Response},
			AxleKind:         route.Kind,
			TransportMethod:  route.TransportMethod,
			AxleResource:     desc.Resource.Name,
			AxleAction:       action,
			AxlePathParams:   route.Params,
			AxleRequestType:  route.Request,
			AxleResponseType: route.Response,
		}
	}
	payload, err := json.MarshalIndent(document{OpenAPI: "3.1.0", Info: info{Title: desc.Resource.Name + " API", Version: "0.1.0"}, Paths: paths}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload) + "\n", nil
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
