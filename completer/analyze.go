package completer

import (
	"go/ast"
	"go/parser"
	"go/token"
)

func analyze(path string) (map[string]*ast.Package, error) {
	fset := token.NewFileSet()
	mode := parser.ParseComments | parser.AllErrors
	node, err := parser.ParseDir(fset, path, nil, mode)
	if err != nil {
		return nil, err
	}
	return node, nil
}
