# Add an action route

1. Add an entry under `resource.actions` in the descriptor.
2. Use POST transport for mutating actions while keeping semantic metadata in `kind` and OpenAPI extensions.
3. For nested paths, encode explicit parameters in the action path, for example:

```json
{"name":"UpgradeResourcePolicy","kind":"action","path":"policy/{policy_id}/upgrade"}
```

4. Regenerate and check:

```bash
go run ./cmd/axle gen --descriptor <descriptor> --out <fixture>/generated --json
go run ./cmd/axle gen --descriptor <descriptor> --out <fixture>/generated --check --json
go run ./cmd/axle check --descriptor <descriptor> --root . --json
```
