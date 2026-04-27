package axle

// Descriptor is Axle's V1 pre-generation contract.
type Descriptor struct {
	Schema    string             `json:"schema"`
	Resource  ResourceDescriptor `json:"resource"`
	Generated GeneratedTarget    `json:"generated"`
}

// GeneratedTarget describes where generated code should land.
type GeneratedTarget struct {
	Package string `json:"package"`
}

// ResourceDescriptor defines one CRUD resource and explicit actions.
type ResourceDescriptor struct {
	Name       string                `json:"name"`
	Path       string                `json:"path"`
	Table      string                `json:"table"`
	ID         string                `json:"id"`
	Fields     []FieldDescriptor     `json:"fields"`
	Operations []OperationDescriptor `json:"operations"`
	Actions    []OperationDescriptor `json:"actions,omitempty"`
}

// FieldDescriptor defines a persisted resource field.
type FieldDescriptor struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Mutable bool   `json:"mutable"`
}

// OperationDescriptor defines CRUD or action behavior.
type OperationDescriptor struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path,omitempty"`
	Request  string `json:"request"`
	Response string `json:"response"`
	Policy   string `json:"policy"`
	Handler  string `json:"handler"`
}

// RouteDescriptor is the public route contract generated code may import.
type RouteDescriptor struct {
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`
	TransportMethod string   `json:"transport_method"`
	Path            string   `json:"path"`
	Params          []string `json:"params"`
	Request         string   `json:"request"`
	Response        string   `json:"response"`
	Policy          string   `json:"policy"`
	Handler         string   `json:"handler"`
}

// Diagnostic is the stable machine-readable repair contract for Axle checks.
type Diagnostic struct {
	Code         string `json:"code"`
	Path         string `json:"path"`
	Message      string `json:"message"`
	SuggestedFix string `json:"suggested_fix"`
}

// CheckResult is returned by JSON-producing CLI commands.
type CheckResult struct {
	Status      string       `json:"status"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
