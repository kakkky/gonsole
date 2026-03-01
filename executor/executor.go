package executor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"path/filepath"
	"runtime"
	"slices"

	"os/exec"
	"regexp"
	"strings"

	"github.com/kakkky/gonsole/declregistry"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/symbols"
	"github.com/kakkky/gonsole/types"
	"golang.org/x/tools/go/packages"
)

// Executor はREPLセッション内でのコード実行を担う
// go-promptのExecutorインターフェースを実装する
type Executor struct {
	declRegistry *declregistry.DeclRegistry
	sessionSrc   *ast.File
	symbolIndex  *symbols.SymbolIndex
	filer
	commander
	importPathResolver
}

// NewExecutor はExecutorのインスタンスを生成する
func NewExecutor(declRegistry *declregistry.DeclRegistry, symbolIndex *symbols.SymbolIndex) (*Executor, error) {
	commander := newDefaultCommander()
	return &Executor{
		declRegistry:       declRegistry,
		sessionSrc:         initSessionSrc(),
		symbolIndex:        symbolIndex,
		filer:              newDefaultFiler(),
		commander:          commander,
		importPathResolver: newDefaultImportPathResolver(commander),
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
	inputStmt, err := parseInput(input)
	if err != nil {
		errs.HandleError(err)
		return
	}
	if err := e.writeInSessionSrc(inputStmt); err != nil {
		errs.HandleError(err)
		return
	}
	defer clearImportPathAddedInSession()

	// 一時ファイルを作成
	sessionSrcDir, cleanupSessionSrcDir, err := e.createSessionSrcDir()
	if err != nil {
		errs.HandleError(err)
		return
	}
	defer cleanupSessionSrcDir()
	sessionSrcFile, sessionSrcFilePath, cleanupSessionSrcFile, err := e.createSessionSrcFile()
	if err != nil {
		errs.HandleError(err)
		return
	}
	defer func() {
		if err := sessionSrcFile.Close(); err != nil {
			errs.HandleError(err)
		}
	}()
	defer cleanupSessionSrcFile()

	fset := token.NewFileSet()

	// 一時ファイルにflushする
	if err := e.flush(e.sessionSrc, sessionSrcFile, fset); err != nil {
		errs.HandleError(err)
		return
	}

	// go.modファイルを準備する
	goModContent, err := e.goModContent()
	if err != nil {
		errs.HandleError(err)
		return
	}
	goModFilePath, cleanupGoModFile, err := e.prepareGoModFile(sessionSrcDir, goModContent)
	if err != nil {
		errs.HandleError(err)
		return
	}
	defer cleanupGoModFile()

	// go.mod tidyを実行して、go.sumファイルを生成する
	if err := e.execGoModTidy(sessionSrcDir); err != nil {
		errs.HandleError(err)
		return
	}
	defer func() {
		if err := e.cleanupGoSumFile(sessionSrcDir); err != nil {
			errs.HandleError(err)
		}
	}()
	// 一時ファイルを実行する
	cmdOut, cmdErr := e.execGoRun(goModFilePath, sessionSrcFilePath)
	if cmdErr != nil {
		// 実行時のエラー出力を整形して表示する
		cmdErrMsg := string(cmdErr.(*exec.ExitError).Stderr)

		formatted := formatCmdErrMsg(sessionSrcFilePath, cmdErrMsg)
		errs.HandleError(errs.NewBadInputError(formatted))

		// エラー行を削除する
		if err := e.cleanErrElmFromSessionSrc(); err != nil {
			errs.HandleError(err)
		}

		if err := e.flush(e.sessionSrc, sessionSrcFile, fset); err != nil {
			errs.HandleError(err)
		}

		return
	}

	// 実行結果を表示する
	if len(cmdOut) > 0 {
		fmt.Print(string(cmdOut))
	}

	// sessionSrcから式呼び出しを削除する（式呼び出しは実行結果の表示のためだけに追加しているため、実行後は削除する）
	if cleaned := e.cleanCallExprFromSessionSrc(); cleaned {
		return // 式呼び出しだったということは、以降の変数登録処理は不要
	}

	// セッションソースファイルから解決された型情報付きのASTを取得
	pkg, err := loadsessionSrcFile(sessionSrcFilePath)
	if err != nil {
		errs.HandleError(err)
		return
	}

	// 今セッションで入力された宣言文のASTを抽出する
	inputStmtWithTypeInfo, err := extractInputStmtWithTypeInfo(pkg.Syntax[0])
	if err != nil {
		errs.HandleError(err)
		return
	}
	// 入力文の 宣言名：左辺情報(式)のmapを抽出する
	declLHSMap := extractDeclLHSMapFromInputStmt(inputStmtWithTypeInfo)

	// 宣言名の数分だけ宣言情報をレジストリに登録
	for declName, declLHS := range declLHSMap {
		declType, err := detectGoType(pkg.TypesInfo, declLHS)
		if err != nil {
			errs.HandleError(err)
		}
		if err := e.declRegistry.Register(declName, declType); err != nil {
			errs.HandleError(err)
			return
		}
	}
}

func (e *Executor) writeInSessionSrc(inputStmt ast.Stmt) error {
	mainFunc := getMainFunc(e.sessionSrc)
	switch inputStmtV := inputStmt.(type) {
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
				Fun: ast.NewIdent("pp.Println"),
				Args: []ast.Expr{
					exprStmtV,
				},
			},
		}
		if err := e.addImportPath(types.PkgName("pp")); err != nil {
			return err
		}
	case *ast.Ident:
		exprStmt = &ast.ExprStmt{
			X: &ast.CallExpr{
				// AST的には表現が不正確になるがこちらの方がシンプルに書けるのでIdentに押し込める
				Fun: ast.NewIdent("pp.Println"),
				Args: []ast.Expr{
					exprStmtV,
				},
			},
		}
		if err := e.addImportPath(types.PkgName("pp")); err != nil {
			return err
		}
	case *ast.CallExpr:
		var returnValuesCnt int
		selectorBase := extractSelectorBaseFromExpr(exprStmtV)
		if e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
			var pkgName types.PkgName
			for _, decl := range e.declRegistry.Decls {
				if decl.Name == types.DeclName(selectorBase) {
					pkgName = decl.TypePkgName
				}
			}
			methodDeclName := types.DeclName(exprStmtV.Fun.(*ast.SelectorExpr).Sel.Name)
			var methodSets []symbols.MethodSet
			if e.symbolIndex != nil {
				methodSets = e.symbolIndex.Methods[pkgName]
			}
			for _, methodSet := range methodSets {
				if methodSet.Name == methodDeclName {
					returnValuesCnt = len(methodSet.Returns)
				}
			}
		} else {
			if err := e.addImportPath(types.PkgName(selectorBase)); err != nil {
				return err
			}
			pkgName := types.PkgName(selectorBase)
			selExpr := exprStmtV.Fun.(*ast.SelectorExpr)
			declName := types.DeclName(selExpr.Sel.Name)
			switch selExpr.X.(type) {
			case *ast.Ident:
				// 直接のパッケージ関数呼び出し: pkg.Func()
				var funcSets []symbols.FuncSet
				if e.symbolIndex != nil {
					funcSets = e.symbolIndex.Funcs[pkgName]
				}
				for _, funcSet := range funcSets {
					if funcSet.Name == declName {
						returnValuesCnt = len(funcSet.Returns)
					}
				}
			default:
				// メソッドチェーン: pkg.Constructor(...).Method()
				var methodSets []symbols.MethodSet
				if e.symbolIndex != nil {
					methodSets = e.symbolIndex.Methods[pkgName]
				}
				for _, methodSet := range methodSets {
					if methodSet.Name == declName {
						returnValuesCnt = len(methodSet.Returns)
					}
				}
			}
		}

		// 返り値0(void)の場合は返り値がないのでfmt.Printfでwrapしない
		if returnValuesCnt == 0 {
			break
		}
		exprStmt = composePrintClosure(exprStmtV, returnValuesCnt)
		if err := e.addImportPath(types.PkgName("pp")); err != nil {
			return err
		}
	default:
		return errs.NewBadInputError("unsupported expression type")
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

	// ppパッケージはimportPathAddedInSessionには設定しない。
	if pkgName != "pp" {
		importPathAddedInSession = importPath
	}

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

