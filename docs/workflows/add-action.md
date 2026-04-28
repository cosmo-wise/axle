# Add an action route

1. Add an entry under `resource.actions` in the descriptor.
2. Use POST transport for mutating actions while keeping semantic metadata in `kind`, route name, and OpenAPI extensions.
3. For nested paths, encode explicit parameters in the action path:

```json
{"name":"UpgradeResourcePolicy","kind":"action","path":"policy/{policy_id}/upgrade","request":"UpgradeResourcePolicyRequest","response":"UpgradeResourcePolicyResponse","policy":"resource.write","handler":"UpgradeResourcePolicy"}
```

4. Regenerate the descriptor output. The generated `types.gen.go` will include:

- `const HandlerUpgradeResourcePolicy = "UpgradeResourcePolicy"`
- `UpgradeResourcePolicyRequest{ID string, Params map[string]string, Body map[string]any}`
- `UpgradeResourcePolicyResponse{Data map[string]any}`
- `BindUpgradeResourcePolicy(...) axleruntime.ActionHandler`

5. Implement only the custom action handler and bind it in the app:

```go
handlers := axleruntime.ActionHandlers{
    generated.HandlerUpgradeResourcePolicy: generated.BindUpgradeResourcePolicy(upgradePolicy),
}
```

The typed request shape is intentionally stable: `ID` is the primary resource
ID, `Params` contains every `{param}` from the route path, and `Body` is the
request JSON object. The handler should return the generated response type,
usually `Response{Data: map[string]any{...}}`.

6. Run:

```bash
go run ./cmd/axle gen --descriptor <descriptor> --out <generated-dir> --check --json
go run ./cmd/axle catalog gen --manifest catalog/axle.catalog.json --out catalog --check --json
go run ./cmd/axle check --root . --json
```
