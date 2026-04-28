package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/Fel1xKan/axle/internal/codegen"
	"github.com/Fel1xKan/axle/pkg/axle"
)

func runCatalog(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		printCatalogHelp(stdout)
		return 0
	}
	switch args[0] {
	case "gen":
		return runCatalogGen(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown catalog command: %s\n", args[0])
		printCatalogHelp(stderr)
		return 2
	}
}

func runCatalogGen(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("catalog gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	manifestPath := fs.String("manifest", "", "Path to axle.catalog.json")
	outDir := fs.String("out", "", "Output directory")
	checkOnly := fs.Bool("check", false, "Check generated catalog without writing")
	jsonOut := fs.Bool("json", false, "Render JSON result")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *manifestPath == "" || *outDir == "" {
		fmt.Fprintln(stderr, "catalog gen requires --manifest and --out")
		return 2
	}
	manifest, diagnostics := codegen.LoadCatalog(*manifestPath)
	if len(diagnostics) == 0 {
		files, genDiagnostics := codegen.GenerateCatalog(manifest)
		diagnostics = append(diagnostics, genDiagnostics...)
		if len(genDiagnostics) == 0 {
			if *checkOnly {
				diagnostics = append(diagnostics, codegen.Check(*outDir, files)...)
			} else if err := codegen.Write(*outDir, files); err != nil {
				diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_CATALOG_WRITE", Path: *outDir, Message: err.Error(), SuggestedFix: "Make the catalog output directory writable."})
			}
		}
	}
	return renderResult(stdout, stderr, *jsonOut, diagnostics)
}

func printCatalogHelp(out io.Writer) {
	fmt.Fprintln(out, "axle catalog commands:")
	fmt.Fprintln(out, "  catalog gen --manifest <axle.catalog.json> --out <dir> [--check] [--json]")
}
