package executor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime/debug"
	"slices"
	"strconv"

	"os/exec"
	"regexp"
	"strings"

	"github.com/kakkky/gonsole/declregistry"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
)

// Executor はREPLセッション内でのコード実行を担う
// go-promptのExecutorインターフェースを実装する
type Executor struct {
	declRegistry *declregistry.DeclRegistry
	sessionSrc   *ast.File
	filer
	commander
	importPathResolver
	fsetProvider
}

// NewExecutor はExecutorのインスタンスを生成する
func NewExecutor(declRegistry *declregistry.DeclRegistry) (*Executor, error) {
	commander := newDefaultCommander()
	return &Executor{
		declRegistry:       declRegistry,
		sessionSrc:         initSessionSrc(),
		filer:              newDefaultFiler(),
		commander:          commander,
		importPathResolver: newDefaultImportPathResolver(commander),
		fsetProvider:       newDefaultFsetProvider(),
	}, nil
}

// ====================以下にメソッドを定義する======================

// Execute は入力されたコードを実行する
func (e *Executor) Execute(input string) {
	defer func() {
		if r := recover(); r != nil {
			panicMsg := fmt.Sprintf("%v", r)
			errs.HandleError(
				errs.NewInternalError(panicMsg),
			)
			fmt.Println(string(debug.Stack()))
		}
	}()

	if input == "" {
		return
	}

	// 入力文をセッションに書き込む
	if err := e.writeInSessionSrc(input); err != nil {
		errs.HandleError(err)
		return
	}

	// 一時ファイルを作成
	tmpFile, tmpFileName, cleanup, err := e.createTmpFile()
	if err != nil {
		errs.HandleError(err)
	}
	defer tmpFile.Close()
	defer cleanup()

	fset := e.fset()

	// 一時ファイルにflushする
	if err := e.flush(e.sessionSrc, tmpFile, fset); err != nil {
		errs.HandleError(err)
		return
	}

	// 一時ファイルを実行する
	cmdOut, cmdErr := e.execGoRun(tmpFileName)
	if cmdErr != nil {
		// 実行時のエラー出力を整形して表示する
		cmdErrMsg := string(cmdErr.(*exec.ExitError).Stderr)

		formatted := formatCmdErrMsg(cmdErrMsg)
		errs.HandleError(errs.NewBadInputError(formatted))

		// エラー行を削除する
		if err := e.cleanErrLineFromSessionSrc(cmdErrMsg, fset); err != nil {
			errs.HandleError(err)
		}

		if err := e.flush(e.sessionSrc, tmpFile, fset); err != nil {
			errs.HandleError(err)
		}

		return
	}

	// 実行結果を表示する
	if len(cmdOut) > 0 {
		printCmdOutput(cmdOut)
	}

	// 変数エントリに登録する
	if err := e.declRegistry.Register(input); err != nil {
		errs.HandleError(err)
		return
	}

	// 式呼び出しをセッションソースから削除した場合はflushする
	if cleaned := e.cleanCallExprFromSessionSrc(); !cleaned {
		return
	}
	if err := e.flush(e.sessionSrc, tmpFile, fset); err != nil {
		errs.HandleError(err)
		return
	}
}

func (e *Executor) writeInSessionSrc(input string) error {
	inputStmtAst, err := parseInput(input)
	if err != nil {
		return err
	}

	mainFunc := getMainFunc(e.sessionSrc)
	switch inputStmtV := inputStmtAst.(type) {
	case *ast.ExprStmt:
		if err := e.appendExprStmtToMainFuncBody(inputStmtV, mainFunc); err != nil {
			return err
		}
	case *ast.AssignStmt:
		if err := e.appendAssignStmtToMainFuncBody(inputStmtV, mainFunc); err != nil {
			return err
		}
	case *ast.DeclStmt:
		if err := e.appendDeclStmtToMainFuncBody(inputStmtV, mainFunc); err != nil {
			return err
		}
	default:
		return errs.NewBadInputError("unsupported statement type")
	}
	return nil
}

