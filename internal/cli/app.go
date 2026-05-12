package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmo-wise/axle/internal/descriptor"
	"github.com/cosmo-wise/axle/pkg/axle"
)

type appInitResult struct {
	Status      string            `json:"status"`
	Module      string            `json:"module"`
	Out         string            `json:"out"`
	Readme      string            `json:"readme"`
	Descriptor  string            `json:"descriptor"`
	Descriptors []string          `json:"descriptors"`
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
	case "add-resource":
		return runAppAddResource(args[1:], stdout, stderr)
	case "add-action":
		return runAppAddAction(args[1:], stdout, stderr)
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
	axleReplace := fs.String("axle-replace", "../..", "Go replace target for github.com/cosmo-wise/axle")
	descriptorsDir := fs.String("descriptors-dir", "", "Directory containing descriptor.axle.json files to seed the scaffold")
	jsonOut := fs.Bool("json", false, "Render JSON result")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *outDir == "" {
		fmt.Fprintln(stderr, "app init requires --out")
		return 2
	}
	result, diagnostics := writeAppBackend(*outDir, *moduleName, *axleReplace, *descriptorsDir)
	return renderAppInitResult(stdout, stderr, *jsonOut, result, diagnostics)
}

func runAppAddResource(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("app add-resource", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outDir := fs.String("out", "", "Backend directory created by app init")
	moduleName := fs.String("module", "", "Generated backend Go module; read from go.mod when omitted")
	descriptorPath := fs.String("descriptor", "", "Descriptor JSON to add")
	jsonOut := fs.Bool("json", false, "Render JSON result")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *outDir == "" || *descriptorPath == "" {
		fmt.Fprintln(stderr, "app add-resource requires --out and --descriptor")
		return 2
	}
	module := *moduleName
	if module == "" {
		module = readModuleName(filepath.Join(*outDir, "go.mod"))
	}
	result := newAppResult(*outDir, module)
	paths, collectDiagnostics := existingDescriptorPaths(*outDir)
	if len(collectDiagnostics) > 0 {
		return renderAppInitResult(stdout, stderr, *jsonOut, result, collectDiagnostics)
	}
	paths = append(paths, *descriptorPath)
	diagnostics := installDescriptors(*outDir, module, paths)
	return renderAppInitResult(stdout, stderr, *jsonOut, result, diagnostics)
}

func runAppAddAction(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("app add-action", flag.ContinueOnError)
	fs.SetOutput(stderr)
	descriptorPath := fs.String("descriptor", "", "Descriptor JSON to update")
	name := fs.String("name", "", "Action operation name")
	path := fs.String("path", "", "Relative action path")
	request := fs.String("request", "", "Action request type")
	response := fs.String("response", "", "Action response type")
	policy := fs.String("policy", "", "Action policy")
	handler := fs.String("handler", "", "Action handler name")
	jsonOut := fs.Bool("json", false, "Render JSON result")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	missing := []string{}
	for key, value := range map[string]string{"--descriptor": *descriptorPath, "--name": *name, "--path": *path, "--request": *request, "--response": *response, "--policy": *policy, "--handler": *handler} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(stderr, "app add-action requires %s\n", strings.Join(missing, ", "))
		return 2
	}
	desc, diagnostics := descriptor.Load(*descriptorPath)
	if len(diagnostics) == 0 {
		desc.Resource.Actions = append(desc.Resource.Actions, axle.OperationDescriptor{Name: *name, Kind: "action", Path: *path, Request: *request, Response: *response, Policy: *policy, Handler: *handler})
		diagnostics = descriptor.Validate(desc, *descriptorPath)
	}
	if len(diagnostics) == 0 {
		payload, err := json.MarshalIndent(desc, "", "  ")
		if err != nil {
			diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_APP_ACTION_JSON", Path: *descriptorPath, Message: err.Error(), SuggestedFix: "Make descriptor data JSON serializable."})
		} else if err := os.WriteFile(*descriptorPath, append(payload, '\n'), 0o644); err != nil {
			diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_APP_ACTION_WRITE", Path: *descriptorPath, Message: err.Error(), SuggestedFix: "Make descriptor path writable."})
		}
	}
	return renderResult(stdout, stderr, *jsonOut, diagnostics)
}

func newAppResult(outDir string, moduleName string) appInitResult {
	return appInitResult{
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
}

func printAppHelp(out io.Writer) {
	fmt.Fprintln(out, "axle app commands:")
	fmt.Fprintln(out, "  app init --out <dir> [--module <module>] [--axle-replace <path>] [--descriptors-dir <dir>] [--json]")
	fmt.Fprintln(out, "  app add-resource --out <dir> --descriptor <descriptor.axle.json> [--module <module>] [--json]")
	fmt.Fprintln(out, "  app add-action --descriptor <descriptor.axle.json> --name <Name> --path <path> --request <Req> --response <Resp> --policy <policy> --handler <Handler> [--json]")
}
