package executor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
)

// 作成済みのtmpファイルに入力値を追加する
func (e *Executor) addInputToTmpSrc(input string) error {
	// tmpファイル内容を直接読み込む
	tmpContent, err := os.ReadFile(e.tmpFilePath)
	if err != nil {
		return errs.NewInternalError("failed to read temporary file").Wrap(err)
	}
	fset := token.NewFileSet()
	tmpFileAst, err := parser.ParseFile(fset, e.tmpFilePath, string(tmpContent), parser.AllErrors)
	if err != nil {
		return errs.NewInternalError("failed to parse temporary file").Wrap(err)
	}
	// 入力値をmain関数でラップしてparseする
	wrappedSrc := "package main\nfunc main() {\n" + input + "\n}"
	wrappedInputAst, err := parser.ParseFile(fset, "", wrappedSrc, parser.AllErrors)
	if err != nil {
		return errs.NewInternalError("failed to parse input source").Wrap(err)
	}
	// 入力文をASTとして取得
	var inputStmt ast.Stmt
	for _, decl := range wrappedInputAst.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			inputStmt = funcDecl.Body.List[0]
		}
	}

	// 一時ファイルに入力文を追加していく
	// 必要であればパッケージ名を取得してインポート文も追加する
	for _, decl := range tmpFileAst.Decls {
		var pkgNameToImport types.PkgName
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			mainFuncBody := &funcDecl.Body.List
			switch stmt := inputStmt.(type) {
			// 式だった場合は、それを評価して値を返すためにfmt.Printlnでラップする
			case *ast.ExprStmt:
				switch exprV := stmt.X.(type) {
				// 関数呼び出しの場合
				case *ast.CallExpr:
					// 式からパッケージ名を抽出
					selectorBase, found := extractSelectorBaseFrom(exprV)
					if !found {
						return errs.NewInternalError("failed to extract package name from expression")
					}
					if e.declEntry.IsRegisteredDecl(selectorBase) {
						pkgNameToImport = e.declEntry.ReceiverTypePkgName(selectorBase)
					} else {
						pkgNameToImport = types.PkgName(selectorBase)
						// パッケージのインポート文を追加
						if err := e.addImportDecl(tmpFileAst, pkgNameToImport); err != nil {
							return err
						}
					}
					ok, err := e.isFuncVoid(pkgNameToImport, exprV.Fun.(*ast.SelectorExpr).Sel.Name)
					if err != nil {
						return err
					}
					var exprStmt *ast.ExprStmt
					if !ok {
						// 関数が返り値を持つ場合はfmt.Printlnでラップ
						exprStmt = wrapWithPrintln(exprV)
						// fmtが必要になるのでimportに追加
						if err := e.addImportDecl(tmpFileAst, "fmt"); err != nil {
							return err
						}
					} else {
						// 関数が返り値を持たない場合はそのまま使用
						exprStmt = &ast.ExprStmt{X: exprV}
					}
					addInputStmt(exprStmt, mainFuncBody)
				// 変数だった場合(repl上で定義した変数単体)
				case *ast.SelectorExpr:
					selectorBase, found := extractSelectorBaseFrom(exprV)
					if !found {
						return errs.NewInternalError("failed to extract package name from expression")
					}
					if !e.declEntry.IsRegisteredDecl(selectorBase) {
						pkgNameToImport = types.PkgName(selectorBase)
						// パッケージのインポート文を追加
						if err := e.addImportDecl(tmpFileAst, pkgNameToImport); err != nil {
							return err
						}
					}
					wrappedExpr := wrapWithPrintln(exprV)
					addInputStmt(wrappedExpr, mainFuncBody)
					if err := e.addImportDecl(tmpFileAst, "fmt"); err != nil {
						return err
					}
				case *ast.Ident:
					wrappedExpr := wrapWithPrintln(exprV)
					addInputStmt(wrappedExpr, mainFuncBody)
					if err := e.addImportDecl(tmpFileAst, "fmt"); err != nil {
						return err
					}
				}
			// 短縮変数宣言だった場合
			case *ast.AssignStmt:
				switch stmt.Rhs[0].(type) {
				case *ast.BasicLit:
				// 基本リテラルの場合はインポート文を追加しない
				default:
					selectorBase, found := extractSelectorBaseFrom(stmt.Rhs[0])
					if !found {
						return errs.NewInternalError("failed to extract package name from expression")
					}
					if !e.declEntry.IsRegisteredDecl(selectorBase) {
						pkgNameToImport = types.PkgName(selectorBase)
						if err := e.addImportDecl(tmpFileAst, pkgNameToImport); err != nil {
							return err
						}
					}
				}
				addInputStmt(stmt, mainFuncBody)
				// 宣言された各変数に対して空代入を作成（値を評価するため）
				for _, lhs := range stmt.Lhs {
					addBlankAssignStmt(lhs, mainFuncBody)
				}
			// 宣言の場合
			case *ast.DeclStmt:
				switch decl := stmt.Decl.(type) {
				case *ast.GenDecl:
					for _, spec := range decl.Specs {
						switch specV := spec.(type) {
						case *ast.ValueSpec:
							switch valueExprV := specV.Values[0].(type) {
							case *ast.BasicLit:
							// 基本リテラルの場合はimport文を追加しない
							default:
								// 値の各式からパッケージ名を抽出
								selectorBase, found := extractSelectorBaseFrom(valueExprV)
								if !found {
									return errs.NewInternalError("failed to extract package name from expression")
								}
								if !e.declEntry.IsRegisteredDecl(selectorBase) {
									pkgNameToImport = types.PkgName(selectorBase)
									// パッケージのインポート文を追加
									if err := e.addImportDecl(tmpFileAst, pkgNameToImport); err != nil {
										return err
									}
								}
							}
							// 宣言文を追加
							addInputStmt(stmt, mainFuncBody)
							// 宣言された各変数に対して空代入を作成
							for _, name := range specV.Names {
								addBlankAssignStmt(name, mainFuncBody)
							}
						}
					}
				}
			default:
				return errs.NewInternalError("still unsupported input statement type")
			}
		}
	}

	if err := outputToFile(e.tmpFilePath, tmpFileAst); err != nil {
		return err
	}
	return nil
}

