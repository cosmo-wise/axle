# Create an Axle backend

Use this when an LLM needs a complete Go/SQLite backend skeleton without inventing router/store glue.

```bash
go run ./cmd/axle app init --out <backend-dir> --module <module> --axle-replace <path-to-axle> --json
cd <backend-dir>
./scripts/verify.sh
```

## Ownership boundaries

- Axle owns standard CRUD routing and SQLite CRUD through generated `catalog.Catalog`, `pkg/axle/runtime`, and `pkg/axle/sqlite`.
- The app owns custom action handlers only.
- Use `pkg/axle/runtime.NewEdge` for thin HTTP edge conveniences: `/healthz`, `/routes`, CORS headers, and `/api/v1` prefix normalization.
- Do not handwrite standard CRUD `net/http` route switches.
- Do not add typed repositories, query builders, dialects, drivers, or reflection discovery.

## LLM edit loop

1. Edit `descriptors/<resource>/descriptor.axle.json` first.
2. Run `axle gen --descriptor ... --out descriptors/<resource>/generated` for each edited descriptor.
3. Add generated resource imports to `catalog/axle.catalog.json`.
4. Run `axle catalog gen --manifest catalog/axle.catalog.json --out catalog`.
5. Bind only custom actions with generated `Handler<Action>` constants and `Bind<Action>` helpers.
6. Run `axle check --root . --json` and then `scripts/verify.sh`.

The generated scaffold uses sample `resources` and `policies` descriptors. It
does not infer domain resources from an existing frontend. Replace those
descriptors with project resources, regenerate, and keep the same ownership
split. For a project-adaptation checklist, see `adapt-existing-project.md`.

## Payload and defaults

- CRUD create/update accept a bare JSON object or `{"data": {...}}`.
- Actions receive `request.ID`, `request.Params`, and `request.Body`.
- Axle does not generate IDs, timestamps, slugs, or default values; app code, seed data, tests, or clients must provide them.
- All writes use POST transport. Update/delete keep semantic route names: `POST /resources/{id}/update`, `POST /resources/{id}/delete`.

## Hand-written code boundary

| Write by hand | Do not write by hand |
| --- | --- |
| Descriptor facts | CRUD routers |
| Catalog manifest imports | Generated `.gen.go` files |
| Custom action handlers | Repositories/query builders |
| Startup env/DB path/seed calls | Multi-database abstractions |
| Project-specific tests | Reflection or directory scanning registration |
