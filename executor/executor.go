package executor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"time"

	"os/exec"
	"regexp"
	"strings"

	"github.com/kakkky/gonsole/decl_registry"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
)

type Executor struct {
	modPath    string
	registry   *decl_registry.DeclRegistry
	sessionSrc *ast.File
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func NewExecutor(registry *decl_registry.DeclRegistry) (*Executor, error) {
	modPath, err := getGoModPath("go.mod")
	if err != nil {
		return nil, err
	}
	return &Executor{
		modPath:    modPath,
		registry:   registry,
		sessionSrc: initSessionSrc(),
	}, nil
}

func initSessionSrc() *ast.File {
	return &ast.File{
		Name:    &ast.Ident{Name: "main"},
		Imports: []*ast.ImportSpec{}, // import()
		Decls: []ast.Decl{
			&ast.FuncDecl{
				Name: &ast.Ident{Name: "main"},
				Type: &ast.FuncType{
					Params:  &ast.FieldList{List: nil},
					Results: nil,
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.BasicLit{
								Kind:  token.COMMENT,
								Value: "// gonsole session started",
							},
						},
					},
				},
			},
		},
	}
}

func (e *Executor) Execute(input string) {
	defer func() { _ = recover() }()

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
	defer cleanup()

	fset := token.NewFileSet()

	// 一時ファイルにflushする
	if err := e.flushSessionSrc(tmpFile, fset); err != nil {
		errs.HandleError(err)
	}

	// 一時ファイルを実行する
	cmd := exec.Command("go", "run", tmpFileName)
	cmdStdout, cmdStderr := bindCmdStdOutput(cmd)
	if err := cmd.Run(); err != nil {
		// 実行時のエラー出力を整形して表示する
		cmdErrMsg := cmdStderr.String()
		formmatted := formatCmdErrMsg(cmdErrMsg)
		errs.HandleError(errs.NewBadInputError(formmatted))

		// エラー行を削除する
		if err := e.cleanErrLineFromSessionSrc(cmdErrMsg, fset); err != nil {
			errs.HandleError(err)
		}
		e.flushSessionSrc(tmpFile, fset)
	}

	// 実行結果を表示する
	printCmdOutput(cmdStdout)

	// 変数エントリに登録する
	if err := e.registry.Register(input); err != nil {
		errs.HandleError(err)
	}

	// 最後の式呼び出しを削除してflushする
	e.cleanCallExprFromSessionSrc(tmpFile)
	e.flushSessionSrc(tmpFile, fset)
}

func (e *Executor) writeInSessionSrc(input string) error {
	inputStmtAst, err := parseInput(input)
	if err != nil {
		return err
	}

	mainFuncBody := e.sessionSrc.Decls[0].(*ast.FuncDecl).Body.List

	switch inputStmtV := inputStmtAst.(type) {
	case *ast.ExprStmt:
		if err := e.appendExprStmtToMainFuncBody(inputStmtV, mainFuncBody); err != nil {
			return err
		}
	case *ast.AssignStmt:
		if err := e.appendAssignStmtToMainFuncBody(inputStmtV, mainFuncBody); err != nil {
			return err
		}
	case *ast.DeclStmt:
		if err := e.appendDeclStmtToMainFuncBody(inputStmtV, mainFuncBody); err != nil {
			return err
		}
	}

	return nil
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

	// 入力文をASTとして取得
	var inputStmtAst ast.Stmt
	for _, decl := range wrappedInputAst.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			inputStmtAst = funcDecl.Body.List[0]
		}
	}
	return inputStmtAst, nil
}

func (e *Executor) appendExprStmtToMainFuncBody(exprStmt *ast.ExprStmt, mainFuncBody []ast.Stmt) error {
	selectorExprStmt := exprStmt.X.(*ast.SelectorExpr)

	selectorBase := extractSelectorBaseFromExpr(selectorExprStmt)
	if e.registry.IsRegisteredDecl(types.DeclName(selectorBase)) {
		if err := e.addImportPath(types.PkgName(selectorBase)); err != nil {
			return err
		}
	}

	// 標準出力に結果を表示するように書き換え
	exprStmt = &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  ast.NewIdent("fmt.Println"),
			Args: []ast.Expr{selectorExprStmt},
		},
	}
	if err := e.addImportPath(types.PkgName("fmt")); err != nil {
		return err
	}
	mainFuncBody = append(mainFuncBody, exprStmt)
	return nil
}

func (e *Executor) appendAssignStmtToMainFuncBody(assignStmt *ast.AssignStmt, mainFuncBody []ast.Stmt) error {
	assignStmtRhs := assignStmt.Rhs[0]
	switch assignStmtRhs.(type) {
	case *ast.BasicLit:
		// 右辺が基本リテラルの場合は特に何もしない
	default:
		selectorBase := extractSelectorBaseFromExpr(assignStmtRhs)
		if e.registry.IsRegisteredDecl(types.DeclName(selectorBase)) {
			if err := e.addImportPath(types.PkgName(selectorBase)); err != nil {
				return err
			}
		}

		mainFuncBody = append(mainFuncBody, assignStmt, blankAssignStmt(assignStmtRhs))
	}
	return nil
}

