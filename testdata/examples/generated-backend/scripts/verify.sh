#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
AXLE=${AXLE:-"go run github.com/Fel1xKan/axle/cmd/axle"}

# Regenerate before Go module resolution so renamed resources/imports do not
# strand go mod tidy on stale generated packages.
$AXLE gen --descriptor descriptors/resources/descriptor.axle.json --out descriptors/resources/generated --json
$AXLE gen --descriptor descriptors/policies/descriptor.axle.json --out descriptors/policies/generated --json
$AXLE catalog gen --manifest catalog/axle.catalog.json --out catalog --json

go mod tidy

$AXLE gen --descriptor descriptors/resources/descriptor.axle.json --out descriptors/resources/generated --check --json
$AXLE gen --descriptor descriptors/policies/descriptor.axle.json --out descriptors/policies/generated --check --json
$AXLE catalog gen --manifest catalog/axle.catalog.json --out catalog --check --json
$AXLE check --root . --json
go test ./...
go vet ./...
go build -o /tmp/axle-generated-backend-verify ./cmd/example-backend
rm -f /tmp/axle-generated-backend-verify
