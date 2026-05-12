package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/cosmo-wise/axle/pkg/axle"
)

func renderAppInitResult(stdout io.Writer, stderr io.Writer, jsonOut bool, result appInitResult, diagnostics []axle.Diagnostic) int {
	if diagnostics == nil {
		diagnostics = []axle.Diagnostic{}
	}
	result.Diagnostics = diagnostics
	if len(diagnostics) > 0 {
		result.Status = "failed"
	}
	if jsonOut {
		payload, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(payload))
	} else if result.Status == "ok" {
		fmt.Fprintf(stdout, "created backend at %s\n", result.Out)
	} else {
		for _, diagnostic := range diagnostics {
			fmt.Fprintf(stderr, "%s %s: %s\n", diagnostic.Code, diagnostic.Path, diagnostic.Message)
		}
	}
	if result.Status == "ok" {
		return 0
	}
	return 1
}
