package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cosmo-wise/axle/internal/codegen"
	"github.com/cosmo-wise/axle/internal/descriptor"
	"github.com/cosmo-wise/axle/pkg/axle"
)

// Run executes descriptor, generated, and architecture checks.
func Run(descriptorPath, root string) axle.CheckResult {
	var diagnostics []axle.Diagnostic
	if descriptorPath != "" {
		desc, descDiags := descriptor.Load(descriptorPath)
		diagnostics = append(diagnostics, descDiags...)
		if len(descDiags) == 0 {
			files, genDiags := codegen.Generate(desc)
			diagnostics = append(diagnostics, genDiags...)
			out := filepath.Join(filepath.Dir(descriptorPath), "generated")
			diagnostics = append(diagnostics, codegen.Check(out, files)...)
		}
	}
	if root != "" {
		diagnostics = append(diagnostics, scanArchitecture(root)...)
		diagnostics = append(diagnostics, scanCatalogs(root)...)
	}
	sort.SliceStable(diagnostics, func(i, j int) bool {
		return diagnostics[i].Code+diagnostics[i].Path < diagnostics[j].Code+diagnostics[j].Path
	})
	status := "ok"
	if len(diagnostics) > 0 {
		status = "failed"
	}
	return axle.CheckResult{Status: status, Diagnostics: diagnostics}
}

func scanArchitecture(root string) []axle.Diagnostic {
	var diagnostics []axle.Diagnostic
	rootSlash := filepath.ToSlash(root)
	scanTestdata := strings.Contains(rootSlash, "testdata/fixtures/negative")
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry == nil {
			return nil
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == ".cache" || name == "vendor" || name == "generated" {
				return filepath.SkipDir
			}
			if name == "testdata" && !scanTestdata {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, ".gen.go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(path)
		if rootRel, err := filepath.Rel(root, path); err == nil {
			rel = filepath.ToSlash(rootRel)
		}
		diagnostics = append(diagnostics, scanGoFile(path, rel)...)
		return nil
	})
	return diagnostics
}

func scanGoFile(path string, rel string) []axle.Diagnostic {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil
	}
	imports := importPaths(file)
	lower := strings.ToLower(rel)
	var diagnostics []axle.Diagnostic
	if strings.Contains(lower, "controller") && (hasImport(imports, "database/sql") || hasImportSuffix(imports, "/internal/sqlite")) {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_BOUNDARY_CONTROLLER_DB", Path: rel, Message: "controller-like code accesses storage directly", SuggestedFix: "Move SQLite access behind a service/handler binding."})
	}
	if strings.Contains(lower, "service") && hasImport(imports, "net/http") {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_BOUNDARY_SERVICE_HTTP", Path: rel, Message: "service-like code writes HTTP responses", SuggestedFix: "Return structured values and render HTTP at the edge."})
	}
	if strings.HasPrefix(rel, "pkg/axle/") && importsAppOrFixture(imports) {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_PUBLIC_IMPORT_INTERNAL", Path: rel, Message: "public Axle API imports app or fixture internals", SuggestedFix: "Keep pkg/axle limited to stable generated-code contracts."})
	}
	if isRootAxlePublicFile(rel) && (hasForbiddenRootAPIImport(imports) || declaresForbiddenRootAPIType(file)) {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_PUBLIC_API_BLOAT", Path: rel, Message: "root pkg/axle must stay metadata-only", SuggestedFix: "Move HTTP, SQLite, or SQL behavior into narrow subpackages such as pkg/axle/runtime or pkg/axle/sqlite."})
	}
	if declaresMultiDBAbstraction(file) {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_MULTIDB_ABSTRACTION", Path: rel, Message: "V1 must not introduce a generic multi-database abstraction", SuggestedFix: "Use concrete SQLite support only in V1."})
	}
	if declaresTypedORM(file) {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_TYPED_ORM_CREEP", Path: rel, Message: "typed repositories or query builders are outside Axle V1", SuggestedFix: "Keep persistence behind the concrete SQLite CRUD facade and generated DTO edge."})
	}
	if declaresManualCRUDRouting(file, imports) {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_MANUAL_CRUD_ROUTING", Path: rel, Message: "manual standard CRUD route switches duplicate generated Axle routes", SuggestedFix: "Mount generated Catalog with pkg/axle/runtime and implement only custom action handlers."})
	}
	if hasImport(imports, "reflect") || runtimeWalkDiscovery(lower, file) {
		diagnostics = append(diagnostics, axle.Diagnostic{Code: "AXLE_RUNTIME_DISCOVERY", Path: rel, Message: "runtime discovery or reflection registration is not allowed in V1", SuggestedFix: "Use descriptor-driven generated registrations."})
	}
	return diagnostics
}

func importPaths(file *ast.File) []string {
	paths := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		paths = append(paths, strings.Trim(spec.Path.Value, "\""))
	}
	return paths
}

func hasImport(imports []string, want string) bool {
	for _, path := range imports {
		if path == want {
			return true
		}
	}
	return false
}

func hasImportSuffix(imports []string, suffix string) bool {
	for _, path := range imports {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func isRootAxlePublicFile(rel string) bool {
	if !strings.HasPrefix(rel, "pkg/axle/") {
		return false
	}
	rest := strings.TrimPrefix(rel, "pkg/axle/")
	return !strings.Contains(rest, "/")
}

func declaresForbiddenRootAPIType(file *ast.File) bool {
	forbidden := map[string]bool{
		"Record": true, "Store": true, "Repository": true, "QueryBuilder": true,
		"Server": true, "Router": true, "Handler": true,
		"Database": true, "DB": true, "DatabaseDriver": true, "Dialect": true, "DialectPlugin": true,
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		typeSpec, ok := node.(*ast.TypeSpec)
		if ok && forbidden[typeSpec.Name.Name] {
			found = true
		}
		return !found
	})
	return found
}

func hasForbiddenRootAPIImport(imports []string) bool {
	for _, path := range imports {
		if path == "net/http" || path == "database/sql" || strings.HasSuffix(path, "/pkg/axle/runtime") || strings.HasSuffix(path, "/pkg/axle/sqlite") {
			return true
		}
	}
	return false
}

func importsAppOrFixture(imports []string) bool {
	for _, path := range imports {
		if strings.Contains(path, "testdata") || strings.Contains(path, "/fixtures") || strings.Contains(path, "/app/") {
			return true
		}
	}
	return false
}

func declaresMultiDBAbstraction(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.TypeSpec:
			if item.Name.Name == "DatabaseDriver" || item.Name.Name == "DialectPlugin" || item.Name.Name == "Dialect" {
				found = true
			}
		case *ast.FuncDecl:
			if item.Name.Name == "RegisterDialect" || item.Name.Name == "RegisterDriver" {
				found = true
			}
		}
		return !found
	})
	return found
}

func runtimeWalkDiscovery(rel string, file *ast.File) bool {
	if !strings.Contains(rel, "runtime") && !strings.Contains(rel, "registry") {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "WalkDir" {
			found = true
		}
		return !found
	})
	return found
}
