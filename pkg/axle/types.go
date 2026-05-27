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
	OwnerField string                `json:"ownerField,omitempty"`
	Fields     []FieldDescriptor     `json:"fields"`
	Operations []OperationDescriptor `json:"operations"`
	Actions    []OperationDescriptor `json:"actions,omitempty"`
}

// FieldDescriptor defines a persisted resource field.
type FieldDescriptor struct {
	Name       string               `json:"name"`
	Type       string               `json:"type"`
	Mutable    bool                 `json:"mutable"`
	Nullable   *bool                `json:"nullable,omitempty"`
	Unique     bool                 `json:"unique,omitempty"`
	Index      bool                 `json:"index,omitempty"`
	Auto       string               `json:"auto,omitempty"`
	Default    string               `json:"default,omitempty"`
	References *ReferenceDescriptor `json:"references,omitempty"`
}

// ReferenceDescriptor declares a SQLite foreign-key reference for a field.
type ReferenceDescriptor struct {
	Resource string `json:"resource,omitempty"`
	Table    string `json:"table"`
	Field    string `json:"field,omitempty"`
	OnDelete string `json:"on_delete,omitempty"`
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

// ResourceRegistry is the generated contract consumed by Axle runtime mounts.
// It is intentionally metadata-only: runtime packages own HTTP and SQLite behavior.
type ResourceRegistry struct {
	Resource ResourceDescriptor `json:"resource"`
	Routes   []RouteDescriptor  `json:"routes"`
}

// Catalog is a deterministic multi-resource registry generated or assembled by Axle.
type Catalog struct {
	Resources []ResourceRegistry `json:"resources"`
}

// Routes returns all catalog routes in registry order.
func (c Catalog) Routes() []RouteDescriptor {
	var routes []RouteDescriptor
	for _, resource := range c.Resources {
		routes = append(routes, resource.Routes...)
	}
	return routes
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
