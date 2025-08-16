package utils

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"

	"github.com/kakkky/gonsole/errs"
)

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func AnalyzeGoAst(path string) (map[string][]*ast.Package, *token.FileSet, error) {
	fset := token.NewFileSet()
	mode := parser.ParseComments | parser.AllErrors
	// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
	nodes := make(map[string][]*ast.Package)
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "vendor" {
			return filepath.SkipDir
		}
		node, err := parser.ParseDir(fset, path, nil, mode)
		for pkgName, pkg := range node {
			nodes[pkgName] = append(nodes[pkgName], pkg)
		}
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, nil, errs.NewInternalError("failed to walk directory").Wrap(err)
	}
	return nodes, fset, nil
}
