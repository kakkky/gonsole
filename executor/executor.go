package executor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Executor struct {
	modPath     string
	tmpCleaner  func()
	tmpFilePath string
}

func NewExecutor() *Executor {
	tmpFilePath, cleaner, err := makeTmpMainFile()
	if err != nil {
		log.Fatalln("Failed to create temporary main file:", err)
	}
	modPath, err := getGoModPath("go.mod")
	if err != nil {
		log.Println("Failed to get module path, using empty path:", err)
	}
	return &Executor{
		modPath:     modPath,
		tmpCleaner:  cleaner,
		tmpFilePath: tmpFilePath,
	}
}

func (e *Executor) Execute(input string) {
	if err := e.addToTmpSrc(input); err != nil {
		fmt.Println(err)
	}
	cmd := exec.Command("go", "run", e.tmpFilePath)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	errMsg := stderrBuf.String()
	if err != nil {
		fmt.Printf("\033[31m%s\033[0m", formatErrMsg(errMsg))
	}
	if stdoutBuf.Len() > 0 {
		fmt.Printf("\033[32m%s\033[0m", stdoutBuf.String())
	}
	// 関数呼び出しだった場合はそれをtmpファイルから削除する
	if err := e.deleteCallExpr(); err != nil {
		log.Println("Failed to delete call expression from temporary source file:", err)
	}
	// エラーが発生した行を削除する
	if errMsg != "" {
		if err := e.deleteErrLine(errMsg); err != nil {
			log.Println("Failed to delete error line from temporary source file:", err)
		}
	}
}

func (e *Executor) Close() {
	if e.tmpCleaner != nil {
		e.tmpCleaner()
	}
}

func (e *Executor) addToTmpSrc(input string) error {
	// ファイル内容を直接読み込む
	tmpContent, err := os.ReadFile(e.tmpFilePath)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, e.tmpFilePath, string(tmpContent), parser.AllErrors)
	if err != nil {
		return err
	}
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
	var importPkg string
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			switch stmt := inputStmt.(type) {
			case *ast.ExprStmt:
				callExpr, ok := stmt.X.(*ast.CallExpr)
				if ok {
					if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
							importPkg = pkgIdent.Name
						}
					}
					printWrapper := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun:  ast.NewIdent("fmt.Println"),
							Args: []ast.Expr{callExpr}, // 引数に元の関数呼び出し
						},
					}
					funcDecl.Body.List = append(funcDecl.Body.List, printWrapper)
					addFmtImportDecl(file) // fmt パッケージをインポートする
				}
			case *ast.AssignStmt:
				for _, expr := range stmt.Rhs {
					switch rhs := expr.(type) {
					case *ast.SelectorExpr: // 関数呼び出しなどの時はこちら
						if pkgIdent, ok := rhs.X.(*ast.Ident); ok {
							importPkg = pkgIdent.Name
						}
					case *ast.CompositeLit:
						if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
							if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
								importPkg = pkgIdent.Name
							}
						}
					case *ast.CallExpr:
						// 関数の戻り値を代入している場合
						if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
							if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
								importPkg = pkgIdent.Name
							}
						}
					}
				}
				funcDecl.Body.List = append(funcDecl.Body.List, stmt)
				blankAssign := &ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
					Tok: token.ASSIGN,
					Rhs: stmt.Lhs,
				}
				funcDecl.Body.List = append(funcDecl.Body.List, blankAssign)
			case *ast.DeclStmt:
				switch decl := stmt.Decl.(type) {
				case *ast.GenDecl:
					for _, spec := range decl.Specs {
						if valSpec, ok := spec.(*ast.ValueSpec); ok {
							for _, val := range valSpec.Values {
								switch rhs := val.(type) {
								case *ast.SelectorExpr:
									if pkgIdent, ok := rhs.X.(*ast.Ident); ok {
										importPkg = pkgIdent.Name
									}
								case *ast.CompositeLit:
									// 構造体リテラルの型が SelectorExpr
									if selExpr, ok := rhs.Type.(*ast.SelectorExpr); ok {
										if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
											importPkg = pkgIdent.Name
										}
									}
								case *ast.CallExpr:
									// 関数の戻り値を代入している場合
									if selExpr, ok := rhs.Fun.(*ast.SelectorExpr); ok {
										if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
											importPkg = pkgIdent.Name
										}
									}
								}
							}
						}
					}
				}
				funcDecl.Body.List = append(funcDecl.Body.List, stmt)
				blankAssign := &ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{
						stmt.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0],
					},
				}
				funcDecl.Body.List = append(funcDecl.Body.List, blankAssign)
			}
		}
	}
	e.addImportDecl(file, importPkg)
	outFile, err := os.OpenFile(e.tmpFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	// フォーマット済みのコードをファイルに書き込む
	err = format.Node(outFile, fset, file)
	if err != nil {
		return err
	}
	return nil
}

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
								pkgName = pkgIdent.Name
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
	deleteImportDecl(file, "fmt")
	deleteImportDecl(file, fmt.Sprintf("%s/%s", e.modPath, pkgName))
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

func (e *Executor) addImportDecl(file *ast.File, importPkg string) {
	if importPkg == "" {
		return
	}
	importPath := fmt.Sprintf(`"%s/%s"`, e.modPath, importPkg)
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}

		// すでに同じ import があるかチェック
		for _, spec := range genDecl.Specs {
			importSpec := spec.(*ast.ImportSpec)
			if importSpec.Path.Value == importPath {
				// すでにあるので何もしない
				return
			}
		}

		// 追加する
		genDecl.Specs = append(genDecl.Specs, &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: importPath,
			},
		})
		return
	}
}

func addFmtImportDecl(file *ast.File) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}

		// fmt を追加する
		genDecl.Specs = append(genDecl.Specs, &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: `"fmt"`,
			},
		})
		return
	}
}

func deleteImportDecl(file *ast.File, pkg string) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}

		for j, spec := range genDecl.Specs {
			importSpec := spec.(*ast.ImportSpec)
			if importSpec.Path.Value == fmt.Sprintf(`"%s"`, pkg) {
				genDecl.Specs = append(genDecl.Specs[:j], genDecl.Specs[j+1:]...)
			}
			return
		}
	}
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
	file, err := parser.ParseFile(fset, e.tmpFilePath, string(tmpContent), parser.AllErrors)
	if err != nil {
		return err
	}
	var pkgName string
	var isblankAssignExist bool
	for _, decl := range file.Decls {
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
	if !isPkgUsed(pkgName, file) {
		deleteImportDecl(file, fmt.Sprintf("%s/%s", e.modPath, pkgName))
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

func formatErrMsg(input string) string {
	lines := strings.Split(input, "\n")
	var result []string

	cliPattern := regexp.MustCompile(`^# command-line-arguments$`)
	pathPrefixPattern := regexp.MustCompile(`tmp/gonsole[0-9]+/main\.go:\d+:\d+:\s*`)
	var errCount int
	for _, line := range lines {
		if cliPattern.MatchString(line) || line == "" {
			continue
		}
		line = pathPrefixPattern.ReplaceAllString(line, "")
		if !strings.HasPrefix(line, "\t") {
			errCount++
			line = fmt.Sprintf("ERR: %s", line)
		}
		result = append(result, line)
	}

	return fmt.Sprintf("\n%d errors found:\n\n%s\n\n", errCount, strings.Join(result, "\n"))
}