func formatCmdErrMsg(filePath string, cmdErrMsg string) string {
	cmdErrLines := strings.Split(cmdErrMsg, "\n")
	var formattedCmdErrLines []string

	cmdVirtualPkgPattern := regexp.MustCompile(`^# command-line-arguments$`)
	// 新しい一時ファイル名 gonsole_session_src/session_src.go に対応
	sessionSrcFilePathPattern := regexp.MustCompile(filePath + `:\d+:\d+:\s*`)
	var cmdErrCount int
	for _, cmdErrLine := range cmdErrLines {
		// 仮想パッケージに関するエラー行はスキップ
		if cmdVirtualPkgPattern.MatchString(cmdErrLine) || cmdErrLine == "" {
			continue
		}

		// 一時ファイルパス部分を削除
		cmdErrLine = sessionSrcFilePathPattern.ReplaceAllString(cmdErrLine, "")

		// インデントがない行はエラーの件数としてカウントする
		if !strings.HasPrefix(cmdErrLine, "\t") {
			cmdErrCount++
		}

		formattedCmdErrLines = append(formattedCmdErrLines, cmdErrLine)
	}
	formattedCmdErrLine := strings.Join(formattedCmdErrLines, "\n")
	return fmt.Sprintf("\n%d errors found\n\n%s\n\n", cmdErrCount, formattedCmdErrLine)
}

