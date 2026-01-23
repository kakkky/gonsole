package executor

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"slices"
	"strconv"
	"time"

	"os/exec"
	"regexp"
	"strings"

	"github.com/kakkky/go-prompt"
	"github.com/kakkky/gonsole/decl_registry"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/stdpkg"
	"github.com/kakkky/gonsole/types"
)

// ExecutorはREPLセッション内でのコード実行を担う
// go-promptのExecutorインターフェースを実装する
type Executor struct {
	declRegistry *decl_registry.DeclRegistry
	sessionSrc   *ast.File
}

// NewExecutor はExecutorのインスタンスを生成する
func NewExecutor(declRegistry *decl_registry.DeclRegistry) (*Executor, error) {
	return &Executor{
		declRegistry: declRegistry,
		sessionSrc:   initSessionSrc(),
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
		}
	}()

	if input == "" {
		return
	}

	// 入力文をセッションに書き込む
	if err := e.writeInSessionSrc(input); err != nil {
		errs.HandleError(err)
	}

	// 一時ファイルを作成
	tmpFile, tmpFileName, cleanup, err := makeTmpFile()
	if err != nil {
		errs.HandleError(err)
	}
	defer tmpFile.Close()
	defer cleanup()

	fset := token.NewFileSet()

	// 一時ファイルにflushする
	if err := e.flushSessionSrc(tmpFile, fset); err != nil {
		errs.HandleError(err)
	}

	// 一時ファイルを実行する
	cmd := exec.Command("go", "run", tmpFileName)
	cmdOut, cmdErr := cmd.Output()
	if cmdErr != nil {
		// 実行時のエラー出力を整形して表示する
		cmdErrMsg := string(cmdErr.(*exec.ExitError).Stderr)

		formatted := formatCmdErrMsg(cmdErrMsg)
		errs.HandleError(errs.NewBadInputError(formatted))

		// エラー行を削除する
		if err := e.cleanErrLineFromSessionSrc(cmdErrMsg, fset); err != nil {
			errs.HandleError(err)
		}
		if err := e.flushSessionSrc(tmpFile, fset); err != nil {
			errs.HandleError(err)
		}
	}

	// 実行結果を表示する
	printCmdOutput(cmdOut)

	// 変数エントリに登録する
	if err := e.declRegistry.Register(input); err != nil {
		errs.HandleError(err)
	}

	// 最後の式呼び出しを削除してflushする
	e.cleanCallExprFromSessionSrc()
	if err := e.flushSessionSrc(tmpFile, fset); err != nil {
		errs.HandleError(err)
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
				Fun:  ast.NewIdent("fmt.Println"),
				Args: []ast.Expr{exprStmtV},
			},
		}
	case *ast.Ident:
		exprStmt = &ast.ExprStmt{
			X: &ast.CallExpr{
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
	assignStmtLhs := assignStmt.Lhs[0].(*ast.Ident)
	mainFunc.Body.List = append(mainFunc.Body.List, assignStmt, blankAssignStmt(types.DeclName(assignStmtLhs.Name)))
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
	assignStmtLhs := declStmt.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0]
	mainFunc.Body.List = append(mainFunc.Body.List, declStmt, blankAssignStmt(types.DeclName(assignStmtLhs.Name)))
	return nil
}

// cleanErrLineFromSessionSrcでエラー時に追加していたimportPathを削除するために使う
// その1replセッション内ごとに一つだけ保持
var importPathAddedInSession types.ImportPath

