package executor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"

	"github.com/kakkky/gonsole/errs"
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

	var pkgNameToDelete string
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
func extractPkgNameFromPrintlnExprArg(callExpr *ast.CallExpr) string {
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
			selExpr := argExpr.Fun.(*ast.SelectorExpr)
			pkgIdent := selExpr.X.(*ast.Ident)
			return pkgIdent.Name
		case *ast.SelectorExpr:
			pkgIdent := argExpr.X.(*ast.Ident)
			return pkgIdent.Name
		case *ast.Ident:
			// 直接識別子の場合はパッケージ名がないので空
			return ""
		}
	}
	return ""
}

func (e *Executor) deleteImportDecl(file *ast.File, pkgNameToDelete string) error {
	var importPathQuoted string
	if pkgNameToDelete == "fmt" {
		importPathQuoted = `"fmt"`
	} else {
		importPath, err := e.resolveImportPath(pkgNameToDelete)
		if err != nil {
			return err
		}
		importPathQuoted = fmt.Sprintf(`"%s"`, importPath)
	}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		for j, spec := range genDecl.Specs {
			importSpec := spec.(*ast.ImportSpec)
			if importSpec.Path.Value == importPathQuoted {
				genDecl.Specs = append(genDecl.Specs[:j], genDecl.Specs[j+1:]...)
			}
			break
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

	var pkgNameToDelete string
	var isblankAssignExist bool
	for _, decl := range tmpFileAst.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			newMainFuncBody := []ast.Stmt{}
			for _, stmt := range funcDecl.Body.List {
				pos := fset.Position(stmt.Pos())
				if isblankAssignExist {
					continue
				}
				if pos.Line == errLineNum {
					switch stmt.(type) {
					case *ast.ExprStmt:
						if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
							if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
								if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
									if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
										pkgNameToDelete = pkgIdent.Name
									}
								}
							}
						}
					case *ast.AssignStmt:
						if assignStmt, ok := stmt.(*ast.AssignStmt); ok {
							for _, rhs := range assignStmt.Rhs {
								switch rhs := rhs.(type) {
								case *ast.SelectorExpr:
									if pkgIdent, ok := rhs.X.(*ast.Ident); ok {
										pkgNameToDelete = pkgIdent.Name
									}
								case *ast.CompositeLit:
									// 構造体リテラルの型が SelectorExpr
									if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
										if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
											pkgNameToDelete = pkgIdent.Name
										}
									}
								case *ast.CallExpr:
									// 関数の戻り値を代入している場合
									if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
										if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
											pkgNameToDelete = pkgIdent.Name
										}
									}
								}
							}
						}
						// 短縮変数宣言の場合は次の行にブランク代入された変数があるはずなので、フラグをtrueにしておく
						isblankAssignExist = true
					case *ast.DeclStmt:
						if declStmt, ok := stmt.(*ast.DeclStmt); ok {
							for _, decl := range declStmt.Decl.(*ast.GenDecl).Specs {
								if valSpec, ok := decl.(*ast.ValueSpec); ok {
									for _, val := range valSpec.Values {
										switch rhs := val.(type) {
										case *ast.SelectorExpr:
											if pkgIdent, ok := rhs.X.(*ast.Ident); ok {
												pkgNameToDelete = pkgIdent.Name
											}
										case *ast.CompositeLit:
											// 構造体リテラルの型が SelectorExpr
											if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
												if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
													pkgNameToDelete = pkgIdent.Name
												}
											}
										case *ast.CallExpr:
											// 関数の戻り値を代入している場合
											if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
												if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
													pkgNameToDelete = pkgIdent.Name
												}
											}
										}
									}
								}
							}
						}
						// 変数宣言の場合は次の行にブランク代入された変数があるはずなので、フラグをtrueにしておく
						isblankAssignExist = true
					}
					continue
				}
				newMainFuncBody = append(newMainFuncBody, stmt)
			}
			funcDecl.Body.List = newMainFuncBody
		}
	}
	if !isPkgUsed(pkgNameToDelete, tmpFileAst) {
		if err := e.deleteImportDecl(tmpFileAst, pkgNameToDelete); err != nil {
			return err
		}
	}
	if err := outputToFile(e.tmpFilePath, tmpFileAst); err != nil {
		return err
	}
	return nil
}

func isPkgUsed(pkgName string, file *ast.File) bool {
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			for _, stmt := range funcDecl.Body.List {
				switch stmt := stmt.(type) {
				case *ast.AssignStmt:
					for _, expr := range stmt.Rhs {
						switch rhs := expr.(type) {
						case *ast.SelectorExpr: // 関数呼び出しなどの時はこちら
							if pkgIdent, ok := rhs.X.(*ast.Ident); ok {
								if pkgName == pkgIdent.Name {
									return true
								}
							}
						case *ast.CompositeLit:
							if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
								if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
									if pkgName == pkgIdent.Name {
										return true
									}
								}
							}
						case *ast.CallExpr:
							// 関数の戻り値を代入している場合
							if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
								if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
									if pkgName == pkgIdent.Name {
										return true
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
								for _, val := range valSpec.Values {
									switch rhs := val.(type) {
									case *ast.SelectorExpr:
										if pkgIdent, ok := rhs.X.(*ast.Ident); ok {
											if pkgName == pkgIdent.Name {
												return true
											}
										}
									case *ast.CompositeLit:
										// 構造体リテラルの型が SelectorExpr
										if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
											if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
												if pkgName == pkgIdent.Name {
													return true
												}
											}
										}
									case *ast.CallExpr:
										// 関数の戻り値を代入している場合
										if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
											if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
												if pkgName == pkgIdent.Name {
													return true
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
		}
	}
	return false
}
