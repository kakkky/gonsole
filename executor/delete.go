package executor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"slices"
	"strconv"

	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
)

func (e *Executor) deleteCallExpr() error {
	// 一時ファイルの内容を読み込む
	tmpContent, err := os.ReadFile(e.tmpFilePath)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, e.tmpFilePath, string(tmpContent), parser.AllErrors)
	if err != nil {
		return err
	}

	var pkgNameToDelete types.PkgName
	// main関数を探す
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != "main" {
			continue
		}

		// fmt.Println(pkg.func()) パターンを探して削除

		originMainFuncBody := &funcDecl.Body.List
		newMainFuncBody := []ast.Stmt{}
		for _, stmt := range *originMainFuncBody {
			// ExprStmt でない場合はそのまま追加
			exprStmt, ok := stmt.(*ast.ExprStmt)
			if !ok {
				newMainFuncBody = append(newMainFuncBody, stmt)
				continue
			}

			// CallExpr でない場合もそのまま追加
			callExpr, ok := exprStmt.X.(*ast.CallExpr)
			if !ok {
				newMainFuncBody = append(newMainFuncBody, stmt)
				continue
			}

			// fmt.Println(pkg.func()) パターンからパッケージ名を抽出
			pkgNameToDelete = extractPkgNameFromPrintlnExprArg(callExpr)
		}

		// fmt.Println(pkg.func()) パターンが取り除かれたmain関数の中身に置き換える
		*originMainFuncBody = newMainFuncBody
		break
	}

	// 関数呼び出しはfmt.Printlnでラップされているので、fmtパッケージを削除
	if err := e.deleteImportDecl(file, "fmt"); err != nil {
		return err
	}

	// パッケージ名が使用されていない場合はインポートを削除
	if !isPkgUsed(pkgNameToDelete, file) {
		if err := e.deleteImportDecl(file, pkgNameToDelete); err != nil {
			return err
		}
	}

	return outputToFile(e.tmpFilePath, file)
}

// fmt.Println(pkg.func()) パターンからパッケージ名を抽出
// 該当しない場合は空文字列を返す
func extractPkgNameFromPrintlnExprArg(callExpr *ast.CallExpr) types.PkgName {
	// 関数式をチェック
	switch funExprV := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		// fmt.Println を期待
		xIdent, ok := funExprV.X.(*ast.Ident)
		if !ok || xIdent.Name != "fmt" || funExprV.Sel.Name != "Println" {
			return ""
		}

		// 引数があるか確認
		if len(callExpr.Args) == 0 {
			return ""
		}

		// 第一引数をチェック
		switch argExpr := callExpr.Args[0].(type) {
		case *ast.CallExpr:
			selExpr, ok := argExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return ""
			}
			switch x := selExpr.X.(type) {
			case *ast.Ident:
				return types.PkgName(x.Name)
			default:
				// メソッドチェーンなどに対応
				return extractPkgNameFromRhs(selExpr.X)
			}
		case *ast.SelectorExpr:
			pkgIdent := argExpr.X.(*ast.Ident)
			return types.PkgName(pkgIdent.Name)
		case *ast.Ident:
			// 直接識別子の場合はパッケージ名がないので空
			return ""
		}
	}
	return ""
}

// deleteImportDecl は指定されたパッケージ名のインポート宣言を削除する
func (e *Executor) deleteImportDecl(file *ast.File, pkgNameToDelete types.PkgName) error {
	var importPathQuoteds []string

	importPaths, err := e.resolveImportPathForDelete(pkgNameToDelete)
	if err != nil {
		return err
	}
	for _, importPath := range importPaths {
		importPathQuoteds = append(importPathQuoteds, fmt.Sprintf(`"%s"`, importPath))
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		for j, spec := range genDecl.Specs {
			importSpec := spec.(*ast.ImportSpec)
			if slices.Contains(importPathQuoteds, importSpec.Path.Value) {
				genDecl.Specs = append(genDecl.Specs[:j], genDecl.Specs[j+1:]...)
				break
			}
			continue
		}
	}
	return nil
}

