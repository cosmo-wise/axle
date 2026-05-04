package cli

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	internalsqlite "github.com/cosmo-wise/axle/internal/sqlite"
	"github.com/cosmo-wise/axle/pkg/axle"
)

func runDoctor(stdout io.Writer, stderr io.Writer, jsonOut bool) int {
	var diagnostics []axle.Diagnostic
	goPath, err := findGoBinary()
	if err != nil {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_DOCTOR_GO", Path: "go", Message: err.Error(), SuggestedFix: "Install Go and ensure go is on PATH or /usr/local/go/bin/go."})
	} else if out, err := exec.Command(goPath, "version").CombinedOutput(); err != nil || !strings.Contains(string(out), "go") {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_DOCTOR_GO_VERSION", Path: goPath, Message: strings.TrimSpace(string(out)), SuggestedFix: "Ensure go version runs successfully."})
	}
	db, err := internalsqlite.Open(":memory:")
	if err != nil {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_DOCTOR_SQLITE", Path: "sqlite", Message: err.Error(), SuggestedFix: "Ensure modernc SQLite can open an in-memory database."})
	} else {
		defer db.Close()
		if _, err := db.ExecContext(context.Background(), "SELECT 1"); err != nil {
			diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_DOCTOR_SQLITE_QUERY", Path: "sqlite", Message: err.Error(), SuggestedFix: "Ensure SQLite queries can execute."})
		}
	}
	return renderResult(stdout, stderr, jsonOut, diagnostics)
}

func findGoBinary() (string, error) {
	if path, err := exec.LookPath("go"); err == nil {
		return path, nil
	}
	const fallback = "/usr/local/go/bin/go"
	if _, err := os.Stat(fallback); err == nil {
		return fallback, nil
	}
	return "", exec.ErrNotFound
}