func (e *Executor) appendExprStmtToMainFuncBody(exprStmt *ast.ExprStmt, mainFunc *ast.FuncDecl) error {
	switch exprStmtV := exprStmt.X.(type) {
	case *ast.SelectorExpr:
		selectorBase := extractSelectorBaseFromExpr(exprStmtV)
		if !e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
			if err := e.addImportPath(types.PkgName(selectorBase)); err != nil {
				return err
			}
		}
		exprStmt = &ast.ExprStmt{
			X: &ast.CallExpr{
				// AST的には表現が不正確になるがこちらの方がシンプルに書けるのでIdentに押し込める
				Fun:  ast.NewIdent("fmt.Println"),
				Args: []ast.Expr{exprStmtV},
			},
		}
	case *ast.Ident:
		exprStmt = &ast.ExprStmt{
			X: &ast.CallExpr{
				// AST的には表現が不正確になるがこちらの方がシンプルに書けるのでIdentに押し込める
				Fun:  ast.NewIdent("fmt.Println"),
				Args: []ast.Expr{exprStmtV},
			},
		}

	case *ast.CallExpr:
		selectorBase := extractSelectorBaseFromExpr(exprStmtV)
		if !e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
			if err := e.addImportPath(types.PkgName(selectorBase)); err != nil {
				return err
			}
		}
		exprStmt = &ast.ExprStmt{
			X: &ast.CallExpr{
				// AST的には表現が不正確になるがこちらの方がシンプルに書けるのでIdentに押し込める
				Fun:  ast.NewIdent("fmt.Println"),
				Args: []ast.Expr{exprStmtV},
			},
		}
	default:
		return errs.NewBadInputError("unsupported expression type")
	}
	if err := e.addImportPath(types.PkgName("fmt")); err != nil {
		return err
	}
	mainFunc.Body.List = append(mainFunc.Body.List, exprStmt)
	return nil
}

func (e *Executor) appendAssignStmtToMainFuncBody(assignStmt *ast.AssignStmt, mainFunc *ast.FuncDecl) error {
	assignStmtRHS := assignStmt.Rhs[0]
	switch assignStmtRHS.(type) {
	case *ast.BasicLit:
		// 右辺が基本リテラルの場合は特に何もしない
	default:
		selectorBase := extractSelectorBaseFromExpr(assignStmtRHS)
		if !e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
			if err := e.addImportPath(types.PkgName(selectorBase)); err != nil {
				return err
			}
		}

	}
	mainFunc.Body.List = append(mainFunc.Body.List, assignStmt)
	if assignStmt.Tok == token.DEFINE {
		for _, lhsExpr := range assignStmt.Lhs {
			declName := types.DeclName(lhsExpr.(*ast.Ident).Name)
			mainFunc.Body.List = append(mainFunc.Body.List, blankAssignStmt(declName))
		}
	}
	return nil
}

func (e *Executor) appendDeclStmtToMainFuncBody(declStmt *ast.DeclStmt, mainFunc *ast.FuncDecl) error {
	declStmtRHS := declStmt.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0]
	switch declStmtRHS.(type) {
	case *ast.BasicLit:
		// 右辺が基本リテラルの場合は特に何もしない
	default:
		selectorBase := extractSelectorBaseFromExpr(declStmtRHS)
		if !e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
			if err := e.addImportPath(types.PkgName(selectorBase)); err != nil {
				return err
			}
		}
	}
	mainFunc.Body.List = append(mainFunc.Body.List, declStmt)
	for _, name := range declStmt.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names {
		declName := types.DeclName(name.Name)
		mainFunc.Body.List = append(mainFunc.Body.List, blankAssignStmt(declName))
	}
	return nil
}

// cleanErrLineFromSessionSrcでエラー時に追加していたimportPathを削除するために使う
// その1replセッション内ごとに一つだけ保持
var importPathAddedInSession types.ImportPath

func (e *Executor) addImportPath(pkgName types.PkgName) error {
	importPath, err := e.resolve(pkgName)
	if err != nil {
		return err
	}

	for _, importSpec := range e.sessionSrc.Imports {
		if importSpec.Path.Value == string(importPath) {
			return nil
		}
	}
	importPathAddedInSession = importPath

	newImportSpec := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: string(importPath),
		},
	}
	e.sessionSrc.Imports = append(e.sessionSrc.Imports, newImportSpec)

	// DeclsにGenDecl(import)があればそこに追加、なければ新規作成
	var importDecl *ast.GenDecl
	for _, decl := range e.sessionSrc.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			importDecl = genDecl
			break
		}
	}
	if importDecl != nil {
		importDecl.Specs = append(importDecl.Specs, newImportSpec)
	} else {
		importDecl = &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: []ast.Spec{newImportSpec},
		}
		e.sessionSrc.Decls = append([]ast.Decl{importDecl}, e.sessionSrc.Decls...)
	}
	return nil
}

