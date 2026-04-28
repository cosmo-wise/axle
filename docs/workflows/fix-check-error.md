# Fix an `axle check` error

`axle check --json` returns `status` and `diagnostics[]` with `code`, `path`, `message`, and `suggested_fix`.

Common fixes:

- `AXLE_OPERATION_REQUEST`, `AXLE_OPERATION_RESPONSE`, `AXLE_OPERATION_POLICY`, `AXLE_OPERATION_HANDLER`: add the missing binding to the descriptor operation/action.
- `AXLE_GENERATED_MISSING`: run `axle gen` or `axle catalog gen` for the missing output.
- `AXLE_GENERATED_STALE`: regenerate from descriptors or `catalog/axle.catalog.json`; do not hand-edit generated files.
- `AXLE_CATALOG_PACKAGE`, `AXLE_CATALOG_ALIAS`, `AXLE_CATALOG_IMPORT`, `AXLE_CATALOG_DUPLICATE`: fix `catalog/axle.catalog.json` schema, aliases, or imports.
- `AXLE_BOUNDARY_CONTROLLER_DB`: move storage calls behind an app action/service binding.
- `AXLE_BOUNDARY_SERVICE_HTTP`: return structured values from services and render HTTP at the edge.
- `AXLE_PUBLIC_IMPORT_INTERNAL`: keep public Axle packages independent of apps, fixtures, and generated examples.
- `AXLE_PUBLIC_API_BLOAT`: keep root `pkg/axle` metadata-only; HTTP/SQLite behavior belongs in `pkg/axle/runtime` or `pkg/axle/sqlite`.
- `AXLE_MULTIDB_ABSTRACTION`: remove generic DB abstractions; V1 is concrete SQLite-only.
- `AXLE_RUNTIME_DISCOVERY`: use descriptor-generated catalog registration, not reflection or directory walking.
- `AXLE_MANUAL_CRUD_ROUTING`: mount generated `catalog.Catalog` with `pkg/axle/runtime.New` or `pkg/axle/runtime.NewEdge`; app wrappers may expose `/healthz`, `/routes`, CORS, and prefixes but must not duplicate standard CRUD routes.
- `AXLE_TYPED_ORM_CREEP`: remove repositories/query builders; use the concrete SQLite CRUD facade plus generated DTO edge.