func (e *Executor) cleanCallExprFromSessionSrc() (cleaned bool) {
	mainFunc := getMainFunc(e.sessionSrc)
	body := mainFunc.Body.List
	lastExprStmt, ok := body[len(body)-1].(*ast.ExprStmt)
	if !ok {
		return
	}
	if _, ok := lastExprStmt.X.(*ast.CallExpr); !ok {
		return
	}

	callExpr := lastExprStmt.X.(*ast.CallExpr)
	var selectorBase string
	switch fun := callExpr.Fun.(type) {
	case *ast.FuncLit:
		// closure即時実行の場合
		if len(fun.Body.List) > 0 {
			if assignStmt, ok := fun.Body.List[0].(*ast.AssignStmt); ok && len(assignStmt.Rhs) > 0 {
				selectorBase = extractSelectorBaseFromExpr(assignStmt.Rhs[0])
			}
		}
	default:
		// 通常の関数呼び出し
		if len(callExpr.Args) > 0 {
			ppPrintlnFuncArgExpr := callExpr.Args[0]
			selectorBase = extractSelectorBaseFromExpr(ppPrintlnFuncArgExpr)
		}
	}

	if !e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
		var isUsed bool
		for _, decl := range e.declRegistry.Decls {
			if decl.TypePkgName == types.PkgName(selectorBase) {
				isUsed = true
				break
			}
		}
		if !isUsed {
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
	// pp importを削除する
	e.sessionSrc.Imports = slices.DeleteFunc(e.sessionSrc.Imports, func(importSpec *ast.ImportSpec) bool {
		return importSpec.Path.Value == `"github.com/k0kubun/pp/v3"`
	})
	for _, decl := range e.sessionSrc.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			genDecl.Specs = slices.DeleteFunc(genDecl.Specs, func(spec ast.Spec) bool {
				importSpec := spec.(*ast.ImportSpec)
				return importSpec.Path.Value == `"github.com/k0kubun/pp/v3"`
			})
			break
		}
	}

	mainFunc.Body.List = body[:len(body)-1]
	return true
}

func (e *Executor) cleanErrElmFromSessionSrc() error {
	mainFunc := getMainFunc(e.sessionSrc)
	cleanMainFuncBody := []ast.Stmt{}

	// 最後の要素を取得
	lastElm := mainFunc.Body.List[len(mainFunc.Body.List)-1]

	var selectorBase string
	switch lastElmV := lastElm.(type) {
	case *ast.AssignStmt:
		// 最後の要素がブランク代入の場合、ブランク代入とその前の宣言文を削除する
		if ident, ok := lastElmV.Lhs[0].(*ast.Ident); ok && ident.Name == "_" {
			// 最後から２番目の要素が宣言文なので、そこからselectorBaseを抽出する
			prevElm := mainFunc.Body.List[len(mainFunc.Body.List)-2]
			switch prevElmV := prevElm.(type) {
			case *ast.AssignStmt:
				selectorBase = extractSelectorBaseFromExpr(prevElmV.Rhs[0])
			case *ast.DeclStmt:
				selectorBase = extractSelectorBaseFromExpr(prevElmV.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0])
			}
			cleanMainFuncBody = mainFunc.Body.List[:len(mainFunc.Body.List)-2]
		}
	case *ast.ExprStmt:
		selectorBase = extractSelectorBaseFromExpr(lastElmV.X)
		cleanMainFuncBody = mainFunc.Body.List[:len(mainFunc.Body.List)-1]
	}

	mainFunc.Body.List = cleanMainFuncBody

	if !e.declRegistry.IsRegisteredDecl(types.DeclName(selectorBase)) {
		// 該当packageを利用している宣言がなければpackage importを削除する
		var isUsed bool
		for _, decl := range e.declRegistry.Decls {
			if decl.TypePkgName == types.PkgName(selectorBase) {
				isUsed = true
				break
			}
		}
		if !isUsed {
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

	return nil
}

const GO_MOD_NAME = "gonsole_session_src"

func (e *Executor) goModContent() ([]byte, error) {
	modName, err := e.execGoListMod()
	if err != nil {
		return nil, err
	}

	projectRootAbsPath, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("module " + GO_MOD_NAME + "\n\n")
	buf.WriteString("go " + strings.TrimPrefix(runtime.Version(), "go") + "\n\n")
	buf.WriteString("require " + strings.TrimSpace(string(modName)) + " v0.0.0\n")
	buf.WriteString("replace " + strings.TrimSpace(string(modName)) + " => " + projectRootAbsPath + "\n")
	buf.WriteString("require github.com/k0kubun/pp/v3 v3.5.1\n")

	return buf.Bytes(), nil
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
func getMainFunc(file *ast.File) *ast.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			return fn
		}
	}
	return nil
}