func extractSelectorBaseFromExpr(expr ast.Expr) string {
	switch exprV := expr.(type) {
	case *ast.SelectorExpr:
		return extractSelectorBaseFromExpr(exprV.X)
	case *ast.CompositeLit:
		if sel, ok := exprV.Type.(*ast.SelectorExpr); ok {
			return extractSelectorBaseFromExpr(sel.X)
		}
	case *ast.UnaryExpr:
		if comp, ok := exprV.X.(*ast.CompositeLit); ok {
			if sel, ok := comp.Type.(*ast.SelectorExpr); ok {
				return extractSelectorBaseFromExpr(sel.X)
			}
		}
	case *ast.CallExpr:
		return extractSelectorBaseFromExpr(exprV.Fun)
	case *ast.Ident:
		return exprV.Name
	}
	return ""
}

func formatCmdErrMsg(cmdErrMsg string) string {
	cmdErrLines := strings.Split(cmdErrMsg, "\n")
	var formattedCmdErrLines []string

	cmdVirtualPkgPattern := regexp.MustCompile(`^# command-line-arguments$`)
	tmpFilePathPattern := regexp.MustCompile(`\./?\d+_gonsole_tmp\.go:\d+:\d+:\s*`)
	var cmdErrCount int
	for _, cmdErrLine := range cmdErrLines {
		// 仮想パッケージに関するエラー行はスキップ
		if cmdVirtualPkgPattern.MatchString(cmdErrLine) || cmdErrLine == "" {
			continue
		}

		// 一時ファイルパス部分を削除
		cmdErrLine = tmpFilePathPattern.ReplaceAllString(cmdErrLine, "")

		// インデントがない行はエラーの件数としてカウントする
		if !strings.HasPrefix(cmdErrLine, "\t") {
			cmdErrCount++
		}

		formattedCmdErrLines = append(formattedCmdErrLines, cmdErrLine)
	}
	formattedCmdErrLine := strings.Join(formattedCmdErrLines, "\n")
	return fmt.Sprintf("\n%d errors found\n\n%s\n\n", cmdErrCount, formattedCmdErrLine)
}

func (e *Executor) cleanCallExprFromSessionSrc() (isCleaned bool) {
	mainFunc := getMainFunc(e.sessionSrc)
	body := mainFunc.Body.List
	lastExprStmt, ok := body[len(body)-1].(*ast.ExprStmt)
	if !ok {
		return false
	}
	if _, ok := lastExprStmt.X.(*ast.CallExpr); ok {
		mainFunc.Body.List = body[:len(body)-1]

		// 該当packageを利用している宣言がなければpackage importを削除する
		selectorBase := extractSelectorBaseFromExpr(lastExprStmt.X)
		if !e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
			for _, decl := range e.declRegistry.Decls() {
				if decl.RHS().PkgName() == types.PkgName(selectorBase) {
					e.sessionSrc.Imports = slices.DeleteFunc(e.sessionSrc.Imports, func(importSpec *ast.ImportSpec) bool {
						return importSpec.Path.Value == string(importPathAddedInSession)
					})
					importPathAddedInSession = ""

					for _, decl := range e.sessionSrc.Decls {
						if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
							genDecl.Specs = slices.DeleteFunc(genDecl.Specs, func(spec ast.Spec) bool {
								importSpec := spec.(*ast.ImportSpec)
								return importSpec.Path.Value == string(importPathAddedInSession)
							})
							break
						}
					}
				}
			}
		}

		// fmt importを削除する
		e.sessionSrc.Imports = slices.DeleteFunc(e.sessionSrc.Imports, func(importSpec *ast.ImportSpec) bool {
			return importSpec.Path.Value == `"fmt"`
		})
		for _, decl := range e.sessionSrc.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
				genDecl.Specs = slices.DeleteFunc(genDecl.Specs, func(spec ast.Spec) bool {
					importSpec := spec.(*ast.ImportSpec)
					return importSpec.Path.Value == `"fmt"`
				})
				break
			}
		}
		return true
	}
	return false
}

