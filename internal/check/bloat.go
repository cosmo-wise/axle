package check

import (
	"go/ast"
	"go/token"
	"strings"
)

func declaresTypedORM(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.TypeSpec:
			name := item.Name.Name
			if strings.HasSuffix(name, "Repository") || strings.HasSuffix(name, "QueryBuilder") || strings.HasSuffix(name, "RelationLoader") {
				found = true
			}
		case *ast.FuncDecl:
			name := item.Name.Name
			if strings.HasPrefix(name, "Query") && strings.HasSuffix(name, "Relations") {
				found = true
			}
		}
		return !found
	})
	return found
}

func declaresManualCRUDRouting(file *ast.File, imports []string) bool {
	if !hasImport(imports, "net/http") {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		item, ok := node.(*ast.BasicLit)
		if !ok || item.Kind != token.STRING {
			return true
		}
		if isManualCRUDRouteLiteral(strings.Trim(item.Value, "`\"")) {
			found = true
		}
		return !found
	})
	return found
}

func isManualCRUDRouteLiteral(value string) bool {
	if !strings.HasPrefix(value, "/") {
		return false
	}
	trimmed := strings.Trim(value, "/")
	if trimmed == "" {
		return false
	}
	parts := strings.Split(trimmed, "/")
	last := parts[len(parts)-1]
	if last != "update" && last != "delete" {
		return false
	}
	if len(parts) < 3 {
		return false
	}
	for _, part := range parts[:len(parts)-1] {
		if part == "{id}" {
			return true
		}
	}
	return false
}
