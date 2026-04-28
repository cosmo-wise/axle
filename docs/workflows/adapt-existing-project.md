# Adapt an existing project to Axle

Use this when an LLM has an existing frontend or generated app and must add a
small Go/SQLite CRUD backend without inventing router/store glue.

## 1. Scaffold first, then replace sample resources

```bash
go run ./cmd/axle app init --out <backend-dir> --module <module> --axle-replace <path-to-axle> --json
cd <backend-dir>
./scripts/verify.sh
```

`app init` intentionally creates a verified example backend with sample
`resources` and `policies`. It does not infer business resources from a
frontend. For a real project, replace the sample descriptors with project
descriptors and regenerate.

## 2. Extract only stable resource facts

For each resource, write `descriptors/<resource>/descriptor.axle.json` with:

- singular Go-style `resource.name`
- plural URL `resource.path`
- SQLite `resource.table`
- primary key `resource.id`
- persisted `resource.fields`
- all five CRUD operation kinds: `list`, `get`, `create`, `update`, `delete`
- custom `resource.actions` for non-CRUD behavior

Nested actions use relative paths:

```json
{
  "name": "SchedulePlanTask",
  "kind": "action",
  "path": "tasks/{task_id}/schedule",
  "request": "SchedulePlanTaskRequest",
  "response": "SchedulePlanTaskResponse",
  "policy": "plan.write",
  "handler": "SchedulePlanTask"
}
```

This generates `POST /plans/{id}/tasks/{task_id}/schedule` and passes
`request.ID == <plan id>` plus `request.Params["task_id"]`.

## 3. Regenerate before module tidy

After descriptor or catalog edits:

```bash
axle gen --descriptor descriptors/<resource>/descriptor.axle.json --out descriptors/<resource>/generated --json
axle catalog gen --manifest catalog/axle.catalog.json --out catalog --json
go mod tidy
axle check --root . --json
go test ./...
```

The scaffold `scripts/verify.sh` already follows this order. This prevents
`go mod tidy` from resolving stale generated import paths after resource
renames.

## 4. Use generated action binders

Bind only custom actions:

```go
return axleruntime.NewEdge(appcatalog.Catalog, db, axleruntime.ActionHandlers{
    plans.HandlerSchedulePlanTask: plans.BindSchedulePlanTask(acts.SchedulePlanTask),
}, axleruntime.EdgeOptions{
    Name:      "Example backend",
    APIPrefix: "/api/v1",
    CORS:      true,
})
```

Generated request types always expose:

- `ID string`
- `Params map[string]string`
- `Body map[string]any`

Generated action responses use `Data map[string]any`.

## 5. Hand-written code boundary

Write:

- descriptors
- `catalog/axle.catalog.json`
- custom action handlers
- startup/env/seed code
- project-specific tests

Do not write:

- standard CRUD route switches
- generated `.gen.go` files
- repositories or query builders
- generic multi-database abstractions
- reflection or directory-walking registration

## 6. Defaults and IDs

Axle does not generate IDs, timestamps, slugs, or default values. Put those in
client payloads, seed data, tests, or custom action code.

Create/update accepts either a bare JSON object or `{"data": {...}}`.
Update/delete keep semantic CRUD kinds but use POST transport:

- `POST /resources/{id}/update`
- `POST /resources/{id}/delete`
