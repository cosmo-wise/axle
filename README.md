# Axle

Axle is a Chariot-family Go/SQLite CRUD framework skeleton optimized for LLM-maintained resource modules.

V1 is descriptor-first: edit a JSON resource descriptor, run deterministic generation, run structured checks, then run tests. Axle intentionally avoids business modules, Admin UI generation, multi-database abstractions, runtime directory scanning, and reflection-based resource discovery.

## Core loop

```bash
go run ./cmd/axle gen --descriptor testdata/fixtures/single/descriptor.axle.json --out testdata/fixtures/single/generated
go run ./cmd/axle gen --descriptor testdata/fixtures/single/descriptor.axle.json --out testdata/fixtures/single/generated --check --json
go run ./cmd/axle check --descriptor testdata/fixtures/single/descriptor.axle.json --root . --json
go test ./...
```

## Boundaries

- `cmd/axle` is a thin CLI edge.
- `internal/*` owns descriptor parsing, deterministic generation, OpenAPI, checks, and concrete SQLite support.
- `pkg/axle` is the minimal public contract generated code may import.
