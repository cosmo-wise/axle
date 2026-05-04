# Axle generated backend

This backend is a minimal Go/SQLite Axle scaffold for module "github.com/cosmo-wise/axle/testdata/examples/generated-backend".

## What Axle owns

- Generated CRUD routes and DTOs under `descriptors/<resource>/generated`.
- Multi-resource catalog output under `catalog/catalog.gen.go`.
- SQLite CRUD through `github.com/cosmo-wise/axle/pkg/axle/sqlite`.
- Optional HTTP edge conveniences through `github.com/cosmo-wise/axle/pkg/axle/runtime.NewEdge`: `/healthz`, `/routes`, CORS, and `/api/v1` prefix normalization.

## What app code owns

- Resource descriptor facts in `descriptors/*/descriptor.axle.json`.
- Custom action handler business logic in `internal/app/app.go`.
- Startup configuration in `cmd/example-backend/main.go`.
- Seed/demo data and project-specific tests when the scaffold is adapted to a real app.

Do not handwrite standard CRUD routers, repositories, query builders, generic DB dialects, or runtime directory scanning. Change descriptors/catalog manifests and regenerate.

## Adapt this scaffold to an existing project

1. Replace the sample descriptors in `descriptors/*/descriptor.axle.json` with your real resources.
2. Keep every resource descriptor declaring all five CRUD operation kinds: list, get, create, update, delete.
3. Add nested actions with relative action paths such as `policy/{policy_id}/upgrade`.
4. Regenerate each descriptor output with `axle gen --descriptor ... --out ...`.
5. Update `catalog/axle.catalog.json` with every generated resource import.
6. Run `./scripts/verify.sh`. It regenerates first, then runs `go mod tidy`, stale-output checks, `axle check --root . --json`, tests, vet, and build.
7. Bind only custom actions with generated `Handler<Action>` constants and `Bind<Action>` helpers.

## Runtime contract

- Reads use `GET /resources` and `GET /resources/{id}`.
- Creates and custom actions use `POST /resources` and `POST /resources/{id}/action`.
- Updates and deletes keep semantic kinds but use `POST /resources/{id}/update` and `POST /resources/{id}/delete`.
- Create/update accept a bare JSON object or `{"data": {...}}`.
- Axle auto-generates only descriptor-declared values such as text fields with `auto: "uuid"`; timestamps, slugs, and arbitrary defaults remain app/client responsibilities unless the descriptor explicitly adds support.
