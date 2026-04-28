package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Fel1xKan/axle/internal/codegen"
	"github.com/Fel1xKan/axle/internal/descriptor"
	"github.com/Fel1xKan/axle/pkg/axle"
)

type appInitResult struct {
	Status      string            `json:"status"`
	Module      string            `json:"module"`
	Out         string            `json:"out"`
	Readme      string            `json:"readme"`
	Descriptor  string            `json:"descriptor"`
	Generated   string            `json:"generated"`
	Catalog     string            `json:"catalog"`
	Verify      string            `json:"verify"`
	NextSteps   []string          `json:"next_steps"`
	Diagnostics []axle.Diagnostic `json:"diagnostics"`
}

func runApp(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		printAppHelp(stdout)
		return 0
	}
	switch args[0] {
	case "init":
		return runAppInit(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown app command: %s\n", args[0])
		printAppHelp(stderr)
		return 2
	}
}

func runAppInit(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("app init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outDir := fs.String("out", "", "Output backend directory")
	moduleName := fs.String("module", "example.com/axle-generated-backend", "Generated backend Go module")
	axleReplace := fs.String("axle-replace", "../..", "Go replace target for github.com/Fel1xKan/axle")
	jsonOut := fs.Bool("json", false, "Render JSON result")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *outDir == "" {
		fmt.Fprintln(stderr, "app init requires --out")
		return 2
	}
	result, diagnostics := writeAppBackend(*outDir, *moduleName, *axleReplace)
	if diagnostics == nil {
		diagnostics = []axle.Diagnostic{}
	}
	result.Diagnostics = diagnostics
	if len(diagnostics) > 0 {
		result.Status = "failed"
	}
	if *jsonOut {
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

func writeAppBackend(outDir string, moduleName string, axleReplace string) (appInitResult, []axle.Diagnostic) {
	result := appInitResult{
		Status:     "ok",
		Module:     moduleName,
		Out:        outDir,
		Readme:     filepath.ToSlash(filepath.Join(outDir, "README.md")),
		Descriptor: filepath.ToSlash(filepath.Join(outDir, "descriptors", "resources", "descriptor.axle.json")),
		Generated:  filepath.ToSlash(filepath.Join(outDir, "descriptors")),
		Catalog:    filepath.ToSlash(filepath.Join(outDir, "catalog", "catalog.gen.go")),
		Verify:     filepath.ToSlash(filepath.Join(outDir, "scripts", "verify.sh")),
		NextSteps: []string{
			"Read README.md for the scaffold ownership map.",
			"Replace sample descriptors with project resources before editing generated files.",
			"Run scripts/verify.sh after descriptor, catalog, or action changes.",
		},
		Diagnostics: []axle.Diagnostic{},
	}
	files := map[string]string{
		"README.md":                   renderAppReadme(moduleName),
		"go.mod":                      renderAppGoMod(moduleName, axleReplace),
		"cmd/example-backend/main.go": renderAppMain(moduleName),
		"internal/app/app.go":         renderAppPackage(moduleName),
		"internal/app/app_test.go":    renderAppTest(moduleName),
		"scripts/verify.sh":           renderVerifyScript(),
		"descriptors/resources/descriptor.axle.json":                            appResourceDescriptorJSON,
		"descriptors/policies/descriptor.axle.json":                             appPolicyDescriptorJSON,
		filepath.ToSlash(filepath.Join("catalog", codegen.CatalogManifestName)): renderAppCatalogManifest(moduleName),
	}
	for rel, content := range files {
		if diagnostics := writeAppFile(outDir, rel, content); len(diagnostics) > 0 {
			return result, diagnostics
		}
	}
	for _, rel := range []string{"descriptors/resources", "descriptors/policies"} {
		descPath := filepath.Join(outDir, filepath.FromSlash(rel), "descriptor.axle.json")
		desc, descDiags := descriptor.Load(descPath)
		if len(descDiags) > 0 {
			return result, descDiags
		}
		generated, genDiags := codegen.Generate(desc)
		if len(genDiags) > 0 {
			return result, genDiags
		}
		if err := codegen.Write(filepath.Join(outDir, filepath.FromSlash(rel), "generated"), generated); err != nil {
			return result, []axle.Diagnostic{{Code: "AXLE_APP_INIT_GENERATE", Path: rel, Message: err.Error(), SuggestedFix: "Make the generated output path writable."}}
		}
	}
	catalogPath := filepath.Join(outDir, "catalog", codegen.CatalogManifestName)
	catalog, catalogDiags := codegen.LoadCatalog(catalogPath)
	if len(catalogDiags) > 0 {
		return result, catalogDiags
	}
	catalogFiles, genDiags := codegen.GenerateCatalog(catalog)
	if len(genDiags) > 0 {
		return result, genDiags
	}
	if err := codegen.Write(filepath.Dir(catalogPath), catalogFiles); err != nil {
		return result, []axle.Diagnostic{{Code: "AXLE_APP_INIT_CATALOG", Path: filepath.Dir(catalogPath), Message: err.Error(), SuggestedFix: "Make the catalog output path writable."}}
	}
	return result, nil
}

func writeAppFile(outDir string, rel string, content string) []axle.Diagnostic {
	path := filepath.Join(outDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return []axle.Diagnostic{{Code: "AXLE_APP_INIT_WRITE", Path: path, Message: err.Error(), SuggestedFix: "Make the backend output path writable."}}
	}
	mode := os.FileMode(0o644)
	if rel == "scripts/verify.sh" {
		mode = 0o755
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return []axle.Diagnostic{{Code: "AXLE_APP_INIT_WRITE", Path: path, Message: err.Error(), SuggestedFix: "Make the backend output path writable."}}
	}
	return nil
}

func printAppHelp(out io.Writer) {
	fmt.Fprintln(out, "axle app commands:")
	fmt.Fprintln(out, "  app init --out <dir> [--module <module>] [--axle-replace <path>] [--json]")
}