func (e *Executor) appendDeclStmtToMainFuncBody(declStmt *ast.DeclStmt, mainFuncBody []ast.Stmt) error {
	declStmtRhs := declStmt.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0]
	switch declStmtRhs.(type) {
	case *ast.BasicLit:
		// 右辺が基本リテラルの場合は特に何もしない
	default:
		selectorBase := extractSelectorBaseFromExpr(declStmtRhs)
		if e.registry.IsRegisteredDecl(types.DeclName(selectorBase)) {
			if err := e.addImportPath(types.PkgName(selectorBase)); err != nil {
				return err
			}
		}

		mainFuncBody = append(mainFuncBody, declStmt, blankAssignStmt(declStmtRhs))
	}
	return nil
}

func (e *Executor) addImportPath(pkgName types.PkgName) error {
	// パッケージパスを探索
	importPath, err := e.resolveImportPathForAdd(pkgName) // TODO: ここのロジック後で整理する
	if err != nil {
		return err
	}
	importPathQuoted := fmt.Sprintf(`"%s"`, importPath)
	for _, importSpec := range e.sessionSrc.Imports {
		if importSpec.Path.Value == importPathQuoted {
			// すでにimportされている場合は何もしない
			return nil
		}
	}

	e.sessionSrc.Imports = append(e.sessionSrc.Imports, &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: importPathQuoted,
		},
	})
	return nil
}

// extractSelectorBaseFromExpr は式からselectorBaseを抽出する
func extractSelectorBaseFromExpr(expr ast.Expr) string {
	switch exprV := expr.(type) {
	// セレクタ式の場合（pkg.Name）
	case *ast.SelectorExpr:
		return exprV.X.(*ast.Ident).Name
	// 複合リテラルの場合（pkg.Type{}）
	case *ast.CompositeLit:
		return exprV.Type.(*ast.SelectorExpr).X.(*ast.Ident).Name
	// 演算子つきの場合（&pkg.Type{}）
	case *ast.UnaryExpr:
		return exprV.X.(*ast.CompositeLit).Type.(*ast.SelectorExpr).X.(*ast.Ident).Name
	// 関数呼び出しの場合（pkg.Func()）
	case *ast.CallExpr:
		return exprV.Fun.(*ast.SelectorExpr).X.(*ast.Ident).Name
	}
	return ""
}

func blankAssignStmt(expr ast.Expr) *ast.AssignStmt {
	blankAssign := ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{expr},
	}
	return &blankAssign
}

func makeTmpFile() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
	prefix := time.Now().Unix()
	tmpFileName = fmt.Sprintf(".%d_gonsole_tmp.go", prefix)

	file, err := os.Create(tmpFileName)
	defer file.Close()

	if err != nil {
		return nil, "", nil, errs.NewInternalError("failed to create temporary file").Wrap(err)
	}

	cleanup = func() { os.Remove(tmpFileName) }

	return file, tmpFileName, cleanup, nil
}

func (e *Executor) flushSessionSrc(file *os.File, fset *token.FileSet) error {
	if err := format.Node(file, fset, e.sessionSrc); err != nil {
		return errs.NewInternalError("failed to format AST node").Wrap(err)
	}
	return nil
}

func bindCmdStdOutput(cmd *exec.Cmd) (cmdStdout, cmdStderr *bytes.Buffer) {
	cmdStdout = &bytes.Buffer{}
	cmdStderr = &bytes.Buffer{}
	cmd.Stdout = cmdStdout
	cmd.Stderr = cmdStderr
	return cmdStdout, cmdStderr
}

func formatCmdErrMsg(cmdErrMsg string) string {
	cmdErrLines := strings.Split(cmdErrMsg, "\n")
	var formattedCmdErrLines []string

	cmdVirtualPkgPattern := regexp.MustCompile(`^# command-line-arguments$`)
	tmpFilePathPattern := regexp.MustCompile(`\.\d+_gonsole_tmp\.go:\d+:\d+:\s*`)
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

func printCmdOutput(cmdStdout *bytes.Buffer) {
	cmdOutputText := cmdStdout.String()

	const greenColor = "\033[32m"
	const colorReset = "\033[0m"
	fmt.Printf("\n%s%s%s\n", greenColor, cmdOutputText, colorReset)
}

func (e *Executor) cleanCallExprFromSessionSrc(file *os.File) {
	mainFunc := e.sessionSrc.Decls[0].(*ast.FuncDecl)
	body := mainFunc.Body.List
	lastExprStmt := body[len(body)-1].(*ast.ExprStmt)
	if _, ok := lastExprStmt.X.(*ast.CallExpr); ok {
		mainFunc.Body.List = body[:len(body)-1]
	}
}

func (e *Executor) cleanErrLineFromSessionSrc(errMsg string, fset *token.FileSet) error {

}
