#!/bin/bash
# Generate TypeScript API client from Axle OpenAPI spec
# Usage: ./scripts/gen-ts-client.sh <openapi.json> <output-dir>

set -e

OPENAPI="${1:-openapi.gen.json}"
OUTDIR="${2:-ts-client}"

if [ ! -f "$OPENAPI" ]; then
  echo "Usage: $0 <openapi.json> [output-dir]"
  echo "Run 'axle gen' first to produce openapi.gen.json"
  exit 1
fi

echo "Generating TypeScript client from $OPENAPI to $OUTDIR/"

# Option A: openapi-typescript (type-only, lightweight)
npx --yes openapi-typescript "$OPENAPI" -o "$OUTDIR/schema.d.ts"

# Option B: openapi-typescript-codegen (full client with fetch)
# npx --yes openapi-typescript-codegen --input "$OPENAPI" --output "$OUTDIR" --client fetch

echo "Done. Types in $OUTDIR/schema.d.ts"
