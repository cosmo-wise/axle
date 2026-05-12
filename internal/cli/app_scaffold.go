package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmo-wise/axle/pkg/axle"
)

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
