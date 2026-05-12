# Template Extension Interface

Custom templates can be injected by placing Go `text/template` files
in a `templates/` directory within the project using Axle.

## Extension Contract

Each custom template must define:
- `{{define "type"}}` — the type definition block
- `{{define "routes"}}` — the HTTP route registration block
- `{{define "handler"}}` — the handler function block

## Example

See `examples/` for sample custom template implementations.
