package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmo-wise/axle/internal/codegen"
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

func existingDescriptorPaths(outDir string) ([]string, []axle.Diagnostic) {
	root := filepath.Join(outDir, "descriptors")
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) == "descriptor.axle.json" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, []axle.Diagnostic{{Code: "AXLE_APP_DESCRIPTOR_WALK", Path: root, Message: err.Error(), SuggestedFix: "Run app init before add-resource or make descriptors readable."}}
	}
	return paths, nil
}

func writeAppBackend(outDir string, moduleName string, axleReplace string, descriptorsDir string) (appInitResult, []axle.Diagnostic) {
	result := newAppResult(outDir, moduleName)
	files := map[string]string{
		"README.md":                   renderAppReadme(moduleName),
		"go.mod":                      renderAppGoMod(moduleName, axleReplace),
		"cmd/example-backend/main.go": renderAppMain(moduleName),
		"internal/app/app.go":         renderAppPackage(moduleName),
		"internal/app/app_test.go":    renderAppTest(moduleName),
		"scripts/verify.sh":           renderVerifyScript(),
	}
	for rel, content := range files {
		if diagnostics := writeAppFile(outDir, rel, content); len(diagnostics) > 0 {
			return result, diagnostics
		}
	}
	paths, diagnostics := descriptorSeeds(descriptorsDir)
	if len(diagnostics) > 0 {
		return result, diagnostics
	}
	if diagnostics := installDescriptors(outDir, moduleName, paths); len(diagnostics) > 0 {
		return result, diagnostics
	}
	result.Descriptors = descriptorResultPaths(outDir, paths)
	if len(result.Descriptors) > 0 {
		result.Descriptor = result.Descriptors[0]
	}
	return result, nil
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

func descriptorSeeds(descriptorsDir string) ([]string, []axle.Diagnostic) {
	if strings.TrimSpace(descriptorsDir) == "" {
		return defaultDescriptorSeeds()
	}
	var paths []string
	err := filepath.WalkDir(descriptorsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) == "descriptor.axle.json" || strings.HasSuffix(path, ".axle.json") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, []axle.Diagnostic{{Code: "AXLE_APP_DESCRIPTOR_WALK", Path: descriptorsDir, Message: err.Error(), SuggestedFix: "Make descriptors-dir readable."}}
	}
	if len(paths) == 0 {
		return nil, []axle.Diagnostic{{Code: "AXLE_APP_DESCRIPTOR_EMPTY", Path: descriptorsDir, Message: "no descriptor.axle.json files found", SuggestedFix: "Provide one or more Axle descriptor files."}}
	}
	return paths, nil
}

