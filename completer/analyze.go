package completer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
)

func analyze(path string) (map[string]*ast.Package, error) {
	fset := token.NewFileSet()
	mode := parser.ParseComments | parser.AllErrors
	nodes := make(map[string]*ast.Package)
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		node, err := parser.ParseDir(fset, path, nil, mode)
		for pkgName, pkg := range node {
			nodes[pkgName] = pkg
		}
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return nodes, nil
}
