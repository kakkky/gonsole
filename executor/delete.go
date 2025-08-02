package executor

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	var pkgName string
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			newBody := []ast.Stmt{}
			for _, stmt := range funcDecl.Body.List {
				if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
					if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
						if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
							if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
								if pkgIdent.Name == "fmt" && selExpr.Sel.Name == "Println" {
									// 最初の引数を調べる
									if argCallExpr, ok := callExpr.Args[0].(*ast.CallExpr); ok {
										// 引数が関数呼び出しの場合
										if argSelExpr, ok := argCallExpr.Fun.(*ast.SelectorExpr); ok {
											if argPkgIdent, ok := argSelExpr.X.(*ast.Ident); ok {
												// 引数の関数呼び出しからパッケージ名を取得
												pkgName = argPkgIdent.Name
											}
										}
									}
								}
							}
						}
						// 関数呼び出しを削除
						continue
					}
				}
				newBody = append(newBody, stmt)
			}
			funcDecl.Body.List = newBody
			break
		}
	}
	if err := e.deleteImportDecl(file, "fmt"); err != nil {
		return err
	}
	if !isPkgUsed(pkgName, file) {
		if err := e.deleteImportDecl(file, pkgName); err != nil {
			return err
		}
	}
	outFile, err := os.OpenFile(e.tmpFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	err = format.Node(outFile, fset, file)
	if err != nil {
		return err
	}
	return nil
}

func (e *Executor) deleteImportDecl(file *ast.File, pkg string) error {
	// モジュールルート相対の全パッケージディレクトリを探索
	var importPath string
	if err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// パッケージ名に一致するディレクトリか？
		base := filepath.Base(path)
		if base == pkg {
			relPath, err := filepath.Rel(".", path)
			if err != nil {
				return err
			}
			importPath = filepath.ToSlash(filepath.Join(e.modPath, relPath))
			return io.EOF // 早期終了
		}
		return nil
	}); err != nil && err != io.EOF {
		return err
	}
	if importPath == "" {
		importPath = pkg // 直接パッケージ名が指定された場合
	}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}

		for j, spec := range genDecl.Specs {
			importSpec := spec.(*ast.ImportSpec)
			if importSpec.Path.Value == fmt.Sprintf(`"%s"`, importPath) {
				genDecl.Specs = append(genDecl.Specs[:j], genDecl.Specs[j+1:]...)
			}
			break
		}
	}
	return nil
}

func (e *Executor) deleteErrLine(errMsg string) error {
	re := regexp.MustCompile(`/main\.go:(\d+):(\d+)`)
	matches := re.FindStringSubmatch(errMsg)
	line := matches[1]
	lineNum, err := strconv.Atoi(line)
	if err != nil {
		return err
	}

	// 一時ファイルの内容を読み込む
	tmpContent, err := os.ReadFile(e.tmpFilePath)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	tmpFileAst, err := parser.ParseFile(fset, e.tmpFilePath, string(tmpContent), parser.AllErrors)
	if err != nil {
		return err
	}
	var pkgName string
	var isblankAssignExist bool
	for _, decl := range tmpFileAst.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			newList := []ast.Stmt{}
			for _, stmt := range funcDecl.Body.List {
				pos := fset.Position(stmt.Pos())
				if isblankAssignExist {
					continue
				}
				if pos.Line == lineNum {
					switch stmt.(type) {
					case *ast.ExprStmt:
						if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
							if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
								if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
									if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
										pkgName = pkgIdent.Name
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
										pkgName = pkgIdent.Name
									}
								case *ast.CompositeLit:
									// 構造体リテラルの型が SelectorExpr
									if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
										if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
											pkgName = pkgIdent.Name
										}
									}
								case *ast.CallExpr:
									// 関数の戻り値を代入している場合
									if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
										if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
											pkgName = pkgIdent.Name
										}
									}
								}
							}
						}
						isblankAssignExist = true
					case *ast.DeclStmt:
						if declStmt, ok := stmt.(*ast.DeclStmt); ok {
							for _, decl := range declStmt.Decl.(*ast.GenDecl).Specs {
								if valSpec, ok := decl.(*ast.ValueSpec); ok {
									for _, val := range valSpec.Values {
										switch rhs := val.(type) {
										case *ast.SelectorExpr:
											if pkgIdent, ok := rhs.X.(*ast.Ident); ok {
												pkgName = pkgIdent.Name
											}
										case *ast.CompositeLit:
											// 構造体リテラルの型が SelectorExpr
											if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
												if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
													pkgName = pkgIdent.Name
												}
											}
										case *ast.CallExpr:
											// 関数の戻り値を代入している場合
											if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
												if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
													pkgName = pkgIdent.Name
												}
											}
										}
									}
								}
							}
						}
						isblankAssignExist = true
					}
					continue
				}
				newList = append(newList, stmt)
			}
			funcDecl.Body.List = newList
		}
	}
	if !isPkgUsed(pkgName, tmpFileAst) {
		if err := e.deleteImportDecl(tmpFileAst, pkgName); err != nil {
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