// deleteErrLine はエラーメッセージから行番号を抽出し、その行を削除する
// もし行番号が抽出できなかった場合はエラーを返す
func (e *Executor) deleteErrLine(errMsg string) error {
	// エラーメッセージから行番号を抽出
	re := regexp.MustCompile(`/main\.go:(\d+):(\d+)`)
	matches := re.FindStringSubmatch(errMsg)
	line := matches[1]
	errLineNum, err := strconv.Atoi(line)
	if err != nil {
		return errs.NewInternalError("failed to parse line number from error message").Wrap(err)
	}

	// 一時ファイルの内容を読み込む
	tmpContent, err := os.ReadFile(e.tmpFilePath)
	if err != nil {
		return errs.NewInternalError("failed to read temporary file").Wrap(err)
	}
	fset := token.NewFileSet()
	tmpFileAst, err := parser.ParseFile(fset, e.tmpFilePath, string(tmpContent), parser.AllErrors)
	if err != nil {
		return errs.NewInternalError("failed to parse temporary file").Wrap(err)
	}

	var pkgNameToDelete types.PkgName
	var isblankAssignExist bool

	// main関数を探して対象行を削除
	for _, decl := range tmpFileAst.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != "main" {
			continue
		}

		newMainFuncBody := []ast.Stmt{}
		for _, stmt := range funcDecl.Body.List {
			pos := fset.Position(stmt.Pos())

			// 前の処理で空代入フラグが立っていたら、この行をスキップ
			// 空代入はエラー行の次の行にあることを前提とする
			if isblankAssignExist {
				continue
			}

			// エラー行なら削除してパッケージ名を取得
			if pos.Line == errLineNum {
				// エラー行の種類によって処理を分ける
				switch stmtV := stmt.(type) {
				case *ast.ExprStmt:
					// 式文の場合（関数呼び出しなど）
					switch exprV := stmtV.X.(type) {
					case *ast.CallExpr:
						// 関数呼び出しの場合
						switch exprFuncV := exprV.Fun.(type) {
						case *ast.SelectorExpr:
							// パッケージ関数呼び出し
							pkgNameToDelete = types.PkgName(exprFuncV.X.(*ast.Ident).Name)
						}
					}

				case *ast.AssignStmt:
					// 代入文・変数宣言の場合
					pkgNameToDelete = extractPkgNameFromRhs(stmtV.Rhs[0])
					// 次の行の空代入をスキップするためのフラグを立てる
					isblankAssignExist = true
				case *ast.DeclStmt:
					// 宣言文の場合
					switch declType := stmtV.Decl.(type) {
					case *ast.GenDecl:
						for _, spec := range declType.Specs {
							switch specV := spec.(type) {
							case *ast.ValueSpec:
								pkgNameToDelete = extractPkgNameFromRhs(specV.Values[0])
							}
						}
					}
					// 次の行の空代入をスキップするためのフラグを立てる
					isblankAssignExist = true
				}

				continue // エラー行は追加しない（削除）
			}

			// エラー行でなければそのまま追加
			newMainFuncBody = append(newMainFuncBody, stmt)
		}

		// 修正した本文で更新
		funcDecl.Body.List = newMainFuncBody
		break
	}

	// 使用されていないパッケージのインポートを削除
	if !isPkgUsed(pkgNameToDelete, tmpFileAst) {
		if err := e.deleteImportDecl(tmpFileAst, pkgNameToDelete); err != nil {
			return err
		}
	}

	// 修正したASTをファイルに書き出す
	return outputToFile(e.tmpFilePath, tmpFileAst)
}

// 右辺式からパッケージ名を抽出するヘルパー関数
func extractPkgNameFromRhs(expr ast.Expr) types.PkgName {
	switch exprV := expr.(type) {
	case *ast.SelectorExpr:
		// パッケージ.名前 の形式
		return types.PkgName(exprV.X.(*ast.Ident).Name)
	case *ast.CompositeLit:
		// パッケージ.型{} の形式
		switch exprTypeV := exprV.Type.(type) {
		case *ast.SelectorExpr:
			return types.PkgName(exprTypeV.X.(*ast.Ident).Name)
		}

	case *ast.CallExpr:
		// パッケージ.関数() の形式やメソッドチェーンにも対応
		switch exprFuncV := exprV.Fun.(type) {
		case *ast.SelectorExpr:
			switch x := exprFuncV.X.(type) {
			case *ast.Ident:
				return types.PkgName(x.Name)
			default:
				// メソッドチェーンなどに対応
				return extractPkgNameFromRhs(exprFuncV.X)
			}
		}
	}

	return ""
}

// パッケージが使用されているかをチェックする
func isPkgUsed(pkgName types.PkgName, fileAst *ast.File) bool {
	if pkgName == "" {
		return false
	}

	for _, decl := range fileAst.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Name.Name != "main" {
				continue
			}

			// main関数内の各文を確認
			for _, stmt := range decl.Body.List {
				switch stmt := stmt.(type) {
				case *ast.AssignStmt:
					// 代入文の右辺をチェック
					for _, expr := range stmt.Rhs {
						if isPkgInExpr(expr, pkgName) {
							return true
						}
					}

				case *ast.DeclStmt:
					// 宣言文の場合
					switch decl := stmt.Decl.(type) {
					case *ast.GenDecl:
						for _, spec := range decl.Specs {
							switch spec := spec.(type) {
							case *ast.ValueSpec:
								// 変数宣言の値をチェック
								for _, val := range spec.Values {
									if isPkgInExpr(val, pkgName) {
										return true
									}
								}
							}
						}
					}

				case *ast.ExprStmt:
					// 式文の場合
					if isPkgInExpr(stmt.X, pkgName) {
						return true
					}
				}
			}
		}
	}
	return false
}

// 式内にパッケージが使用されているかをチェック
func isPkgInExpr(expr ast.Expr, pkgName types.PkgName) bool {
	switch expr := expr.(type) {
	case *ast.SelectorExpr:
		// pkg.Name パターン
		switch x := expr.X.(type) {
		case *ast.Ident:
			return x.Name == string(pkgName)
		}

	case *ast.CompositeLit:
		// pkg.Type{} パターン
		switch typ := expr.Type.(type) {
		case *ast.SelectorExpr:
			switch x := typ.X.(type) {
			case *ast.Ident:
				return x.Name == string(pkgName)
			}
		}

	case *ast.CallExpr:
		// pkg.Func() パターン
		switch fun := expr.Fun.(type) {
		case *ast.SelectorExpr:
			switch x := fun.X.(type) {
			case *ast.Ident:
				return x.Name == string(pkgName)
			}
		}
	}

	return false
}
