package decls

import (
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/kakkky/gonsole/errs"
)

type DeclEntry struct {
	decls *[]Decl
}

func NewDeclEntry() *DeclEntry {
	return &DeclEntry{
		decls: &[]Decl{},
	}
}

func (de *DeclEntry) Register(input string) error {
	fset := token.NewFileSet()
	wrappedSrc := "package main\nfunc main() {\n" + input + "\n}"
	inputAst, err := parser.ParseFile(fset, "", wrappedSrc, parser.AllErrors)
	if err != nil {
		return errs.NewInternalError("failed to parse input").Wrap(err)
	}
	var inputStmt ast.Stmt
	for _, decl := range inputAst.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			inputStmt = funcDecl.Body.List[0]
		}
	}
	switch stmt := inputStmt.(type) {
	// 短縮変数宣言の場合
	case *ast.AssignStmt:
		for i, rhsExpr := range stmt.Rhs {
			switch exprV := rhsExpr.(type) {
			// 右辺がセレクタ式の場合
			case *ast.SelectorExpr:
				pkgSelName := exprV.X.(*ast.Ident).Name
				name := stmt.Lhs[i].(*ast.Ident).Name
				declVar := &declVar{
					name: exprV.Sel.Name,
				}
				de.register(pkgSelName, name, *declVar)
			// 右辺が複合リテラルの場合
			case *ast.CompositeLit:
				switch innerExprV := exprV.Type.(type) {
				// セレクタ式の場合
				// 基本的にセレクタ式しか想定しない
				case *ast.SelectorExpr:
					pkgName := innerExprV.X.(*ast.Ident).Name
					declStruct := &declStruct{
						// 型名を取得
						typeName: innerExprV.Sel.Name,
					}
					name := stmt.Lhs[i].(*ast.Ident).Name

					de.register(pkgName, name, *declStruct)
				}
			// 右辺が演算子つきの場合
			case *ast.UnaryExpr:
				switch exprV.Op {
				// & 演算子の場合
				// (構造体をポインタ型で表現している場合など）
				case token.AND:
					switch innerExprV := exprV.X.(type) {
					// 複合リテラルの場合
					case *ast.CompositeLit:
						switch typeExpr := innerExprV.Type.(type) {
						case *ast.SelectorExpr:
							pkgName := typeExpr.X.(*ast.Ident).Name
							declStruct := &declStruct{
								typeName: typeExpr.Sel.Name,
							}
							name := stmt.Lhs[i].(*ast.Ident).Name
							de.register(pkgName, name, *declStruct)
						}
					}
				}
			// 右辺が関数呼び出しの場合
			case *ast.CallExpr:
				switch funExprV := exprV.Fun.(type) {
				case *ast.SelectorExpr:
					// . 呼び出しの左側はまだパッケージ名か定義した変数かわからない
					xName := funExprV.X.(*ast.Ident).Name
					// 定義ずみの変数だったら、それはメソッド呼び出し
					if de.IsRegisteredDecl(xName) {
						declReceiver := xName
						pkgName := de.receiverTypePkgName(declReceiver)
						for j, lhsExpr := range stmt.Lhs {
							methodDecl := &declMethod{
								name:  funExprV.Sel.Name,
								order: j,
							}
							name := lhsExpr.(*ast.Ident).Name
							de.register(pkgName, name, *methodDecl)
						}
					}
					// パッケージ名付きの関数呼び出し (pkg.Func())
					pkgName := xName
					for j, lhsExpr := range stmt.Lhs {
						funcDecl := &declFunc{
							name:  funExprV.Sel.Name,
							order: j,
						}
						name := lhsExpr.(*ast.Ident).Name
						de.register(pkgName, name, *funcDecl)
					}
					return nil

				}
			}
		}
	// 宣言の場合
	case *ast.DeclStmt:
		switch decl := stmt.Decl.(type) {
		// 汎用的宣言の場合(const, var)
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch specV := spec.(type) {
				// 変数宣言
				case *ast.ValueSpec:
					for i, valExpr := range specV.Values {
						switch valExprV := valExpr.(type) {
						// セレクタ式の場合
						case *ast.SelectorExpr:
							pkgName := valExprV.X.(*ast.Ident).Name
							declVar := &declVar{
								name: valExprV.Sel.Name,
							}
							name := specV.Names[i].Name
							de.register(pkgName, name, *declVar)
						// 複合リテラルの場合
						case *ast.CompositeLit:
							switch valTypeExprV := valExprV.Type.(type) {
							// セレクタ式の場合
							case *ast.SelectorExpr:
								pkgName := valTypeExprV.X.(*ast.Ident).Name
								declStruct := &declStruct{
									typeName: valTypeExprV.Sel.Name,
								}
								name := specV.Names[i].Name
								de.register(pkgName, name, *declStruct)
							}
						// 演算子つきの場合
						case *ast.UnaryExpr:
							switch valExprV.Op {
							// & 演算子の場合
							case token.AND:
								switch innerValExprV := valExprV.X.(type) {
								// 複合リテラルの場合
								case *ast.CompositeLit:
									switch compositeLitTypeV := innerValExprV.Type.(type) {
									case *ast.SelectorExpr:
										pkgName := compositeLitTypeV.X.(*ast.Ident).Name
										declStruct := &declStruct{
											typeName: compositeLitTypeV.Sel.Name,
										}
										name := specV.Names[i].Name
										de.register(pkgName, name, *declStruct)
									}
								}
							}
						// 関数呼び出しの場合
						case *ast.CallExpr:
							switch funExprV := valExprV.Fun.(type) {
							case *ast.SelectorExpr:
								// パッケージ名付きの関数呼び出し (pkg.Func())
								pkgName := funExprV.X.(*ast.Ident).Name
								funcName := funExprV.Sel.Name
								for j, nameIdent := range specV.Names {
									funcDecl := &declFunc{
										name:  funcName,
										order: j,
									}
									de.register(pkgName, nameIdent.Name, *funcDecl)
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

func (de *DeclEntry) receiverTypePkgName(receiverName string) string {
	for _, decl := range *de.decls {
		if decl.Name() == receiverName {
			return decl.Pkg()
		}
	}
	return ""
}

func (de *DeclEntry) Decls() []Decl {
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
		*de.decls = append(*de.decls, Decl{
			pkg:  pkg,
			name: name,
			rhs:  declRhs{declVar: v},
		})
	case declFunc:
		*de.decls = append(*de.decls, Decl{
			pkg:  pkg,
			name: name,
			rhs:  declRhs{declFunc: v},
		})
	case declMethod:
		*de.decls = append(*de.decls, Decl{
			pkg:  pkg,
			name: name,
			rhs:  declRhs{declMethod: v},
		})
	case declStruct:
		*de.decls = append(*de.decls, Decl{
			pkg:  pkg,
			name: name,
			rhs:  declRhs{declStruct: v},
		})
	default:
	}
}
