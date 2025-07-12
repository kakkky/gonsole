package completer

import (
	"go/ast"
	"go/token"
)

func convertFromNodeToCandidates(node map[string]*ast.Package) candidates {
	c := candidates{
		pkgs:    make([]pkgName, 0, len(node)),
		funcs:   make(map[pkgName][]funcName),
		methods: make(map[pkgName][]receiverMap),
		vars:    make(map[pkgName][]varName),
		consts:  make(map[pkgName][]constName),
		types:   make(map[pkgName][]typeName),
	}

	for pkg, pkgAst := range node {
		c.pkgs = append(c.pkgs, pkgName(pkg))
		c.processPackageAst(pkg, pkgAst)
	}

	return c
}

func (c *candidates) processPackageAst(pkg string, pkgAst *ast.Package) {
	for _, fileAst := range pkgAst.Files {
		c.processFileAst(pkg, fileAst)
	}
}

func (c *candidates) processFileAst(pkg string, fileAst *ast.File) {
	for _, decl := range fileAst.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if isMethod(d) {
				c.processMethodDecl(pkg, d)
				continue
			}
			c.processFuncDecl(pkg, d)
		case *ast.GenDecl:
			c.processGenDecl(pkg, d)
		}
	}
}

func (c *candidates) processFuncDecl(pkg string, funcDecl *ast.FuncDecl) {
	c.funcs[pkgName(pkg)] = append(c.funcs[pkgName(pkg)], funcName(funcDecl.Name.Name))
}

func isMethod(funcDecl *ast.FuncDecl) bool {
	return funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0
}

func (c *candidates) processMethodDecl(pkg string, funcDecl *ast.FuncDecl) {
	var receiverTypeName string
	switch receiverType := funcDecl.Recv.List[0].Type.(type) {
	case *ast.StarExpr, *ast.Ident:
		receiverTypeName = receiverType.(*ast.Ident).Name
	}
	// receiverMapが存在しない場合は初期化
	if c.methods[pkgName(pkg)] == nil {
		c.methods[pkgName(pkg)] = make([]receiverMap, 0)
	}
	c.methods[pkgName(pkg)] = append(c.methods[pkgName(pkg)], receiverMap{typeName(receiverTypeName): methodName(funcDecl.Name.Name)})
}

func (c *candidates) processGenDecl(pkg string, genDecl *ast.GenDecl) {
	switch genDecl.Tok {
	case token.VAR:
		c.processVarDecl(pkg, genDecl)
	case token.CONST:
		c.processConstDecl(pkg, genDecl)
	case token.TYPE:
		c.processTypeDecl(pkg, genDecl)
	}
}

func (c *candidates) processVarDecl(pkg string, genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		varspec := spec.(*ast.ValueSpec)
		for _, varname := range varspec.Names {
			c.vars[pkgName(pkg)] = append(c.vars[pkgName(pkg)], varName(varname.Name))
		}
	}
}

func (c *candidates) processConstDecl(pkg string, genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		constspec := spec.(*ast.ValueSpec)
		for _, constname := range constspec.Names {
			c.consts[pkgName(pkg)] = append(c.consts[pkgName(pkg)], constName(constname.Name))
		}
	}
}
func (c *candidates) processTypeDecl(pkg string, genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		typespec := spec.(*ast.TypeSpec)
		c.types[pkgName(pkg)] = append(c.types[pkgName(pkg)], typeName(typespec.Name.Name))
	}
}