func defaultDescriptorSeeds() ([]string, []axle.Diagnostic) {
	dir, err := os.MkdirTemp("", "axle-default-descriptors-")
	if err != nil {
		return nil, []axle.Diagnostic{{Code: "AXLE_APP_DESCRIPTOR_TEMP", Path: os.TempDir(), Message: err.Error(), SuggestedFix: "Make temporary directory writable."}}
	}
	seeds := map[string]string{"resources/descriptor.axle.json": appResourceDescriptorJSON, "policies/descriptor.axle.json": appPolicyDescriptorJSON}
	var paths []string
	for rel, content := range seeds {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, []axle.Diagnostic{{Code: "AXLE_APP_DESCRIPTOR_TEMP", Path: path, Message: err.Error(), SuggestedFix: "Make temporary directory writable."}}
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, []axle.Diagnostic{{Code: "AXLE_APP_DESCRIPTOR_TEMP", Path: path, Message: err.Error(), SuggestedFix: "Make temporary descriptor writable."}}
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func installDescriptors(outDir string, moduleName string, paths []string) []axle.Diagnostic {
	manifest := codegen.CatalogDescriptor{Package: "catalog"}
	for _, source := range paths {
		desc, diagnostics := descriptor.Load(source)
		if len(diagnostics) > 0 {
			return diagnostics
		}
		slug := resourceSlug(desc.Resource)
		destDir := filepath.Join(outDir, "descriptors", slug)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return []axle.Diagnostic{{Code: "AXLE_APP_DESCRIPTOR_WRITE", Path: destDir, Message: err.Error(), SuggestedFix: "Make descriptors directory writable."}}
		}
		payload, _ := json.MarshalIndent(desc, "", "  ")
		if err := os.WriteFile(filepath.Join(destDir, "descriptor.axle.json"), append(payload, '\n'), 0o644); err != nil {
			return []axle.Diagnostic{{Code: "AXLE_APP_DESCRIPTOR_WRITE", Path: destDir, Message: err.Error(), SuggestedFix: "Make descriptor path writable."}}
		}
		generated, genDiags := codegen.Generate(desc)
		if len(genDiags) > 0 {
			return genDiags
		}
		if err := codegen.Write(filepath.Join(destDir, "generated"), generated); err != nil {
			return []axle.Diagnostic{{Code: "AXLE_APP_INIT_GENERATE", Path: destDir, Message: err.Error(), SuggestedFix: "Make the generated output path writable."}}
		}
		manifest.Resources = append(manifest.Resources, codegen.CatalogResource{Alias: importAlias(slug), ImportPath: moduleName + "/descriptors/" + slug + "/generated"})
	}
	if len(manifest.Resources) < 2 {
		return []axle.Diagnostic{{Code: "AXLE_APP_DESCRIPTOR_COUNT", Path: outDir, Message: "app scaffold requires at least two resources so catalog generation stays explicit", SuggestedFix: "Add at least two descriptors or use the default scaffold."}}
	}
	catalogDir := filepath.Join(outDir, "catalog")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		return []axle.Diagnostic{{Code: "AXLE_APP_CATALOG_WRITE", Path: catalogDir, Message: err.Error(), SuggestedFix: "Make catalog directory writable."}}
	}
	payload, _ := json.MarshalIndent(manifest, "", "  ")
	catalogPath := filepath.Join(catalogDir, codegen.CatalogManifestName)
	if err := os.WriteFile(catalogPath, append(payload, '\n'), 0o644); err != nil {
		return []axle.Diagnostic{{Code: "AXLE_APP_CATALOG_WRITE", Path: catalogPath, Message: err.Error(), SuggestedFix: "Make catalog manifest writable."}}
	}
	catalogFiles, genDiags := codegen.GenerateCatalog(manifest)
	if len(genDiags) > 0 {
		return genDiags
	}
	if err := codegen.Write(catalogDir, catalogFiles); err != nil {
		return []axle.Diagnostic{{Code: "AXLE_APP_INIT_CATALOG", Path: catalogDir, Message: err.Error(), SuggestedFix: "Make the catalog output path writable."}}
	}
	return nil
}

func descriptorResultPaths(outDir string, seeds []string) []string {
	var out []string
	for _, source := range seeds {
		desc, diagnostics := descriptor.Load(source)
		if len(diagnostics) > 0 {
			continue
		}
		out = append(out, filepath.ToSlash(filepath.Join(outDir, "descriptors", resourceSlug(desc.Resource), "descriptor.axle.json")))
	}
	return out
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

func resourceSlug(resource axle.ResourceDescriptor) string {
	value := strings.Trim(strings.TrimSpace(resource.Path), "/")
	if value == "" {
		value = resource.Name
	}
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, "/", "_")
	return strings.ToLower(value)
}

func importAlias(slug string) string {
	parts := strings.FieldsFunc(slug, func(r rune) bool { return r == '_' || r == '-' || r == '/' })
	if len(parts) == 0 {
		return "resource"
	}
	return strings.Join(parts, "")
}

func readModuleName(goModPath string) string {
	payload, err := os.ReadFile(goModPath)
	if err != nil {
		return "example.com/axle-generated-backend"
	}
	for _, line := range strings.Split(string(payload), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return "example.com/axle-generated-backend"
}

func printAppHelp(out io.Writer) {
	fmt.Fprintln(out, "axle app commands:")
	fmt.Fprintln(out, "  app init --out <dir> [--module <module>] [--axle-replace <path>] [--descriptors-dir <dir>] [--json]")
	fmt.Fprintln(out, "  app add-resource --out <dir> --descriptor <descriptor.axle.json> [--module <module>] [--json]")
	fmt.Fprintln(out, "  app add-action --descriptor <descriptor.axle.json> --name <Name> --path <path> --request <Req> --response <Resp> --policy <policy> --handler <Handler> [--json]")
}
