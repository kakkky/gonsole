package decls

import (
	"go/ast"
	"go/parser"
	"go/token"
)

type DeclEntry struct {
	decls *[]decl
}

func NewDeclEntry() *DeclEntry {
	return &DeclEntry{
		decls: &[]decl{},
	}
}

func (de *DeclEntry) Register(input string) error {
	fset := token.NewFileSet()
	wrappedSrc := "package main\nfunc main() {\n" + input + "\n}"
	inputAst, err := parser.ParseFile(fset, "", wrappedSrc, parser.AllErrors)
	if err != nil {
		return err
	}
	var inputStmt ast.Stmt
	for _, decl := range inputAst.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			inputStmt = funcDecl.Body.List[0]
		}
	}
	switch stmt := inputStmt.(type) {
	case *ast.AssignStmt:
		for i, expr := range stmt.Rhs {
			switch rhs := expr.(type) {
			case *ast.SelectorExpr:
				pkgName := rhs.X.(*ast.Ident).Name
				name := stmt.Lhs[i].(*ast.Ident).Name
				varDecl := &declVar{
					name: rhs.Sel.Name,
				}
				de.register(pkgName, name, *varDecl)
			case *ast.CompositeLit:
				if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
					pkgName := selExpr.X.(*ast.Ident).Name
					structDecl := &declStruct{
						typeName: selExpr.Sel.Name,
					}
					name := stmt.Lhs[i].(*ast.Ident).Name
					de.register(pkgName, name, *structDecl)
				}
			case *ast.UnaryExpr:
				if rhs.Op == token.AND {
					// & 演算子の場合
					if compLit, ok := rhs.X.(*ast.CompositeLit); ok {
						if selExpr, ok := compLit.Type.(*ast.SelectorExpr); ok {
							pkgName := selExpr.X.(*ast.Ident).Name
							structDecl := &declStruct{
								typeName: selExpr.Sel.Name,
							}
							name := stmt.Lhs[i].(*ast.Ident).Name
							de.register(pkgName, name, *structDecl)
						}
					}
				}
			case *ast.CallExpr:
				// 関数の戻り値を代入している場合
				if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
					pkgName := selExpr.X.(*ast.Ident).Name
					if de.IsRegisteredDecl(pkgName) {
						pkgName = de.receiverTypePkgName(pkgName)
						for i, lhsExpr := range stmt.Lhs {
							methodDecl := &declMethod{
								name:  selExpr.Sel.Name,
								order: i,
							}
							name := lhsExpr.(*ast.Ident).Name
							de.register(pkgName, name, *methodDecl)
						}
					} else {
						for i, lhsExpr := range stmt.Lhs {
							funcDecl := &declFunc{
								name:  selExpr.Sel.Name,
								order: i,
							}
							name := lhsExpr.(*ast.Ident).Name
							de.register(pkgName, name, *funcDecl)
						}
					}
				}
			}
		}
	case *ast.DeclStmt:
		switch decl := stmt.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				if valSpec, ok := spec.(*ast.ValueSpec); ok {
					for i, val := range valSpec.Values {
						switch rhs := val.(type) {
						case *ast.SelectorExpr:
							pkgName := rhs.X.(*ast.Ident).Name
							varDecl := &declVar{
								name: rhs.Sel.Name,
							}
							name := valSpec.Names[i].Name
							de.register(pkgName, name, *varDecl)

						case *ast.CompositeLit:
							// 構造体リテラルの型が SelectorExpr
							if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
								pkgName := selExpr.X.(*ast.Ident).Name
								structDecl := &declStruct{
									typeName: selExpr.Sel.Name,
								}
								name := valSpec.Names[i].Name
								de.register(pkgName, name, *structDecl)
							}
						case *ast.UnaryExpr:
							if rhs.Op == token.AND {
								// & 演算子の場合
								if compLit, ok := rhs.X.(*ast.CompositeLit); ok {
									if selExpr, ok := compLit.Type.(*ast.SelectorExpr); ok {
										pkgName := selExpr.X.(*ast.Ident).Name
										structDecl := &declStruct{
											typeName: selExpr.Sel.Name,
										}
										name := valSpec.Names[i].Name
										de.register(pkgName, name, *structDecl)
									}
								}
							}
						case *ast.CallExpr:
							// 関数の戻り値を代入している場合
							if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
								pkgName := selExpr.X.(*ast.Ident).Name
								for i, name := range valSpec.Names {
									funcDecl := &declFunc{
										name:  selExpr.Sel.Name,
										order: i,
									}
									de.register(pkgName, name.Name, *funcDecl)
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (de *DeclEntry) Decls() []decl {
	return *de.decls
}

func (de *DeclEntry) IsRegisteredDecl(name string) bool {
	for _, decl := range *de.decls {
		if decl.Name() == name {
			return true
		}
	}
	return false
}

func (de *DeclEntry) register(pkg, name string, rhs any) {
	switch v := rhs.(type) {
	case declVar:
		*de.decls = append(*de.decls, decl{
			pkg:  pkg,
			name: name,
			rhs:  declRhs{declVar: v},
		})
	case declFunc:
		*de.decls = append(*de.decls, decl{
			pkg:  pkg,
			name: name,
			rhs:  declRhs{declFunc: v},
		})
	case declMethod:
		*de.decls = append(*de.decls, decl{
			pkg:  pkg,
			name: name,
			rhs:  declRhs{declMethod: v},
		})
	case declStruct:
		*de.decls = append(*de.decls, decl{
			pkg:  pkg,
			name: name,
			rhs:  declRhs{declStruct: v},
		})
	default:
	}
}

func (de *DeclEntry) receiverTypePkgName(receiverName string) string {
	for _, decl := range *de.decls {
		if decl.Name() == receiverName {
			return decl.Pkg()
		}
	}
	return ""
}