func (e *Executor) addImportPath(pkgName types.PkgName) error {
	importPath, err := resolveImportPath(pkgName)
	if err != nil {
		return err
	}

	for _, importSpec := range e.sessionSrc.Imports {
		if importSpec.Path.Value == string(*importPath) {
			return nil
		}
	}
	importPathAddedInSession = *importPath

	newImportSpec := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: string(*importPath),
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

func (e *Executor) flushSessionSrc(file *os.File, fset *token.FileSet) error {
	if _, err := file.Seek(0, 0); err != nil {
		return errs.NewInternalError("failed to seek file").Wrap(err)
	}
	if err := file.Truncate(0); err != nil {
		return errs.NewInternalError("failed to truncate file").Wrap(err)
	}
	if err := format.Node(file, fset, e.sessionSrc); err != nil {
		return errs.NewInternalError("failed to format AST node").Wrap(err)
	}
	return nil
}

func formatCmdErrMsg(cmdErrMsg string) string {
	cmdErrLines := strings.Split(cmdErrMsg, "\n")
	var formattedCmdErrLines []string

	cmdVirtualPkgPattern := regexp.MustCompile(`^# command-line-arguments$`)
	tmpFilePathPattern := regexp.MustCompile(`\d+_gonsole_tmp\.go:\d+:\d+:\s*`)
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

func (e *Executor) cleanCallExprFromSessionSrc() {
	mainFunc := getMainFunc(e.sessionSrc)
	body := mainFunc.Body.List
	lastExprStmt, ok := body[len(body)-1].(*ast.ExprStmt)
	if !ok {
		return
	}
	if _, ok := lastExprStmt.X.(*ast.CallExpr); ok {
		mainFunc.Body.List = body[:len(body)-1]

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
	}
}

func (e *Executor) cleanErrLineFromSessionSrc(errMsg string, fset *token.FileSet) error {
	// エラーメッセージからエラー行番号を抽出する
	tmpFilePattern := regexp.MustCompile(`\.\d+_gonsole_tmp\.go:(\d+):(\d+)`)
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

			if !e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
				e.sessionSrc.Imports = slices.DeleteFunc(e.sessionSrc.Imports, func(importSpec *ast.ImportSpec) bool {
					return importSpec.Path.Value == string(importPathAddedInSession)
				})
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
		return nil, errs.NewInternalError("failed to parse input source").Wrap(err)
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

func makeTmpFile() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
	prefix := time.Now().Unix()
	tmpFileName = fmt.Sprintf("%d_gonsole_tmp.go", prefix)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return nil, "", nil, errs.NewInternalError("failed to create temporary file").Wrap(err)
	}

	cleanup = func() { os.Remove(tmpFileName) }

	return file, tmpFileName, cleanup, nil
}

func printCmdOutput(cmdOut []byte) {
	cmdOutText := string(cmdOut)

	const greenColor = "\033[32m"
	const colorReset = "\033[0m"
	fmt.Printf("\n%s%s%s\n", greenColor, cmdOutText, colorReset)
}

func resolveImportPath(pkgName types.PkgName) (*types.ImportPath, error) {
	var importPathCandidates []types.ImportPath

	if stdpkgImportPath, found := stdpkg.IsStandardPackage(pkgName); found {
		return &stdpkgImportPath, nil
	}

	cmd := exec.Command("go", "list", "./...")
	cmdOut, err := cmd.Output()
	if err != nil {
		return nil, errs.NewInternalError("failed to resolve import path").Wrap(err)
	}

	allImportPaths := strings.Split(string(cmdOut), "\n")
	for _, importPath := range allImportPaths {
		if importPath == "" {
			continue
		}
		if types.PkgName(path.Base(importPath)) == pkgName {
			quoted := fmt.Sprintf(`"%s"`, importPath)
			importPathCandidates = append(importPathCandidates, types.ImportPath(quoted))
		}
	}

	if len(importPathCandidates) == 1 {
		return &importPathCandidates[0], nil
	}

	// 複数候補がある場合はユーザーに選択させる
	selectedImportPath, err := selectImportPathRepl(importPathCandidates)
	if err != nil {
		return nil, err
	}
	return selectedImportPath, nil
}

func selectImportPathRepl(importPathCandidates []types.ImportPath) (*types.ImportPath, error) {
	toBlue := func(s string) string {
		colorBlue := "\033[94m"
		colorReset := "\033[0m"
		return fmt.Sprintf("%s%s%s", colorBlue, s, colorReset)
	}
	completer := func(d prompt.Document) []prompt.Suggest {
		suggests := make([]prompt.Suggest, len(importPathCandidates))
		for i, importPath := range importPathCandidates {
			suggests[i] = prompt.Suggest{Text: string(importPath)}
		}
		return suggests
	}

	fmt.Println(toBlue("\nMultiple import candidates found.\n\nUse Tab key to select import path.\n\n"))
	fmt.Print(toBlue("\n>>> "))
	selectedImportPath := prompt.Input(
		"",
		completer, prompt.OptionShowCompletionAtStart(),
		prompt.OptionPreviewSuggestionTextColor(prompt.Turquoise),
		prompt.OptionInputTextColor(prompt.Turquoise),
	)
	if selectedImportPath == "" {
		return nil, errs.NewBadInputError("no import path selected")
	}
	if !slices.Contains(importPathCandidates, types.ImportPath(selectedImportPath)) {
		return nil, errs.NewBadInputError("invalid import path selected")
	}

	sip := types.ImportPath(selectedImportPath)
	return &sip, nil
}

func getMainFunc(file *ast.File) *ast.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			return fn
		}
	}
	return nil
}
