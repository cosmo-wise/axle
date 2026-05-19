<h1 align="center">Axle</h1>

<p align="center">
  A descriptor-driven CRUD framework for LLM-maintained Go backends
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?logo=go" alt="Go 1.24">
  <img src="https://img.shields.io/badge/license-Apache%202.0-blue" alt="Apache 2.0">
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> &middot;
  <a href="#install">Install</a> &middot;
  <a href="#usage">Usage</a> &middot;
  <a href="docs/workflows/create-backend.md">Docs</a>
</p>

---

## The Problem

Writing and maintaining CRUD backend code is repetitive boilerplate. When an LLM generates it from scratch each time, the output drifts — different route patterns, inconsistent payload shapes, bespoke SQLite wiring, missing migrations. The backend diverges from the frontend contract, and every regeneration risks introducing subtle breakage.

Axle solves this with a descriptor-first approach: define your resources in JSON, generate deterministic Go code, mount the result as an HTTP handler, and let `axle check` catch drift before it reaches production.

## Features

- **Descriptor-first generation** — Define fields, CRUD operations, and custom actions in JSON. Axle generates Go types, route metadata, registries, OpenAPI specs, and SQL migrations from a single descriptor.
- **Multi-resource catalogs** — Combine any number of resource descriptors into a unified catalog with a simple JSON manifest. The catalog mounts every resource behind a single HTTP handler.
- **Concrete SQLite CRUD** — No ORM, no query builder, no database abstraction. A single pure-Go SQLite facade handles list, get, create, update, and delete for every generated resource.
- **Deterministic, untouchable output** — Generated files carry `DO NOT EDIT` headers and are byte-identical across identical descriptors. No reflection, no runtime directory scanning, no registration magic.
- **Architecture anti-bloat checks** — `axle check` rejects handwritten CRUD routes, generic database wrappers, typed repositories, reflection discovery, and public API bloat — the patterns LLMs reach for by default.
- **Backend scaffolding** — `axle app init` produces a complete, verified Go backend skeleton with generated CRUD, migrations, verify script, and a clear ownership split between Axle-owned generated code and app-owned handlers.
- **Edge runtime conveniences** — The optional `NewEdge` wrapper adds `/healthz`, `/routes`, CORS, and API prefix normalization without hand-written route switches.

## When to Use

Reach for Axle when you need a Go/SQLite CRUD backend maintained through AI-assisted regeneration. Use after you've defined your resource model, not during initial prototyping.

**Not for:** Multi-database support, admin UI generation, typed ORM layers, reflection-based registration, or runtime directory scanning. Axle V1 is deliberately scoped to single-process SQLite backends with descriptor-driven generation.

## Quick Start

```bash
# Create a new backend skeleton
go run github.com/cosmo-wise/axle/cmd/axle app init \
  --out /tmp/axle-backend \
  --module example.com/axle-backend \
  --axle-replace /path/to/axle

# Verify it works
cd /tmp/axle-backend && ./scripts/verify.sh
```

## Install

```bash
go install github.com/cosmo-wise/axle/cmd/axle@latest
```

Requires Go 1.24+. No runtime dependencies beyond a pure-Go SQLite build.

## Usage

### Core loop

Define a resource in `descriptors/tasks/descriptor.axle.json`, then generate:

```bash
axle gen --descriptor descriptors/tasks/descriptor.axle.json --out descriptors/tasks/generated
```

The output includes `types.gen.go`, `routes.gen.go`, `registry.gen.go`, `openapi.gen.json`, and migration SQL.

Check generated output without writing:

```bash
axle gen --descriptor descriptors/tasks/descriptor.axle.json --out descriptors/tasks/generated --check --json
```

### Multi-resource catalog

For apps with two or more resources, create `catalog/axle.catalog.json`:

```json
{
  "package": "catalog",
  "resources": [
    {"alias": "plans", "import": "example.com/app/descriptors/plans/generated"},
    {"alias": "tasks", "import": "example.com/app/descriptors/tasks/generated"}
  ]
}
```

```bash
axle catalog gen --manifest catalog/axle.catalog.json --out catalog
```

### Mount as HTTP server

```go
db, _ := axlesqlite.Open(ctx, ":memory:", appcatalog.Catalog)
db.Migrate(ctx)
handler := axleruntime.New(appcatalog.Catalog, db, axleruntime.ActionHandlers{
    resources.HandlerRenameResource: resources.BindRenameResource(renameResource),
})
```

Or with edge conveniences:

```go
handler := axleruntime.NewEdge(appcatalog.Catalog, db, handlers, axleruntime.EdgeOptions{
    Name:      "Task backend",
    APIPrefix: "/api/v1",
    CORS:      true,
})
```

### Run architecture checks

```bash
axle check --root . --json
```

This rejects handwritten CRUD routing, database abstractions, reflection, stale generated files, and other common LLM drift patterns.

### Verify readiness

```bash
axle doctor --json
```

Reports Go binary availability and SQLite runtime health.

## Project Boundaries

- `cmd/axle` — thin CLI edge
- `internal/*` — descriptor parsing, code generation, OpenAPI, checks, SQLite internals
- `pkg/axle` — metadata-only public contracts (Descriptor, Catalog, ResourceRegistry)
- `pkg/axle/runtime` — HTTP mount for generated catalogs
- `pkg/axle/sqlite` — concrete SQLite CRUD facade

Generated files carry `Code generated by axle gen; DO NOT EDIT.` Change descriptors or catalog manifests, then regenerate.

---

## How It Works

Axle reads a JSON resource descriptor, parses it against a versioned schema, then deterministically generates Go source files, an OpenAPI spec, and SQL migration. Generated types include resource DTOs, CRUD DTOs, typed action binders, and route metadata. A catalog manifest combines multiple generated packages into one registry. The runtime matches incoming HTTP requests against catalog route metadata and dispatches to the SQLite facade or to registered custom action handlers.

→ [Architecture deep dive](docs/workflows/create-backend.md)

## Workflows

Axle ships with workflow guides for common operations:

- [Create a new backend](docs/workflows/create-backend.md) — full walkthrough from `app init` to running server
- [Add a resource](docs/workflows/add-resource.md) — incremental resource addition to an existing backend
- [Add a custom action](docs/workflows/add-action.md) — adding typed custom action handlers
- [Adapt an existing project](docs/workflows/adapt-existing-project.md) — adding Axle to an existing codebase
- [Regenerate and check](docs/workflows/regenerate-and-check.md) — the standard verify loop
- [Fix a check error](docs/workflows/fix-check-error.md) — resolving common `axle check` failures
- [Harness integration](docs/workflows/harness-integration.md) — wiring Axle into Chariot's test harness

## Deployment

Axle backends are standard Go binaries. Build with `go build` and deploy as a single binary. The SQLite database file path is specified at runtime — there is no embedded server or external database process.

## Contributing

Contributions are welcome. Open an issue or pull request on [GitHub](https://github.com/cosmo-wise/axle).

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
