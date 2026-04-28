package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fel1xKan/axle/internal/cli"
	"github.com/Fel1xKan/axle/pkg/axle"
)

func TestCLIJSONDiagnosticsArray(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Main([]string{"check", "--descriptor", "../../testdata/fixtures/single/descriptor.axle.json", "--root", "../..", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cli failed code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var result axle.CheckResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Diagnostics == nil {
		t.Fatalf("diagnostics should be an empty array, got nil; output=%s", stdout.String())
	}
}

func TestCLIAppInitGeneratedBackendE2E(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "backend")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Main([]string{"app", "init", "--out", out, "--module", "example.com/generatedbackend", "--axle-replace", repoRoot, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("app init failed code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var result struct {
		Status    string   `json:"status"`
		Readme    string   `json:"readme"`
		Verify    string   `json:"verify"`
		NextSteps []string `json:"next_steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid json %v: %s", err, stdout.String())
	}
	if result.Status != "ok" || result.Readme == "" || result.Verify == "" || len(result.NextSteps) == 0 {
		t.Fatalf("unexpected app init result: %s", stdout.String())
	}
	readme, err := os.ReadFile(filepath.Join(out, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "Adapt this scaffold to an existing project") {
		t.Fatalf("generated README should explain project adaptation: %s", readme)
	}
	verifyScript, err := os.ReadFile(filepath.Join(out, "scripts", "verify.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(verifyScript), "go mod tidy") < strings.Index(string(verifyScript), "axle catalog gen --manifest catalog/axle.catalog.json --out catalog --json") {
		t.Fatalf("verify script should regenerate before go mod tidy:\n%s", verifyScript)
	}
	cmd := exec.Command("bash", "scripts/verify.sh")
	cmd.Dir = out
	cmd.Env = append(os.Environ(), "PATH=/usr/local/go/bin:"+os.Getenv("PATH"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated backend failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "ok") {
		t.Fatalf("expected generated backend tests to run, got: %s", output)
	}
}

func TestCLICatalogGenCheck(t *testing.T) {
	out := t.TempDir()
	manifest := filepath.Join(out, "axle.catalog.json")
	if err := os.WriteFile(manifest, []byte(`{
  "package": "catalog",
  "resources": [
    {"alias": "tasks", "import": "example.com/app/descriptors/tasks/generated"},
    {"alias": "plans", "import": "example.com/app/descriptors/plans/generated"}
  ]
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Main([]string{"catalog", "gen", "--manifest", manifest, "--out", out, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("catalog gen failed code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(out, "catalog.gen.go")); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Main([]string{"catalog", "gen", "--manifest", manifest, "--out", out, "--check", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("catalog check failed code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}
