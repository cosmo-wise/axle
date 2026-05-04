package cli

import "fmt"

func renderAppGoMod(moduleName string, axleReplace string) string {
	return fmt.Sprintf("module %s\n\ngo 1.24.0\n\nrequire github.com/cosmo-wise/axle v0.0.0\n\nreplace github.com/cosmo-wise/axle => %s\n", moduleName, axleReplace)
}

func renderVerifyScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
AXLE=${AXLE:-"go run github.com/cosmo-wise/axle/cmd/axle"}
mapfile -t DESCRIPTORS < <(find descriptors -mindepth 2 -maxdepth 2 -name descriptor.axle.json | sort)
if [ "${#DESCRIPTORS[@]}" -lt 1 ]; then
  echo "expected at least one Axle descriptor" >&2
  exit 1
fi
go mod tidy

for descriptor in "${DESCRIPTORS[@]}"; do
  out_dir="$(dirname "$descriptor")/generated"
  $AXLE gen --descriptor "$descriptor" --out "$out_dir" --json
done
$AXLE catalog gen --manifest catalog/axle.catalog.json --out catalog --json

go mod tidy

for descriptor in "${DESCRIPTORS[@]}"; do
  out_dir="$(dirname "$descriptor")/generated"
  $AXLE gen --descriptor "$descriptor" --out "$out_dir" --check --json
done
$AXLE catalog gen --manifest catalog/axle.catalog.json --out catalog --check --json
$AXLE check --root . --json
go test ./...
go vet ./...
go build -o /tmp/axle-generated-backend-verify ./cmd/example-backend
rm -f /tmp/axle-generated-backend-verify
`
}

func renderAppMain(moduleName string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"log"
	"net/http"
	"os"

	appcatalog "%s/catalog"
	"%s/internal/app"
	axlesqlite "github.com/cosmo-wise/axle/pkg/axle/sqlite"
)

func main() {
	ctx := context.Background()
	dsn := "file:axle.db"
	if len(os.Args) > 1 {
		dsn = os.Args[1]
	}
	db, err := axlesqlite.Open(ctx, dsn, appcatalog.Catalog)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		log.Fatal(err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, app.New(db)))
}
`, moduleName, moduleName)
}

func renderAppReadme(moduleName string) string {
	return fmt.Sprintf(`# Axle generated backend

This backend is a minimal Go/SQLite Axle scaffold for module %q.

## What Axle owns

- Generated CRUD routes and DTOs under %[2]s.
- Multi-resource catalog output under %[3]s.
- SQLite CRUD through %[4]s.
- Optional HTTP edge conveniences through %[5]s: %[6]s, %[7]s, CORS, and %[8]s prefix normalization.

## What app code owns

- Resource descriptor facts in %[9]s.
- Custom action handler business logic in %[10]s.
- Startup configuration in %[11]s.
- Seed/demo data and project-specific tests when the scaffold is adapted to a real app.

Do not handwrite standard CRUD routers, repositories, query builders, generic DB dialects, or runtime directory scanning. Change descriptors/catalog manifests and regenerate.

## Adapt this scaffold to an existing project

1. Replace the sample descriptors in %[9]s with your real resources.
2. Keep every resource descriptor declaring all five CRUD operation kinds: list, get, create, update, delete.
3. Add nested actions with relative action paths such as %[12]s.
4. Regenerate each descriptor output with %[13]s.
5. Update %[14]s with every generated resource import.
6. Run %[15]s. It regenerates first, then runs %[16]s, stale-output checks, %[17]s, tests, vet, and build.
7. Bind only custom actions with generated %[18]s constants and %[19]s helpers.

## Runtime contract

- Reads use %[20]s.
- Creates and custom actions use %[21]s.
- Updates and deletes keep semantic kinds but use %[22]s and %[23]s.
- Create/update accept a bare JSON object or %[24]s.
- Axle can auto-generate descriptor fields marked auto=uuid; timestamps/slugs/default policies remain descriptor-owned unless declared explicitly.

`, moduleName,
		"`descriptors/<resource>/generated`",
		"`catalog/catalog.gen.go`",
		"`github.com/cosmo-wise/axle/pkg/axle/sqlite`",
		"`github.com/cosmo-wise/axle/pkg/axle/runtime.NewEdge`",
		"`/healthz`",
		"`/routes`",
		"`/api/v1`",
		"`descriptors/*/descriptor.axle.json`",
		"`internal/app/app.go`",
		"`cmd/example-backend/main.go`",
		"`policy/{policy_id}/upgrade`",
		"`axle gen --descriptor ... --out ...`",
		"`catalog/axle.catalog.json`",
		"`./scripts/verify.sh`",
		"`go mod tidy`",
		"`axle check --root . --json`",
		"`Handler<Action>`",
		"`Bind<Action>`",
		"`GET /resources` and `GET /resources/{id}`",
		"`POST /resources` and `POST /resources/{id}/action`",
		"`POST /resources/{id}/update`",
		"`POST /resources/{id}/delete`",
		"`{\"data\": {...}}`",
	)
}
