package check

import (
	"io/fs"
	"path/filepath"

	"github.com/cosmo-wise/axle/internal/codegen"
	"github.com/cosmo-wise/axle/pkg/axle"
)

func scanCatalogs(root string) []axle.Diagnostic {
	var diagnostics []axle.Diagnostic
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry == nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".cache", "vendor", "generated":
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != codegen.CatalogManifestName {
			return nil
		}
		desc, loadDiags := codegen.LoadCatalog(path)
		diagnostics = append(diagnostics, loadDiags...)
		if len(loadDiags) > 0 {
			return nil
		}
		files, genDiags := codegen.GenerateCatalog(desc)
		diagnostics = append(diagnostics, genDiags...)
		if len(genDiags) == 0 {
			diagnostics = append(diagnostics, codegen.Check(filepath.Dir(path), files)...)
		}
		return nil
	})
	return diagnostics
}
