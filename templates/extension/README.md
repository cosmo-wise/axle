# Template Extension Interface

Axle code generation templates are compiled into Go code (`internal/cli/app_templates.go`)
and are not currently extensible through external template files.

To customize generated output:

1. Edit the descriptor and regenerate with `axle gen`.
2. For custom action handlers, implement them in the app's `internal/app/app.go`
   and bind them with generated `Handler<Action>` constants and `Bind<Action>` helpers.
3. For structural changes to generation output, edit the Go template functions
   in `internal/cli/app_templates.go` within the Axle repository.

Axle's template architecture is intentionally kept in Go code to ensure
deterministic, testable generation output. This avoids the drift and fragility
of external template injection.
