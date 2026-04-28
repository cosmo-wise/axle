# Add a resource

1. Create `descriptors/<resource>/descriptor.axle.json` from an existing descriptor.
2. Edit only descriptor facts first: resource name, URL path, table, ID field, fields, CRUD operations, policies, and handlers.
3. Generate deterministic output:

```bash
go run ./cmd/axle gen --descriptor descriptors/<resource>/descriptor.axle.json --out descriptors/<resource>/generated --json
```

4. Add the generated package to `catalog/axle.catalog.json`:

```json
{"alias":"tasks","import":"example.com/app/descriptors/tasks/generated"}
```

5. Regenerate the catalog before `go mod tidy`, then check:

```bash
go run ./cmd/axle catalog gen --manifest catalog/axle.catalog.json --out catalog --json
go mod tidy
go run ./cmd/axle check --root . --json
go test ./...
```

Do not create repositories, query builders, manual CRUD routes, or multi-DB abstractions. App code should mount `catalog.Catalog` with `pkg/axle/runtime` and `pkg/axle/sqlite`.
