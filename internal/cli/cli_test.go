package cli_test

import (
	"bytes"
	"encoding/json"
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