func (e *Executor) cleanErrLineFromSessionSrc(errMsg string, fset *token.FileSet) error {
	// エラーメッセージからエラー行番号を抽出する
	tmpFilePattern := regexp.MustCompile(`\./?\d+_gonsole_tmp\.go:(\d+):(\d+)`)
	matches := tmpFilePattern.FindStringSubmatch(errMsg)
	errLine, err := strconv.Atoi(matches[1])
	if err != nil {
		return errs.NewInternalError("failed to convert error line to int").Wrap(err)
	}

	// セッションソースからエラー行を削除する
	mainFunc := getMainFunc(e.sessionSrc)
	cleanMainFuncBody := []ast.Stmt{}

	var errDeclNames []types.DeclName // エラー行で定義された変数名リストを保持する

	for _, stmt := range mainFunc.Body.List {
		stmtPos := fset.Position(stmt.Pos())
		if stmtPos.Line == errLine {
			var selectorBase string
			switch errStmtV := stmt.(type) {
			case *ast.AssignStmt:
				selectorBase = extractSelectorBaseFromExpr(errStmtV.Rhs[0])
				errDeclNames = append(errDeclNames, types.DeclName(errStmtV.Lhs[0].(*ast.Ident).Name))
			case *ast.DeclStmt:
				selectorBase = extractSelectorBaseFromExpr(errStmtV.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0])
				errDeclNames = append(errDeclNames, types.DeclName(errStmtV.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0].Name))
			case *ast.ExprStmt:
				selectorBase = extractSelectorBaseFromExpr(errStmtV.X)
			}

			// 該当packageを利用している宣言がなければpackage importを削除する
			if !e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
				for _, decl := range e.declRegistry.Decls() {
					if decl.RHS().PkgName() == types.PkgName(selectorBase) {
						e.sessionSrc.Imports = slices.DeleteFunc(e.sessionSrc.Imports, func(importSpec *ast.ImportSpec) bool {
							return importSpec.Path.Value == string(importPathAddedInSession)
						})

						for _, decl := range e.sessionSrc.Decls {
							if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
								genDecl.Specs = slices.DeleteFunc(genDecl.Specs, func(spec ast.Spec) bool {
									importSpec := spec.(*ast.ImportSpec)
									return importSpec.Path.Value == string(importPathAddedInSession)
								})
								break
							}
						}
					}
				}
				importPathAddedInSession = ""
			}

			continue
		}

		// エラー行で定義された変数をブランク代入している行を削除する
		if len(errDeclNames) > 0 {
			blankAssignmentDeclName := types.DeclName(stmt.(*ast.AssignStmt).Rhs[0].(*ast.Ident).Name)
			for _, errDeclName := range errDeclNames {
				if blankAssignmentDeclName == errDeclName {
					continue
				}
			}
			errDeclNames = slices.DeleteFunc(errDeclNames, func(errDeclName types.DeclName) bool {
				return blankAssignmentDeclName == errDeclName
			})
		}

		cleanMainFuncBody = append(cleanMainFuncBody, stmt)
	}
	mainFunc.Body.List = cleanMainFuncBody
	return nil
}

// ================以下に関数を定義する======================

func initSessionSrc() *ast.File {
	return &ast.File{
		Name: &ast.Ident{Name: "main"},
		Decls: []ast.Decl{
			&ast.FuncDecl{
				Name: &ast.Ident{Name: "main"},
				Type: &ast.FuncType{
					Params:  &ast.FieldList{List: nil},
					Results: nil,
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{},
				},
			},
		},
	}
}

// go/parserを使って入力文をASTにパースさせる
func parseInput(input string) (ast.Stmt, error) {
	// 入力値をmain関数でラップしてparseする
	fset := token.NewFileSet()
	wrappedInput := "package main\nfunc main() {\n" + input + "\n}"
	wrappedInputAst, err := parser.ParseFile(fset, "", wrappedInput, parser.AllErrors)
	if err != nil {
		return nil, errs.NewBadInputError("invalid input syntax")
	}

	var inputStmtAst ast.Stmt
	for _, decl := range wrappedInputAst.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			inputStmtAst = funcDecl.Body.List[0]
		}
	}
	return inputStmtAst, nil
}

func blankAssignStmt(name types.DeclName) *ast.AssignStmt {
	blankAssign := ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{&ast.Ident{Name: string(name)}},
	}
	return &blankAssign
}

func printCmdOutput(cmdOut []byte) {
	cmdOutText := string(cmdOut)

	const greenColor = "\033[32m"
	const colorReset = "\033[0m"
	fmt.Printf("\n%s%s%s\n", greenColor, cmdOutText, colorReset)
}

func getMainFunc(file *ast.File) *ast.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			return fn
		}
	}
	return nil
}
