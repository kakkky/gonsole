package decl_registry

import (
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
)

type DeclRegistry struct {
	decls []Decl
}

func NewRegistry() *DeclRegistry {
	return &DeclRegistry{
		decls: []Decl{},
	}
}

func (dr *DeclRegistry) Register(input string) error {
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
	switch stmtV := inputStmt.(type) {
	case *ast.AssignStmt:
		dr.registerAssimentStmt(stmtV)
	case *ast.DeclStmt:
		dr.registerDeclStmt(stmtV)
	default:
		return errs.NewBadInputError("unsupported statement type")
	}
	return nil
}

func (dr *DeclRegistry) registerAssimentStmt(assignmentStmt *ast.AssignStmt) {
	for i, stmtRhs := range assignmentStmt.Rhs {
		switch stmtRhsV := stmtRhs.(type) {
		// 右辺がセレクタ式の場合
		case *ast.SelectorExpr:
			decl := Decl{
				name: types.DeclName(assignmentStmt.Lhs[i].(*ast.Ident).Name),
				rhs: declRhs{
					name:    types.DeclName(stmtRhsV.Sel.Name),
					kind:    DeclRhsKindVar,
					pkgName: types.PkgName(stmtRhsV.X.(*ast.Ident).Name),
				},
			}
			dr.register(decl)
			continue
		case *ast.CompositeLit:
			switch stmtRhsTypeV := stmtRhsV.Type.(type) {
			// セレクタ式の場合
			// 基本的にセレクタ式しか想定しない
			case *ast.SelectorExpr:
				decl := Decl{
					name: types.DeclName(assignmentStmt.Lhs[i].(*ast.Ident).Name),
					rhs: declRhs{
						name:    types.DeclName(stmtRhsTypeV.Sel.Name),
						kind:    DeclRhsKindStruct,
						pkgName: types.PkgName(stmtRhsTypeV.X.(*ast.Ident).Name),
					},
				}
				dr.register(decl)
				continue
			}
		// 右辺が演算子つきの場合
		case *ast.UnaryExpr:
			switch stmtRhsV.Op {
			// & 演算子の場合
			// (構造体をポインタ型で表現している場合など）
			case token.AND:
				switch rhsExprV := stmtRhsV.X.(type) {
				case *ast.CompositeLit:
					switch rhsExprTypeV := rhsExprV.Type.(type) {
					case *ast.SelectorExpr:
						decl := Decl{
							name: types.DeclName(assignmentStmt.Lhs[i].(*ast.Ident).Name),
							rhs: declRhs{
								name:    types.DeclName(rhsExprTypeV.Sel.Name),
								kind:    DeclRhsKindStruct,
								pkgName: types.PkgName(rhsExprTypeV.X.(*ast.Ident).Name),
							},
						}
						dr.register(decl)
						continue
					}
				}
			}
		// 右辺が関数呼び出しの場合
		case *ast.CallExpr:
			switch rhsFunV := stmtRhsV.Fun.(type) {
			case *ast.SelectorExpr:
				var selectorBase string
				switch rhsFunExprV := rhsFunV.X.(type) {
				case *ast.Ident:
					selectorBase = rhsFunExprV.Name
				default:
					// TODO: メソッドチェーンなど複雑な場合の対応(ast.SelectorExprが続き得る）
					continue
				}
				if dr.IsRegisteredDecl(types.DeclName(selectorBase)) {
					for i, lhsExpr := range assignmentStmt.Lhs {
						decl := Decl{
							name:        types.DeclName(lhsExpr.(*ast.Ident).Name),
							isReturnVal: true,
							returnedIdx: i,
							rhs: declRhs{
								name:    types.DeclName(rhsFunV.Sel.Name),
								kind:    DeclRhsKindMethod,
								pkgName: dr.PkgNameOfReceiver(types.DeclName(selectorBase)),
							},
						}
						dr.register(decl)
					}
					continue
				}
				for i, lhsExpr := range assignmentStmt.Lhs {
					decl := Decl{
						name:        types.DeclName(lhsExpr.(*ast.Ident).Name),
						isReturnVal: true,
						returnedIdx: i,
						rhs: declRhs{
							name:    types.DeclName(rhsFunV.Sel.Name),
							kind:    DeclRhsKindFunc,
							pkgName: types.PkgName(selectorBase),
						},
					}
					dr.register(decl)
				}
				continue
			}
		}
	}
}
func (dr *DeclRegistry) registerDeclStmt(declStmt *ast.DeclStmt) {
	switch stmtDeclV := declStmt.Decl.(type) {
	case *ast.GenDecl:
		for _, stmtDeclSpec := range stmtDeclV.Specs {
			switch stmtDeclSpecV := stmtDeclSpec.(type) {
			case *ast.ValueSpec:
				for i, value := range stmtDeclSpecV.Values {
					switch valueV := value.(type) {
					case *ast.SelectorExpr:
						decl := Decl{
							name: types.DeclName(stmtDeclSpecV.Names[i].Name),
							rhs: declRhs{
								name:    types.DeclName(valueV.Sel.Name),
								kind:    DeclRhsKindVar,
								pkgName: types.PkgName(valueV.X.(*ast.Ident).Name),
							},
						}
						dr.register(decl)
						continue
					case *ast.CompositeLit:
						switch valueTypeV := valueV.Type.(type) {
						case *ast.SelectorExpr:
							decl := Decl{
								name: types.DeclName(stmtDeclSpecV.Names[i].Name),
								rhs: declRhs{
									name:    types.DeclName(valueTypeV.Sel.Name),
									kind:    DeclRhsKindStruct,
									pkgName: types.PkgName(valueTypeV.X.(*ast.Ident).Name),
								},
							}
							dr.register(decl)
							continue
						}
					case *ast.UnaryExpr:
						switch valueV.Op {
						// & 演算子の場合
						case token.AND:
							switch valueExprV := valueV.X.(type) {
							// 複合リテラルの場合
							case *ast.CompositeLit:
								switch valueExprTypeV := valueExprV.Type.(type) {
								case *ast.SelectorExpr:
									decl := Decl{
										name: types.DeclName(stmtDeclSpecV.Names[i].Name),
										rhs: declRhs{
											name:    types.DeclName(valueExprTypeV.Sel.Name),
											kind:    DeclRhsKindStruct,
											pkgName: types.PkgName(valueExprTypeV.X.(*ast.Ident).Name),
										},
									}
									dr.register(decl)
									continue
								}
							}
						}
					// 関数呼び出しの場合
					case *ast.CallExpr:
						switch valueFunV := valueV.Fun.(type) {
						case *ast.SelectorExpr:
							for i, stmtDeclSpecName := range stmtDeclSpecV.Names {
								decl := Decl{
									name:        types.DeclName(stmtDeclSpecName.Name),
									isReturnVal: true,
									returnedIdx: i,
									rhs: declRhs{
										name:    types.DeclName(valueFunV.Sel.Name),
										kind:    DeclRhsKindFunc,
										pkgName: types.PkgName(valueFunV.X.(*ast.Ident).Name),
									},
								}
								dr.register(decl)
							}
							continue
						}
					}
				}
			}
		}
	}
}

func (dr *DeclRegistry) register(decl Decl) {
	dr.decls = append(dr.decls, decl)
}

func (dr *DeclRegistry) PkgNameOfReceiver(receiverName types.DeclName) types.PkgName {
	for _, decl := range dr.decls {
		if decl.Name() == receiverName {
			return decl.rhs.pkgName
		}
	}
	return ""
}

func (dr *DeclRegistry) Decls() []Decl {
	return dr.decls
}

func (dr *DeclRegistry) IsRegisteredDecl(name types.DeclName) bool {
	for _, decl := range dr.decls {
		if decl.Name() == name {
			return true
		}
	}
	return false
}
