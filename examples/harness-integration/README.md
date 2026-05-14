# Harness Integration Example

This example shows how Axle descriptors integrate with Harness
for automated backend generation.

## Setup

1. Place your `.axle.json` descriptor files in the target repo.
2. Run `axle gen` for each descriptor and `axle catalog gen` for the catalog.
3. Run `axle check --root . --json` to validate the backend.
4. Configure Harness with Axle as runtime knowledge (see `docs/workflows/harness-integration.md`).

## Descriptor Example

```json
{
  "schema": "axle.resource.v1",
  "resource": {
    "name": "Task",
    "path": "tasks",
    "table": "tasks",
    "id": "id",
    "fields": [
      { "name": "id", "type": "text", "mutable": false },
      { "name": "title", "type": "text", "mutable": true },
      { "name": "done", "type": "text", "mutable": true }
    ],
    "operations": [
      {
        "name": "ListTasks",
        "kind": "list",
        "request": "ListTasksRequest",
        "response": "ListTasksResponse",
        "policy": "task.read",
        "handler": "ListTasks"
      },
      {
        "name": "GetTask",
        "kind": "get",
        "request": "GetTaskRequest",
        "response": "GetTaskResponse",
        "policy": "task.read",
        "handler": "GetTask"
      },
      {
        "name": "CreateTask",
        "kind": "create",
        "request": "CreateTaskRequest",
        "response": "CreateTaskResponse",
        "policy": "task.write",
        "handler": "CreateTask"
      },
      {
        "name": "UpdateTask",
        "kind": "update",
        "request": "UpdateTaskRequest",
        "response": "UpdateTaskResponse",
        "policy": "task.write",
        "handler": "UpdateTask"
      },
      {
        "name": "DeleteTask",
        "kind": "delete",
        "request": "DeleteTaskRequest",
        "response": "DeleteTaskResponse",
        "policy": "task.write",
        "handler": "DeleteTask"
      }
    ]
  },
  "generated": {
    "package": "generated"
  }
}
```

## Commands

```bash
# Generate resource code from descriptor
go run ./cmd/axle gen --descriptor descriptors/task/descriptor.axle.json --out descriptors/task/generated --json

# Generate multi-resource catalog
go run ./cmd/axle catalog gen --manifest catalog/axle.catalog.json --out catalog --json

# Run architecture checks
go run ./cmd/axle check --root . --json

# Run doctor for CLI/runtime readiness
go run ./cmd/axle doctor --json

# Run tests
go test ./...
```
