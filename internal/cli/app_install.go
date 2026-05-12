package cli

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cosmo-wise/axle/internal/codegen"
	"github.com/cosmo-wise/axle/internal/descriptor"
	"github.com/cosmo-wise/axle/pkg/axle"
)

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