func clearImportPathAddedInSession() {
	importPathAddedInSession = ""
}

func loadsessionSrcFile(sessionSrcFilePath string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  "",
	}

	pkgs, err := packages.Load(cfg, sessionSrcFilePath)
	if err != nil || len(pkgs) == 0 {
		return nil, errs.NewInternalError("failed to load package").Wrap(err)
	}

	pkg := pkgs[0]

	pkg.Errors = slices.DeleteFunc(pkg.Errors, func(err packages.Error) bool {
		switch {
		case strings.Contains(err.Msg, "declared and not used"):
			return true
		case strings.Contains(err.Msg, "imported and not used"):
			return true
		case strings.Contains(err.Msg, "undefined"):
			return true
		}
		return false
	})

	if len(pkg.Errors) > 0 {
		var errMsgs []string
		for _, pkgErr := range pkg.Errors {
			errMsgs = append(errMsgs, pkgErr.Msg)
		}
		return nil, errs.NewBadInputError("failed to parse input: " + strings.Join(errMsgs, "; "))
	}
	return pkg, nil
}

func detectGoType(typInfo *gotypes.Info, declLHS ast.Expr) (gotypes.Type, error) {
	typ := typInfo.TypeOf(declLHS)
	if typ == nil {
		return nil, errs.NewInternalError("failed to detect type of variable")
	}
	return typ, nil
}

func extractDeclLHSMapFromInputStmt(inputStmt ast.Stmt) map[types.DeclName]ast.Expr {
	declLHS := make(map[types.DeclName]ast.Expr)
	switch inputStmtV := inputStmt.(type) {
	case *ast.AssignStmt:
		for _, lhs := range inputStmtV.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok {
				declLHS[types.DeclName(ident.Name)] = lhs
			}
		}
	case *ast.DeclStmt:
		switch inputDeclStmtV := inputStmtV.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range inputDeclStmtV.Specs {
				switch specV := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range specV.Names {
						declLHS[types.DeclName(name.Name)] = name
					}
				}
			}
		}
	}
	return declLHS
}

func extractInputStmtWithTypeInfo(syntax *ast.File) (ast.Stmt, error) {
	var mainFunc *ast.FuncDecl
	for _, decl := range syntax.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == "main" {
			mainFunc = funcDecl
			break
		}
	}
	if mainFunc == nil {
		return nil, errs.NewBadInputError("main function not found")
	}
	mainFuncBodyList := mainFunc.Body.List
	mainFuncBodyList = slices.DeleteFunc(mainFuncBodyList, func(stmt ast.Stmt) bool {
		assignStmt, ok := stmt.(*ast.AssignStmt)
		if !ok {
			return false
		}
		for _, stmtLHS := range assignStmt.Lhs {
			lhsIdent, ok := stmtLHS.(*ast.Ident)
			if !ok {
				continue
			}
			if lhsIdent.Name == "_" {
				return true
			}
		}
		return false
	})

	lastStmt := mainFuncBodyList[len(mainFuncBodyList)-1]
	return lastStmt, nil
}

// originCallExprの返り値の数だけpp.Printlnを呼び出す式を生成する
func composePrintClosure(originCallExpr ast.Expr, returnValuesCnt int) *ast.ExprStmt {
	// 返り値を束縛する変数名を生成
	retIdents := make([]ast.Expr, returnValuesCnt)
	for i := 0; i < returnValuesCnt; i++ {
		retIdents[i] = &ast.Ident{Name: fmt.Sprintf("ret%d", i)}
	}

	// ret0, ret1, ... := originCallExpr()の形の式を生成
	resultAssignStmt := &ast.AssignStmt{
		Lhs: retIdents,
		Tok: token.DEFINE,
		Rhs: []ast.Expr{originCallExpr},
	}

	// 各返り値をpp.Printlnで出力するための式を生成
	printStmts := make([]ast.Stmt, returnValuesCnt)
	for i := 0; i < returnValuesCnt; i++ {
		printStmts[i] = &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  ast.NewIdent("pp.Println"),
				Args: []ast.Expr{&ast.Ident{Name: fmt.Sprintf("ret%d", i)}},
			},
		}
	}

	printClosure := &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: nil},
			Results: nil,
		},
		Body: &ast.BlockStmt{
			List: append([]ast.Stmt{resultAssignStmt}, printStmts...),
		},
	}
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  printClosure,
			Args: []ast.Expr{},
		},
	}
}
