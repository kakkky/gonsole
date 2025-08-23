package executor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"unicode"

	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/utils"
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

	var isPrivate bool
	var inputWithPrivateIdent string
	if isIncludePrivateIdent(input) {
		isPrivate = true
		inputWithPrivateIdent = input

		wrappedWithPublicFunc := wrapWithPublicFunc(input)

		// tmpFileにはラッパー関数を追加するように代入
		input = wrappedWithPublicFunc
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
				case *ast.CallExpr:
					// 式からパッケージ名を抽出
					pkgNameToImport, found := extractPkgNameFromExpr(exprV)
					if !found {
						return errs.NewInternalError("failed to extract package name from expression")
					}
					// パッケージのインポート文を追加
					importPath, err := e.addImportDecl(tmpFileAst, pkgNameToImport)
					if err != nil {
						return err
					}
					if isPrivate {
						if err := e.defineWrappedPublicFunc(inputWithPrivateIdent, input, importPath, pkgNameToImport); err != nil {
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
						_, err := e.addImportDecl(tmpFileAst, "fmt")
						if err != nil {
							return err
						}

					} else {
						// 関数が返り値を持たない場合はそのまま使用
						exprStmt = &ast.ExprStmt{X: exprV}
					}
					addInputStmt(exprStmt, mainFuncBody)

				// 変数だった場合(repl上で定義した変数単体)
				case *ast.SelectorExpr:
					pkgNameToImport, found := extractPkgNameFromExpr(exprV)
					if !found {
						return errs.NewInternalError("failed to extract package name from expression")
					}
					// パッケージのインポート文を追加
					importPath, err := e.addImportDecl(tmpFileAst, pkgNameToImport)
					if err != nil {
						return err
					}
					wrappedExpr := wrapWithPrintln(exprV)
					addInputStmt(wrappedExpr, mainFuncBody)
					_, err = e.addImportDecl(tmpFileAst, "fmt")
					if err != nil {
						return err
					}
					if isPrivate {
						e.defineWrappedPublicFunc(inputWithPrivateIdent, input, importPath, pkgNameToImport)
					}
				case *ast.Ident:
					wrappedExpr := wrapWithPrintln(exprV)
					addInputStmt(wrappedExpr, mainFuncBody)
					_, err := e.addImportDecl(tmpFileAst, "fmt")
					if err != nil {
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
					importPath, err := e.addImportDecl(tmpFileAst, pkgNameToImport)
					if err != nil {
						return err
					}
					if isPrivate {
						e.defineWrappedPublicFunc(inputWithPrivateIdent, input, importPath, pkgNameToImport)
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
								importPath, err := e.addImportDecl(tmpFileAst, pkgNameToImport)
								if err != nil {
									return err
								}
								if isPrivate {
									e.defineWrappedPublicFunc(inputWithPrivateIdent, input, importPath, pkgNameToImport)
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
			switch x := funExprV.X.(type) {
			case *ast.Ident:
				return x.Name, true
			default:
				// メソッドチェーン対応
				return extractPkgNameFromExpr(funExprV.X)
			}
		}
	}
	return "", false
}

func (e *Executor) addImportDecl(fileAst *ast.File, pkgNameToImport string) (string, error) {
	// repl内で定義された変数エントリにある場合は無視
	// 理由：パッケージ名ではなく、メソッド呼び出しに対するレシーバー名として使用されていると予測できるため
	if e.declEntry.IsRegisteredDecl(pkgNameToImport) {
		return "", nil
	}

	// パッケージパスを探索
	importPath, err := e.resolveImportPathForAdd(pkgNameToImport)
	if err != nil {
		return importPath, err
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
						return importPath, nil // すでにインポート済みとして何もしない
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
	return importPath, nil
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
func (e *Executor) isFuncVoid(pkgName, funcName string) (bool, error) {
	if e.declEntry.IsRegisteredDecl(pkgName) {
		pkgName = e.declEntry.ReceiverTypePkgName(pkgName)
	}
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

// 流れを整理
// privateな構造体、変数、関数、メソッドへのアクセス（呼び出し)だった場合、
// wrappedSrcに入れるのはこちら側でパブリックにラップした関数の呼び出し。
//
// そして、実行する前にラッパーした関数の定義をそのパッケージのあるディレクトリにtmpfileとして保存する
//  関数であれば、その関数の返り値を返り値に取るパブリックラッパー関数
//  メソッドであれば、同じレシーバをとり、返り値もそろえたパブリックラッパー関数
//  変数、構造体であれば、その変数(の型)、構造体を返り値に取るパブリックラッパー関数

func isIncludePrivateIdent(input string) bool {
	// = があればそれ以降を取得
	if strings.Contains(input, "=") {
		input = strings.Split(input, "=")[1]
	}
	// . の後を取得
	ident := strings.SplitN(input, ".", 2)[1]
	return unicode.IsLower(rune(ident[0]))
}

func wrapWithPublicFunc(input string) string {
	// = があればそれ以降を取得
	if strings.Contains(input, "=") {
		input = strings.Split(input, "=")[1]
	}
	// . の後を取得
	ident := strings.SplitN(input, ".", 2)[1]
	// ( か { があればその前までを取得
	if idx := strings.IndexAny(ident, "{("); idx != -1 {
		return strings.ReplaceAll(input, ident, "Wrapped"+strings.Title(ident[:idx]+"()"))
	}
	return strings.ReplaceAll(input, ident, "Wrapped"+strings.Title(ident)+"()")
}

func (e *Executor) defineWrappedPublicFunc(inputWithPrivateIdent, wrappedPublicFunc, importPath, pkgName string) error {
	pimf := e.privateIdentManageInfoMap[importPath]
	if pimf == nil {
		relativeDir := strings.TrimPrefix(importPath, e.modPath)
		createdTmpFilePath, cleaner, err := makeTmpFile("." + relativeDir)
		if err != nil {
			return err
		}
		pimf = &privateIdentManageInfo{
			tmpFilePath: createdTmpFilePath,
			cleaner:     cleaner,
		}
		e.privateIdentManageInfoMap[importPath] = pimf
	}
	// tmpファイル内容を直接読み込む
	tmpContent, err := os.ReadFile(pimf.tmpFilePath)
	if err != nil {
		return errs.NewInternalError("failed to read temporary file").Wrap(err)
	}
	fset := token.NewFileSet()
	if strings.Contains(string(tmpContent), strings.ReplaceAll(wrappedPublicFunc, pkgName+".", "")) {
		// すでに定義されている場合は何もしない
		return nil
	}
	tmpFileAst, err := parser.ParseFile(fset, e.tmpFilePath, string(tmpContent), parser.AllErrors)
	if err != nil {
		// EOFなら無視して新たなFileAst作成
		if !strings.Contains(err.Error(), "expected ';', found 'EOF'") {
			return errs.NewInternalError("failed to parse temporary file").Wrap(err)
		}
		tmpFileAst = &ast.File{
			Name:  ast.NewIdent(pkgName),
			Decls: []ast.Decl{},
		}
	}

	// 入力値をmain関数でラップしてparseする
	wrappedSrc := "package main\nfunc main() {\n" + inputWithPrivateIdent + "\n}"
	wrappedInputAst, err := parser.ParseFile(fset, "", wrappedSrc, parser.AllErrors)
	if err != nil {
		return errs.NewInternalError("failed to parse input source").Wrap(err)
	}
	// 入力文をASTとして取得
	var inputStmtWithPrivateIdent ast.Stmt
	for _, decl := range wrappedInputAst.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == "main" {
			inputStmtWithPrivateIdent = funcDecl.Body.List[0]
		}
	}

	switch stmt := inputStmtWithPrivateIdent.(type) {
	// 式だった場合は、それを評価して値を返すためにfmt.Printlnでラップする
	case *ast.ExprStmt:
		switch exprV := stmt.X.(type) {
		// 関数呼び出しの場合
		case *ast.CallExpr:
			funcOrMethodName := exprV.Fun.(*ast.SelectorExpr).Sel.Name
			if e.declEntry.IsRegisteredDecl(pkgName) {
				pkgName = e.declEntry.ReceiverTypePkgName(pkgName)
			}
			for _, pkgAst := range e.astCache.nodes[pkgName] {
				for _, fileAst := range pkgAst.Files {
					for _, decls := range fileAst.Decls {
						switch declV := decls.(type) {
						case *ast.FuncDecl:
							if declV.Name.Name == funcOrMethodName {
								var recv *ast.FieldList
								if declV.Recv != nil {
									receiverName := declV.Recv.List[0].Names[0].Name
									receiverType := declV.Recv.List[0].Type
									recv = &ast.FieldList{
										List: []*ast.Field{
											{Names: []*ast.Ident{{Name: receiverName}}, Type: receiverType},
										},
									}
								}
								returnTypes := declV.Type.Results

								// exprVから識別子削除
								exprV.Fun = &ast.Ident{Name: funcOrMethodName}

								var bodyStmt ast.Stmt
								if returnTypes != nil && len(returnTypes.List) > 0 {
									// 戻り値がある場合はreturnでラップ
									bodyStmt = &ast.ReturnStmt{Results: []ast.Expr{exprV}}
								} else {
									// 戻り値がない場合は式文としてそのまま
									bodyStmt = &ast.ExprStmt{X: exprV}

								}

								// tmpFileにラッパー関数を追加
								name := strings.ReplaceAll(wrappedPublicFunc, pkgName+".", "")
								name = strings.ReplaceAll(name, "()", "")
								tmpFileAst.Decls = append(tmpFileAst.Decls, &ast.FuncDecl{
									Name: ast.NewIdent(name),
									Recv: recv,
									Type: &ast.FuncType{
										Results: returnTypes,
									},
									Body: &ast.BlockStmt{
										List: []ast.Stmt{
											bodyStmt,
										},
									},
								})
							}
						}
					}
				}

			}
		case *ast.SelectorExpr:

		}
	// 短縮変数宣言だった場合
	case *ast.AssignStmt:
		switch stmt.Rhs[0].(type) {
		case *ast.BasicLit:
		// 基本リテラルの場合はインポート文を追加しない
		default:

		}
	// 宣言の場合
	case *ast.DeclStmt:
		// switch decl := stmt.Decl.(type) {
		// case *ast.GenDecl:
		// 	for _, spec := range decl.Specs {
		// 		switch specV := spec.(type) {
		// 		case *ast.ValueSpec:
		// 			switch valueExprV := specV.Values[0].(type) {
		// 			case *ast.BasicLit:
		// 			// 基本リテラルの場合はimport文を追加しない
		// 			default:

		// 			}

		// 		}
		// 	}
		// }
	}
	if err := outputToFile(pimf.tmpFilePath, tmpFileAst); err != nil {
		return err
	}
	// ASTキャッシュを更新
	nodes, fset, err := utils.AnalyzeGoAst(".")
	if err != nil {
		return err
	}
	e.astCache = &astCache{
		nodes: nodes,
		fset:  fset,
	}

	return nil
}
