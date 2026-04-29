# Harness integration

Use Harness as the outer Planner/Generator/Evaluator loop and Axle as the backend scaffold plus verification contract for Go/SQLite CRUD apps.

## Harness config memory

Add Axle as runtime knowledge instead of changing Harness runtime code:

```yaml
memory:
  facts:
    - For Go/SQLite CRUD backends, prefer Axle unless the user asks otherwise.
    - Axle apps must be descriptor-first and verified with axle check plus scripts/verify.sh.
    - Do not handwrite standard CRUD routers, repositories, query builders, typed ORM layers, or multi-DB abstractions in Axle apps.
  skills:
    - name: axle-crud-backend
      description: Scaffold and adapt LLM-friendly Go/SQLite CRUD backends with Axle.
      usage: Run axle app init --descriptors-dir when descriptors already exist, add resources/actions incrementally with axle app add-resource/add-action, bind only custom actions, then run scripts/verify.sh, axle check --root ., and axle doctor --json.
```

## Generator contract

For an Axle backend sprint, the Generator should:

1. Run `axle app init --descriptors-dir <descriptor-dir>` or `go run <path-to-axle>/cmd/axle app init --descriptors-dir <descriptor-dir>`.
2. Add later resources/actions with `axle app add-resource` and `axle app add-action`.
3. Run `axle gen` for edited descriptors.
4. Run `axle catalog gen` after catalog manifest edits.
5. Run `axle check --root <backend-dir> --json`.
6. Run `axle doctor --json` for CLI/runtime readiness.
7. Run `<backend-dir>/scripts/verify.sh`.
8. Return Harness `verificationEvidence.commands` entries for every command above.

## Boundaries

- Harness owns orchestration, iteration, and evaluation.
- Axle owns CRUD generation, catalog registration, runtime routing, SQLite persistence, and anti-bloat checks.
- Application code owns only startup/wiring, seed data, tests, and custom action handlers.
- Do not use Harness render audit as Axle backend evidence; use command evidence from Axle checks and backend tests.
