package completer

import (
	"go/ast"
	"go/token"
	"strings"
)

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func ConvertFromNodeToCandidates(node map[string]*ast.Package) *candidates {
	c := candidates{
		pkgs:    make([]pkgName, 0),
		funcs:   make(map[pkgName][]funcSet),
		methods: make(map[pkgName][]methodSet),
		vars:    make(map[pkgName][]varSet),
		consts:  make(map[pkgName][]constSet),
		structs: make(map[pkgName][]structSet),
	}

	for pkg, pkgAst := range node {
		c.processPackageAst(pkg, pkgAst)
	}

	return &c
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func (c *candidates) processPackageAst(pkg string, pkgAst *ast.Package) {
	c.pkgs = append(c.pkgs, pkgName(pkg))
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
	var description string
	if funcDecl.Doc != nil {
		description = strings.ReplaceAll(funcDecl.Doc.Text(), "\n", "")
	}
	c.funcs[pkgName(pkg)] = append(c.funcs[pkgName(pkg)], funcSet{name: funcDecl.Name.Name, description: description})
}

func isMethod(funcDecl *ast.FuncDecl) bool {
	return funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0
}

func (c *candidates) processMethodDecl(pkg string, funcDecl *ast.FuncDecl) {
	var receiverTypeName string
	switch receiverType := funcDecl.Recv.List[0].Type.(type) {
	case *ast.Ident:
		receiverTypeName = receiverType.Name
	case *ast.StarExpr:
		if ident, ok := receiverType.X.(*ast.Ident); ok {
			receiverTypeName = ident.Name
		}
	}
	var description string
	if funcDecl.Doc != nil {
		description = strings.ReplaceAll(funcDecl.Doc.Text(), "\n", "")
	}
	c.methods[pkgName(pkg)] = append(c.methods[pkgName(pkg)], methodSet{name: funcDecl.Name.Name, description: description, receiverTypeName: receiverTypeName})
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
	var genDeclDescription string
	if genDecl.Doc != nil {
		genDeclDescription += strings.TrimSpace(genDecl.Doc.Text())
	}
	for _, spec := range genDecl.Specs {
		varspec := spec.(*ast.ValueSpec)
		var specDescription string
		if varspec.Doc != nil {
			specDescription += "   " + strings.TrimSpace(varspec.Doc.Text())
		}
		for _, varname := range varspec.Names {
			c.vars[pkgName(pkg)] = append(c.vars[pkgName(pkg)], varSet{name: varname.Name, description: genDeclDescription + specDescription})
		}
	}
}

func (c *candidates) processConstDecl(pkg string, genDecl *ast.GenDecl) {
	var genDeclDescription string
	if genDecl.Doc != nil {
		genDeclDescription += strings.TrimSpace(genDecl.Doc.Text())
	}
	for _, spec := range genDecl.Specs {
		var specDescription string
		constspec := spec.(*ast.ValueSpec)
		if constspec.Doc != nil {
			specDescription += "   " + strings.TrimSpace(constspec.Doc.Text())
		}
		for _, constname := range constspec.Names {
			c.consts[pkgName(pkg)] = append(c.consts[pkgName(pkg)], constSet{name: constname.Name, description: genDeclDescription + specDescription})
		}
	}
}
func (c *candidates) processTypeDecl(pkg string, genDecl *ast.GenDecl) {
	var genDeclDescription string
	if genDecl.Doc != nil {
		genDeclDescription += strings.TrimSpace(genDecl.Doc.Text())
	}
	for _, spec := range genDecl.Specs {
		typespec := spec.(*ast.TypeSpec)
		var fields []string
		structType, ok := typespec.Type.(*ast.StructType)
		if ok {
			for _, field := range structType.Fields.List {
				if len(field.Names) > 0 {
					fields = append(fields, field.Names[0].Name)
				}
			}
		}
		var specDescription string
		if typespec.Doc != nil {
			specDescription += "   " + strings.TrimSpace(typespec.Doc.Text())
		}
		c.structs[pkgName(pkg)] = append(c.structs[pkgName(pkg)], structSet{name: typespec.Name.Name, fields: fields, description: genDeclDescription + specDescription})
	}
}
