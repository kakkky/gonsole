package executor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/kakkky/gonsole/errs"
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
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			mainFuncBody := &funcDecl.Body.List
			switch stmt := inputStmt.(type) {
			// 式だった場合は、それを評価して値を返すためにfmt.Printlnでラップする
			case *ast.ExprStmt:

				switch exprV := stmt.X.(type) {
				// 関数呼び出しの場合
				case *ast.CallExpr, *ast.SelectorExpr:
					// 式からパッケージ名を抽出
					pkgNameToImport, found := extractPkgNameFromExpr(stmt.X)
					if !found {
						return errs.NewInternalError("failed to extract package name from expression")
					}
					// パッケージのインポート文を追加
					if err := e.addImportDecl(tmpFileAst, pkgNameToImport); err != nil {
						return err
					}

					wrappedExpr := wrapWithPrintln(exprV)
					addInputStmt(wrappedExpr, mainFuncBody)
					if err := e.addImportDecl(tmpFileAst, "fmt"); err != nil {
						return err
					}
				// 変数だった場合(repl上で定義した変数単体)
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
					pkgNameToImport, found := extractPkgNameFromExpr(stmt.Rhs[0])
					if !found {
						return errs.NewInternalError("failed to extract package name from expression")
					}
					if err := e.addImportDecl(tmpFileAst, pkgNameToImport); err != nil {
						return err
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
								pkgNameToImport, found := extractPkgNameFromExpr(valueExprV)
								if !found {
									return errs.NewInternalError("failed to extract package name from expression")
								}
								if err := e.addImportDecl(tmpFileAst, pkgNameToImport); err != nil {
									return err
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

// extractPkgNameFromExpr は式からパッケージ名を抽出する
func extractPkgNameFromExpr(expr ast.Expr) (string, bool) {
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
			return funExprV.X.(*ast.Ident).Name, true
		}
	}
	return "", false
}

func (e *Executor) addImportDecl(fileAst *ast.File, pkgNameToImport string) error {
	// repl内で定義された変数エントリにある場合は無視
	// 理由：パッケージ名ではなく、メソッド呼び出しに対するレシーバー名として使用されていると予測できるため
	if e.declEntry.IsRegisteredDecl(pkgNameToImport) {
		return nil
	}

	// インポートパスの準備
	var importPathQuoted string
	if pkgNameToImport == "fmt" { // プロジェクトのパッケージ以外でfmtパッケージのみ使用される
		importPathQuoted = `"fmt"`
	} else {
		// パッケージパスを探索
		importPath, err := e.resolveImportPathForAdd(pkgNameToImport)
		if err != nil {
			return err
		}
		importPathQuoted = fmt.Sprintf(`"%s"`, importPath)
	}

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