func addInputStmt(input ast.Stmt, list *[]ast.Stmt) {
	*list = append(*list, input)
}

// wrapWithPrintln は、指定された式を fmt.Println でラップする
func wrapWithPrintln(exprV ast.Expr) *ast.ExprStmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  ast.NewIdent("fmt.Println"),
			Args: []ast.Expr{exprV}, // 引数に元の関数呼び出し
		},
	}
}

// extractSelectorBase は式からセレクタのベース部分を抽出する
func extractSelectorBaseFrom(expr ast.Expr) (string, bool) {
	switch exprV := expr.(type) {
	// セレクタ式の場合（pkg.Name）
	case *ast.SelectorExpr:
		return exprV.X.(*ast.Ident).Name, true
	// 複合リテラルの場合（pkg.Type{}）
	case *ast.CompositeLit:
		switch typeExprV := exprV.Type.(type) {
		case *ast.SelectorExpr:
			switch xExpr := typeExprV.X.(type) {
			case *ast.Ident:
				return xExpr.Name, true
			}
		}
	// 演算子つきの場合（&pkg.Type{}）
	case *ast.UnaryExpr:
		switch exprV.Op {
		case token.AND:
			switch innerExprV := exprV.X.(type) {
			case *ast.CompositeLit:
				switch typeExprV := innerExprV.Type.(type) {
				case *ast.SelectorExpr:
					switch xExpr := typeExprV.X.(type) {
					case *ast.Ident:
						return strings.TrimPrefix(xExpr.Name, "&"), true
					}
				}
			}
		}
	// 関数呼び出しの場合（pkg.Func()）
	case *ast.CallExpr:
		switch funExprV := exprV.Fun.(type) {
		case *ast.SelectorExpr:
			switch x := funExprV.X.(type) {
			case *ast.Ident:
				return x.Name, true
			default:
				// メソッドチェーン対応
				return extractSelectorBaseFrom(funExprV.X)
			}
		}
	}
	return "", false
}

func (e *Executor) addImportDecl(fileAst *ast.File, pkgNameToImport types.PkgName) error {
	// パッケージパスを探索
	importPath, err := e.resolveImportPathForAdd(pkgNameToImport)
	if err != nil {
		return err
	}
	importPathQuoted := fmt.Sprintf(`"%s"`, importPath)

	// インポート宣言部分を取得
	var importGenDecl *ast.GenDecl
	for _, decl := range fileAst.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			importGenDecl = genDecl
			// すでにインポートされているか確認
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*ast.ImportSpec); ok {
					if importSpec.Path.Value == importPathQuoted {
						return nil // すでにインポート済みとして何もしない
					}
				}
			}
			break
		}
	}
	// インポート宣言を追加
	importGenDecl.Specs = append(importGenDecl.Specs, &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: importPathQuoted,
		},
	})
	return nil
}

func addBlankAssignStmt(target ast.Expr, list *[]ast.Stmt) {
	if target.(*ast.Ident).Name == "_" {
		// すでに空代入されている場合は何もしない
		return
	}
	blankAssign := &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{target},
	}
	*list = append(*list, blankAssign)
}

// isFuncVoid は、指定されたパッケージ内の関数が返り値を持たないか(void)を判定します。
// この処理はパッケージ全体の型チェックを伴うため、コストが高い可能性があります。
func (e *Executor) isFuncVoid(pkgName types.PkgName, funcName string) (bool, error) {
	targetPkgs, ok := e.astCache.nodes[pkgName]
	if !ok {
		if _, ok := isStandardPackage(pkgName); ok {
			// 標準パッケージの場合はvoidではないと仮定
			return false, nil
		}
		return false, errs.NewInternalError(fmt.Sprintf("package %q not found", pkgName))
	}

	var files []*ast.File
	for _, pkg := range targetPkgs {
		for _, file := range pkg.Files {
			files = append(files, file)
		}
	}

	if len(files) == 0 {
		return false, errs.NewInternalError(fmt.Sprintf("no source files found for package %q", pkgName))
	}

	// MEMO:
	// パッケージ名が同じで、かつ関数名が同じであった場合、最初に見つかった関数の返り値をチェックすることになるので
	// 正しくvoidかどうかの判別ができない可能性がある。しかし、そもそも同名のパッケージ名が存在すること自体がバッドプラクティスであるため
	// ここではそのようなケースは考慮しない。
	for _, file := range files {
		for _, decl := range file.Decls {
			if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == funcName {
				// 関数が見つかった場合、返り値の型リストをチェック
				if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) == 0 {
					return true, nil // 返り値なし（void）
				}
				return false, nil // 返り値あり
			}
		}
	}

	return false, errs.NewInternalError(fmt.Sprintf("function %q not found in package %q", funcName, pkgName))
}
