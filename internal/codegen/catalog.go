package codegen

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cosmo-wise/axle/pkg/axle"
)

const CatalogManifestName = "axle.catalog.json"

// CatalogDescriptor describes a deterministic multi-resource generated catalog.
type CatalogDescriptor struct {
	Package   string            `json:"package"`
	Resources []CatalogResource `json:"resources"`
}

// CatalogResource points at one generated resource package.
type CatalogResource struct {
	Alias      string `json:"alias"`
	ImportPath string `json:"import"`
}

// LoadCatalog reads a catalog manifest from disk.
func LoadCatalog(path string) (CatalogDescriptor, []axle.Diagnostic) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return CatalogDescriptor{}, []axle.Diagnostic{{Code: "AXLE_CATALOG_READ", Path: path, Message: err.Error(), SuggestedFix: "Make the catalog manifest readable."}}
	}
	var desc CatalogDescriptor
	if err := json.Unmarshal(payload, &desc); err != nil {
		return CatalogDescriptor{}, []axle.Diagnostic{{Code: "AXLE_CATALOG_JSON", Path: path, Message: err.Error(), SuggestedFix: "Fix catalog manifest JSON syntax."}}
	}
	return desc, validateCatalog(desc, path)
}

// GenerateCatalog returns a deterministic generated multi-resource catalog.
func GenerateCatalog(desc CatalogDescriptor) ([]GeneratedFile, []axle.Diagnostic) {
	if diags := validateCatalog(desc, CatalogManifestName); len(diags) > 0 {
		return nil, diags
	}
	resources := append([]CatalogResource(nil), desc.Resources...)
	sort.SliceStable(resources, func(i, j int) bool {
		return resources[i].Alias+resources[i].ImportPath < resources[j].Alias+resources[j].ImportPath
	})
	return []GeneratedFile{{Path: "catalog.gen.go", Content: formatGo(renderCatalog(desc.Package, resources))}}, nil
}

func validateCatalog(desc CatalogDescriptor, path string) []axle.Diagnostic {
	var diagnostics []axle.Diagnostic
	add := func(code, pointer, message, fix string) {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: code, Path: path + pointer, Message: message, SuggestedFix: fix})
	}
	if strings.TrimSpace(desc.Package) == "" {
		add("AXLE_CATALOG_PACKAGE", "#/package", "catalog package is required", "Set package, for example catalog.")
	}
	if len(desc.Resources) < 2 {
		add("AXLE_CATALOG_RESOURCES", "#/resources", "catalog requires at least two generated resources", "Add two or more generated resource imports.")
	}
	seen := map[string]bool{}
	for i, resource := range desc.Resources {
		if strings.TrimSpace(resource.Alias) == "" {
			add("AXLE_CATALOG_ALIAS", fmt.Sprintf("#/resources/%d/alias", i), "resource alias is required", "Use a stable Go import alias.")
		}
		if strings.TrimSpace(resource.ImportPath) == "" {
			add("AXLE_CATALOG_IMPORT", fmt.Sprintf("#/resources/%d/import", i), "resource import path is required", "Point at a generated resource package.")
		}
		if seen[resource.Alias] {
			add("AXLE_CATALOG_DUPLICATE", fmt.Sprintf("#/resources/%d/alias", i), "resource alias is duplicated", "Use unique aliases.")
		}
		seen[resource.Alias] = true
	}
	return diagnostics
}

func renderCatalog(packageName string, resources []CatalogResource) string {
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("package " + packageName + "\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"github.com/cosmo-wise/axle/pkg/axle\"\n")
	for _, resource := range resources {
		b.WriteString(fmt.Sprintf("\t%s %q\n", resource.Alias, resource.ImportPath))
	}
	b.WriteString(")\n\n")
	b.WriteString("var Catalog = axle.Catalog{Resources: []axle.ResourceRegistry{\n")
	for _, resource := range resources {
		b.WriteString("\t" + resource.Alias + ".ResourceRegistry,\n")
	}
	b.WriteString("}}\n")
	return b.String()
}
