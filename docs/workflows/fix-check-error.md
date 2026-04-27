# Fix an `axle check` error

`axle check --json` returns:

- `status`
- `diagnostics[].code`
- `diagnostics[].path`
- `diagnostics[].message`
- `diagnostics[].suggested_fix`

Common fixes:

- `AXLE_OPERATION_REQUEST`, `AXLE_OPERATION_RESPONSE`, `AXLE_OPERATION_POLICY`, `AXLE_OPERATION_HANDLER`: add the missing binding to the descriptor.
- `AXLE_GENERATED_STALE`: run `axle gen`; do not hand-edit generated files.
- `AXLE_BOUNDARY_CONTROLLER_DB`: move storage calls behind a service/handler binding.
- `AXLE_BOUNDARY_SERVICE_HTTP`: return structured values from services and render HTTP at the edge.
- `AXLE_PUBLIC_IMPORT_INTERNAL`: keep `pkg/axle` limited to stable generated-code contracts.
- `AXLE_MULTIDB_ABSTRACTION`: remove generic DB abstractions; V1 is concrete SQLite-only.
- `AXLE_RUNTIME_DISCOVERY`: use descriptor-generated registration instead of reflection or directory discovery.
