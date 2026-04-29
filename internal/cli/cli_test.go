package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	if strings.Index(string(verifyScript), "go mod tidy") > strings.Index(string(verifyScript), "$AXLE gen --descriptor") {
		t.Fatalf("verify script should bootstrap go.sum before default AXLE go run:\n%s", verifyScript)
	}
	if strings.LastIndex(string(verifyScript), "go mod tidy") < strings.Index(string(verifyScript), "$AXLE catalog gen --manifest catalog/axle.catalog.json --out catalog --json") {
		t.Fatalf("verify script should tidy again after regeneration:\n%s", verifyScript)
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

func TestCLIAppInitDescriptorsDirAndIncrementalCommands(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	seed := t.TempDir()
	writeDescriptorFixture(t, seed, "Widget", "widgets", "widgets")
	writeDescriptorFixture(t, seed, "Policy", "policies", "policies")
	out := filepath.Join(t.TempDir(), "backend")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Main([]string{"app", "init", "--out", out, "--module", "example.com/custombackend", "--axle-replace", repoRoot, "--descriptors-dir", seed, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("app init descriptors-dir failed code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(out, "descriptors", "widgets", "descriptor.axle.json")); err != nil {
		t.Fatalf("custom widget descriptor not installed: %v", err)
	}
	commentDescriptor := filepath.Join(t.TempDir(), "comments.axle.json")
	writeDescriptorFile(t, commentDescriptor, "Comment", "comments", "comments")
	stdout.Reset()
	stderr.Reset()
	code = cli.Main([]string{"app", "add-resource", "--out", out, "--descriptor", commentDescriptor, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("app add-resource failed code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(out, "descriptors", "comments", "descriptor.axle.json")); err != nil {
		t.Fatalf("incremental resource descriptor not installed: %v", err)
	}
	widgetDescriptor := filepath.Join(out, "descriptors", "widgets", "descriptor.axle.json")
	stdout.Reset()
	stderr.Reset()
	code = cli.Main([]string{"app", "add-action", "--descriptor", widgetDescriptor, "--name", "ArchiveWidget", "--path", "archive", "--request", "ArchiveWidgetRequest", "--response", "ArchiveWidgetResponse", "--policy", "archive", "--handler", "ArchiveWidget", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("app add-action failed code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	payload, err := os.ReadFile(widgetDescriptor)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), "ArchiveWidget") {
		t.Fatalf("add-action did not persist descriptor action: %s", payload)
	}
}

func writeDescriptorFixture(t *testing.T, root string, name string, pathSegment string, table string) {
	t.Helper()
	path := filepath.Join(root, pathSegment, "descriptor.axle.json")
	writeDescriptorFile(t, path, name, pathSegment, table)
}

func writeDescriptorFile(t *testing.T, path string, name string, pathSegment string, table string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := fmt.Sprintf(`{
  "schema": "axle.resource.v1",
  "resource": {
    "name": %q,
    "path": %q,
    "table": %q,
    "id": "id",
    "fields": [
      {"name": "id", "type": "text", "mutable": false, "auto": "uuid"},
      {"name": "title", "type": "text", "mutable": true}
    ],
    "operations": [
      {"name": "List%[1]ss", "kind": "list", "request": "List%[1]ssRequest", "response": "List%[1]ssResponse", "policy": "list", "handler": "List%[1]ss"},
      {"name": "Get%[1]s", "kind": "get", "request": "Get%[1]sRequest", "response": "Get%[1]sResponse", "policy": "get", "handler": "Get%[1]s"},
      {"name": "Create%[1]s", "kind": "create", "request": "Create%[1]sRequest", "response": "Create%[1]sResponse", "policy": "create", "handler": "Create%[1]s"},
      {"name": "Update%[1]s", "kind": "update", "request": "Update%[1]sRequest", "response": "Update%[1]sResponse", "policy": "update", "handler": "Update%[1]s"},
      {"name": "Delete%[1]s", "kind": "delete", "request": "Delete%[1]sRequest", "response": "Delete%[1]sResponse", "policy": "delete", "handler": "Delete%[1]s"}
    ]
  },
  "generated": {"package": "generated"}
}
`, name, pathSegment, table)
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
}
