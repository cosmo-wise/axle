package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Fel1xKan/axle/internal/check"
	"github.com/Fel1xKan/axle/internal/codegen"
	"github.com/Fel1xKan/axle/internal/descriptor"
	"github.com/Fel1xKan/axle/pkg/axle"
)

func Main(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		printHelp(stdout)
		return 0
	}
	switch args[0] {
	case "gen":
		return runGen(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "catalog":
		return runCatalog(args[1:], stdout, stderr)
	case "app":
		return runApp(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, "axle 0.1.0")
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func runGen(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	descriptorPath := fs.String("descriptor", "", "Path to descriptor JSON")
	outDir := fs.String("out", "", "Output directory")
	checkOnly := fs.Bool("check", false, "Check generated output without writing")
	jsonOut := fs.Bool("json", false, "Render JSON result")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *descriptorPath == "" || *outDir == "" {
		fmt.Fprintln(stderr, "gen requires --descriptor and --out")
		return 2
	}
	desc, diagnostics := descriptor.Load(*descriptorPath)
	if len(diagnostics) == 0 {
		files, genDiagnostics := codegen.Generate(desc)
		diagnostics = append(diagnostics, genDiagnostics...)
		if len(genDiagnostics) == 0 {
			if *checkOnly {
				diagnostics = append(diagnostics, codegen.Check(*outDir, files)...)
			} else if err := codegen.Write(*outDir, files); err != nil {
				diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_GENERATED_WRITE", Path: *outDir, Message: err.Error(), SuggestedFix: "Make the output directory writable."})
			}
		}
	}
	return renderResult(stdout, stderr, *jsonOut, diagnostics)
}

func runCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	descriptorPath := fs.String("descriptor", "", "Path to descriptor JSON")
	root := fs.String("root", ".", "Root to scan")
	jsonOut := fs.Bool("json", false, "Render JSON result")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result := check.Run(*descriptorPath, *root)
	return renderCheckResult(stdout, stderr, *jsonOut, result)
}

func renderResult(stdout io.Writer, stderr io.Writer, jsonOut bool, diagnostics []axle.Diagnostic) int {
	status := "ok"
	if len(diagnostics) > 0 {
		status = "failed"
	}
	return renderCheckResult(stdout, stderr, jsonOut, axle.CheckResult{Status: status, Diagnostics: diagnostics})
}

func renderCheckResult(stdout io.Writer, stderr io.Writer, jsonOut bool, result axle.CheckResult) int {
	if result.Diagnostics == nil {
		result.Diagnostics = []axle.Diagnostic{}
	}
	if jsonOut {
		payload, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(payload))
	} else if result.Status == "ok" {
		fmt.Fprintln(stdout, "ok")
	} else {
		for _, diagnostic := range result.Diagnostics {
			fmt.Fprintf(stderr, "%s %s: %s\n", diagnostic.Code, diagnostic.Path, diagnostic.Message)
		}
	}
	if result.Status == "ok" {
		return 0
	}
	return 1
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, "axle commands:")
	fmt.Fprintln(out, "  gen --descriptor <path> --out <dir> [--check] [--json]")
	fmt.Fprintln(out, "  check --descriptor <path> --root <repo> [--json]")
	fmt.Fprintln(out, "  catalog gen --manifest <axle.catalog.json> --out <dir> [--check] [--json]")
	fmt.Fprintln(out, "  app init --out <dir> [--module <module>] [--axle-replace <path>] [--json]")
	fmt.Fprintln(out, "  version")
}

func MainOS() int { return Main(os.Args[1:], os.Stdout, os.Stderr) }
