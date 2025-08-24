package executor

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"unicode"

	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/utils"
)

type privateIdentManageInfo struct {
	tmpFilePath string
	cleaner     func()
}

func isIncludePrivateIdent(input string) bool {
	// = があればそれ以降を取得
	if strings.Contains(input, "=") {
		input = strings.Split(input, "=")[1]
	}
	// . の後を取得
	ident := strings.SplitN(input, ".", 2)[1]
	return unicode.IsLower(rune(ident[0]))
}

func wrapWithPublicFunc(input string) string {
	// = があればそれ以降を取得
	if strings.Contains(input, "=") {
		input = strings.Split(input, "=")[1]
	}
	// . の後を取得
	ident := strings.SplitN(input, ".", 2)[1]
	// ( か { があればその前までを取得
	if idx := strings.IndexAny(ident, "{("); idx != -1 {
		return strings.ReplaceAll(input, ident, "Wrapped"+strings.Title(ident[:idx]+"()"))
	}
	return strings.ReplaceAll(input, ident, "Wrapped"+strings.Title(ident)+"()")
}

func (e *Executor) defineWrappedPublicFunc(inputWithPrivateIdent, wrappedPublicFunc, importPath, pkgName string) error {
	if e.declEntry.IsRegisteredDecl(pkgName) {
		pkgName = e.declEntry.ReceiverTypePkgName(pkgName)
	}

	pimf := e.privateIdentManageInfoMap[importPath]
	if pimf == nil {
		relativeDir := strings.TrimPrefix(importPath, e.modPath)
		createdTmpFilePath, cleaner, err := makeTmpFile("." + relativeDir)
		if err != nil {
			return err
		}
		pimf = &privateIdentManageInfo{
			tmpFilePath: createdTmpFilePath,
			cleaner:     cleaner,
		}
		e.privateIdentManageInfoMap[importPath] = pimf
	}
	// tmpファイル内容を直接読み込む
	tmpContent, err := os.ReadFile(pimf.tmpFilePath)
	if err != nil {
		return errs.NewInternalError("failed to read temporary file").Wrap(err)
	}
	fset := token.NewFileSet()
	if strings.Contains(string(tmpContent), strings.ReplaceAll(wrappedPublicFunc, pkgName+".", "")) {
		// すでに定義されている場合は何もしない
		return nil
	}
	tmpFileAst, err := parser.ParseFile(fset, e.tmpFilePath, string(tmpContent), parser.AllErrors)
	if err != nil {
		// EOFなら無視して新たなFileAst作成
		if !strings.Contains(err.Error(), "expected ';', found 'EOF'") {
			return errs.NewInternalError("failed to parse temporary file").Wrap(err)
		}
		tmpFileAst = &ast.File{
			Name:  ast.NewIdent(pkgName),
			Decls: []ast.Decl{},
		}
	}

	// 入力値をmain関数でラップしてparseする
	wrappedSrc := "package main\nfunc main() {\n" + inputWithPrivateIdent + "\n}"
	wrappedInputAst, err := parser.ParseFile(fset, "", wrappedSrc, parser.AllErrors)
	if err != nil {
		return errs.NewInternalError("failed to parse input source").Wrap(err)
	}
	// 入力文をASTとして取得
	var inputStmtWithPrivateIdent ast.Stmt
	for _, decl := range wrappedInputAst.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			inputStmtWithPrivateIdent = funcDecl.Body.List[0]
		}
	}

	switch stmt := inputStmtWithPrivateIdent.(type) {
	case *ast.ExprStmt:
		switch exprV := stmt.X.(type) {
		// 関数呼び出しの場合
		case *ast.CallExpr:
			funcOrMethodName := exprV.Fun.(*ast.SelectorExpr).Sel.Name
			if e.declEntry.IsRegisteredDecl(pkgName) {
				pkgName = e.declEntry.ReceiverTypePkgName(pkgName)
			}
			for _, pkgAst := range e.astCache.nodes[pkgName] {
				for _, fileAst := range pkgAst.Files {
					for _, decls := range fileAst.Decls {
						switch declV := decls.(type) {
						case *ast.FuncDecl:
							if declV.Name.Name == funcOrMethodName {
								var recv *ast.FieldList
								var funExpr ast.Expr

								if declV.Recv != nil {
									receiverName := declV.Recv.List[0].Names[0].Name
									receiverType := declV.Recv.List[0].Type
									recv = &ast.FieldList{
										List: []*ast.Field{
											{Names: []*ast.Ident{{Name: receiverName}}, Type: receiverType},
										},
									}
									// 新しいSelectorExprを作成（元のexprVを変更しない）
									funExpr = &ast.SelectorExpr{
										X:   ast.NewIdent(receiverName),
										Sel: ast.NewIdent(funcOrMethodName),
									}
								} else {
									// 新しいIdentを作成
									funExpr = &ast.Ident{Name: funcOrMethodName}
								}

								returnTypes := declV.Type.Results

								// 新しいCallExprを作成
								newCallExpr := &ast.CallExpr{
									Fun:  funExpr,
									Args: exprV.Args, // 引数はコピー
								}

								var bodyStmt ast.Stmt
								if returnTypes != nil && len(returnTypes.List) > 0 {
									// 戻り値がある場合はreturnでラップ
									bodyStmt = &ast.ReturnStmt{Results: []ast.Expr{newCallExpr}}
								} else {
									// 戻り値がない場合は式文としてそのまま
									bodyStmt = &ast.ExprStmt{X: newCallExpr}
								}

								// tmpFileにラッパー関数を追加
								var name string
								if dotIdx := strings.Index(wrappedPublicFunc, "."); dotIdx != -1 {
									name = strings.TrimSuffix(wrappedPublicFunc[dotIdx+1:], "()")
								} else {
									name = strings.TrimSuffix(wrappedPublicFunc, "()")
								}

								tmpFileAst.Decls = append(tmpFileAst.Decls, &ast.FuncDecl{
									Name: ast.NewIdent(name),
									Recv: recv,
									Type: &ast.FuncType{
										Results: returnTypes,
									},
									Body: &ast.BlockStmt{
										List: []ast.Stmt{
											bodyStmt,
										},
									},
								})
							}
						}
					}
				}

			}
		// セレクタ式の場合
		// 変数/定数参照の場合
		case *ast.SelectorExpr:
			// ニーズを考えにくいので未対応
			return errs.NewInternalError("selector expression is still unsupported private identifier usage")
		}
	// 短縮変数宣言だった場合
	case *ast.AssignStmt:
		switch exprV := stmt.Rhs[0].(type) {
		case *ast.CallExpr:
			funcOrMethodName := exprV.Fun.(*ast.SelectorExpr).Sel.Name
			if e.declEntry.IsRegisteredDecl(pkgName) {
				pkgName = e.declEntry.ReceiverTypePkgName(pkgName)
			}
			for _, pkgAst := range e.astCache.nodes[pkgName] {
				for _, fileAst := range pkgAst.Files {
					for _, decls := range fileAst.Decls {
						switch declV := decls.(type) {
						case *ast.FuncDecl:
							if declV.Name.Name == funcOrMethodName {
								var recv *ast.FieldList
								var funExpr ast.Expr

								if declV.Recv != nil {
									receiverName := declV.Recv.List[0].Names[0].Name
									receiverType := declV.Recv.List[0].Type
									recv = &ast.FieldList{
										List: []*ast.Field{
											{Names: []*ast.Ident{{Name: receiverName}}, Type: receiverType},
										},
									}
									// 新しいSelectorExprを作成
									funExpr = &ast.SelectorExpr{
										X:   ast.NewIdent(receiverName),
										Sel: ast.NewIdent(funcOrMethodName),
									}
								} else {
									// 新しいIdentを作成
									funExpr = &ast.Ident{Name: funcOrMethodName}
								}

								returnTypes := declV.Type.Results

								// 新しいCallExprを作成
								newCallExpr := &ast.CallExpr{
									Fun:  funExpr,
									Args: exprV.Args, // 引数はコピー
								}

								var bodyStmt ast.Stmt
								if returnTypes != nil && len(returnTypes.List) > 0 {
									// 戻り値がある場合はreturnでラップ
									bodyStmt = &ast.ReturnStmt{Results: []ast.Expr{newCallExpr}}
								} else {
									// 戻り値がない場合は式文としてそのまま
									bodyStmt = &ast.ExprStmt{X: newCallExpr}
								}

								// tmpFileにラッパー関数を追加
								var name string
								if dotIdx := strings.Index(wrappedPublicFunc, "."); dotIdx != -1 {
									name = strings.TrimSuffix(wrappedPublicFunc[dotIdx+1:], "()")
								} else {
									name = strings.TrimSuffix(wrappedPublicFunc, "()")
								}

								tmpFileAst.Decls = append(tmpFileAst.Decls, &ast.FuncDecl{
									Name: ast.NewIdent(name),
									Recv: recv,
									Type: &ast.FuncType{
										Results: returnTypes,
									},
									Body: &ast.BlockStmt{
										List: []ast.Stmt{
											bodyStmt,
										},
									},
								})
							}
						}
					}
				}
			}
		}
	// 宣言の場合
	case *ast.DeclStmt:
		switch decl := stmt.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch specV := spec.(type) {
				case *ast.ValueSpec:
					switch exprV := specV.Values[0].(type) {
					case *ast.CallExpr:
						funcOrMethodName := exprV.Fun.(*ast.SelectorExpr).Sel.Name
						if e.declEntry.IsRegisteredDecl(pkgName) {
							pkgName = e.declEntry.ReceiverTypePkgName(pkgName)
						}
						for _, pkgAst := range e.astCache.nodes[pkgName] {
							for _, fileAst := range pkgAst.Files {
								for _, decls := range fileAst.Decls {
									switch declV := decls.(type) {
									case *ast.FuncDecl:
										if declV.Name.Name == funcOrMethodName {
											var recv *ast.FieldList
											var funExpr ast.Expr

											if declV.Recv != nil {
												receiverName := declV.Recv.List[0].Names[0].Name
												receiverType := declV.Recv.List[0].Type
												recv = &ast.FieldList{
													List: []*ast.Field{
														{Names: []*ast.Ident{{Name: receiverName}}, Type: receiverType},
													},
												}
												// 新しいSelectorExprを作成
												funExpr = &ast.SelectorExpr{
													X:   ast.NewIdent(receiverName),
													Sel: ast.NewIdent(funcOrMethodName),
												}
											} else {
												// 新しいIdentを作成
												funExpr = &ast.Ident{Name: funcOrMethodName}
											}

											returnTypes := declV.Type.Results

											// 新しいCallExprを作成
											newCallExpr := &ast.CallExpr{
												Fun:  funExpr,
												Args: exprV.Args, // 引数はコピー
											}

											var bodyStmt ast.Stmt
											if returnTypes != nil && len(returnTypes.List) > 0 {
												// 戻り値がある場合はreturnでラップ
												bodyStmt = &ast.ReturnStmt{Results: []ast.Expr{newCallExpr}}
											} else {
												// 戻り値がない場合は式文としてそのまま
												bodyStmt = &ast.ExprStmt{X: newCallExpr}
											}

											// tmpFileにラッパー関数を追加
											var name string
											if dotIdx := strings.Index(wrappedPublicFunc, "."); dotIdx != -1 {
												name = strings.TrimSuffix(wrappedPublicFunc[dotIdx+1:], "()")
											} else {
												name = strings.TrimSuffix(wrappedPublicFunc, "()")
											}

											tmpFileAst.Decls = append(tmpFileAst.Decls, &ast.FuncDecl{
												Name: ast.NewIdent(name),
												Recv: recv,
												Type: &ast.FuncType{
													Results: returnTypes,
												},
												Body: &ast.BlockStmt{
													List: []ast.Stmt{
														bodyStmt,
													},
												},
											})
										}
									}
								}
							}
						}
					}

				}
			}
		}
	}
	if err := outputToFile(pimf.tmpFilePath, tmpFileAst); err != nil {
		return err
	}
	// ASTキャッシュを更新
	nodes, fset, err := utils.AnalyzeGoAst(".")
	if err != nil {
		return err
	}
	e.astCache = &astCache{
		nodes: nodes,
		fset:  fset,
	}

	return nil
}
