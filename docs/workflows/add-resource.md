# Add a toy resource

1. Copy `testdata/fixtures/single/descriptor.axle.json` to a test-only fixture path.
2. Edit only the descriptor first: resource name, URL path, table, fields, CRUD bindings.
3. Generate deterministic output:

```bash
go run ./cmd/axle gen --descriptor <descriptor> --out <fixture>/generated --json
```

4. Check the contract:

```bash
go run ./cmd/axle check --descriptor <descriptor> --root . --json
```

5. Run `go test ./...`.

Do not create real product modules in Axle V1. Fixtures must stay toy-only.
