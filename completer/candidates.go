package completer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
)

var BuildStdPkgCandidatesMode bool

type candidates struct {
	Pkgs       []types.PkgName
	Funcs      map[types.PkgName][]funcSet
	Methods    map[types.PkgName][]methodSet
	Vars       map[types.PkgName][]varSet
	Consts     map[types.PkgName][]constSet
	Structs    map[types.PkgName][]structSet
	Interfaces map[types.PkgName][]interfaceSet
}

type (
	funcSet struct {
		Name        types.DeclName
		Description string // TODO: descriptionにも型をつけたい
		Returns     []returnSet
	}
	methodSet struct {
		Name             types.DeclName
		Description      string
		ReceiverTypeName types.ReceiverTypeName
		Returns          []returnSet
	}
	varSet struct {
		Name        types.DeclName
		Description string
		TypeName    types.TypeName
		PkgName     types.PkgName
	}
	constSet struct {
		Name        types.DeclName
		Description string
	}
	structSet struct {
		Name        types.DeclName
		Fields      []types.StructFieldName // 型情報を持たせてSetにしても良さそう。
		Description string
	}
	// interfaceの候補を返すわけではなく、関数がinterfaceを返す場合に、
	// そのinterfaceのメソッドを候補として返すためのもの
	interfaceSet struct {
		Name         types.DeclName
		Methods      []types.DeclName
		Descriptions []string
	}
)

type returnSet struct {
	TypeName types.TypeName
	PkgName  types.PkgName
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func NewCandidates(projectRootPath string) (*candidates, error) {
	nodes, err := parseProject(projectRootPath)
	if err != nil {
		return nil, err
	}
	c := candidates{
		Pkgs:       make([]types.PkgName, 0),
		Funcs:      make(map[types.PkgName][]funcSet),
		Methods:    make(map[types.PkgName][]methodSet),
		Vars:       make(map[types.PkgName][]varSet),
		Consts:     make(map[types.PkgName][]constSet),
		Structs:    make(map[types.PkgName][]structSet),
		Interfaces: make(map[types.PkgName][]interfaceSet),
	}
	for pkgName, pkgAsts := range nodes {
		for _, pkgAst := range pkgAsts {
			c.processPackageAst(pkgName, pkgAst)
		}
	}

	// 標準ライブラリの候補とマージ
	c.mergeCandidates(stdPkgCandidates)

	return &c, nil
}

// mergeCandidates は他のcandidatesをマージする
func (c *candidates) mergeCandidates(other *candidates) {
	// パッケージ名をマージ
	for _, pkg := range other.Pkgs {
		if !slices.Contains(c.Pkgs, pkg) {
			c.Pkgs = append(c.Pkgs, pkg)
		}
	}

	// 関数をマージ
	for pkgName, funcs := range other.Funcs {
		c.Funcs[pkgName] = append(c.Funcs[pkgName], funcs...)
	}

	// メソッドをマージ
	for pkgName, methods := range other.Methods {
		c.Methods[pkgName] = append(c.Methods[pkgName], methods...)
	}

	// 変数をマージ
	for pkgName, vars := range other.Vars {
		c.Vars[pkgName] = append(c.Vars[pkgName], vars...)
	}

	// 定数をマージ
	for pkgName, consts := range other.Consts {
		c.Consts[pkgName] = append(c.Consts[pkgName], consts...)
	}

	// 構造体をマージ
	for pkgName, structs := range other.Structs {
		c.Structs[pkgName] = append(c.Structs[pkgName], structs...)
	}

	// インターフェースをマージ
	for pkgName, interfaces := range other.Interfaces {
		c.Interfaces[pkgName] = append(c.Interfaces[pkgName], interfaces...)
	}
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func parseProject(path string) (types.GoAstNodes, error) {
	fset := token.NewFileSet()
	mode := parser.ParseComments | parser.AllErrors
	// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
	nodes := make(types.GoAstNodes)
	skipBasePaths := []string{"vendor", "internal", "testdata", "cmd", "runtime", "benchmark"}
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if slices.Contains(skipBasePaths, filepath.Base(path)) {
			return filepath.SkipDir
		}
		// _test.goファイルを除外するフィルタ
		filter := func(info fs.FileInfo) bool {
			return !strings.HasSuffix(info.Name(), "_test.go")
		}
		node, err := parser.ParseDir(fset, path, filter, mode)
		for pkgName, pkg := range node {
			// _testパッケージは除外
			if strings.HasSuffix(string(pkgName), "_test") || pkgName == "main" {
				continue
			}
			nodes[types.PkgName(pkgName)] = append(nodes[types.PkgName(pkgName)], pkg)
		}
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, errs.NewInternalError("failed to walk directory").Wrap(err)
	}
	return nodes, nil
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func (c *candidates) processPackageAst(pkgName types.PkgName, pkgAst *ast.Package) {
	if !slices.Contains(c.Pkgs, pkgName) {
		c.Pkgs = append(c.Pkgs, pkgName)
	}
	for _, fileAst := range pkgAst.Files {
		c.processFileAst(pkgName, fileAst)
	}
}

func (c *candidates) processFileAst(pkgName types.PkgName, fileAst *ast.File) {
	for _, decl := range fileAst.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if isMethod(d) {
				c.processMethodDecl(pkgName, d)
				continue
			}
			c.processFuncDecl(pkgName, d)
		case *ast.GenDecl:
			c.processGenDecl(pkgName, d)
		}
	}
}

func isMethod(funcDecl *ast.FuncDecl) bool {
	return funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0
}

// TODO: paramsも見て、補完候補のdescription部分に追加したさはある
func (c *candidates) processFuncDecl(pkgName types.PkgName, funcDecl *ast.FuncDecl) {

	if BuildStdPkgCandidatesMode && isPrivate(funcDecl.Name.Name) {
		return
	}
	// 空の名前やアンダースコアのみの名前を除外
	if funcDecl.Name.Name == "" || funcDecl.Name.Name == "_" {
		return
	}

	var description string
	var returns []returnSet
	if funcDecl.Doc != nil {
		description = strings.ReplaceAll(funcDecl.Doc.Text(), "\n", "")
	}
	funcDeclReturns := funcDecl.Type.Results
	if funcDeclReturns != nil {
		for _, returnElm := range funcDeclReturns.List {
			var typeNameOfReturnV types.TypeName
			var pkgNameOfReturnV types.PkgName
			switch returnV := returnElm.Type.(type) {
			case *ast.Ident:
				typeNameOfReturnV = types.TypeName(returnV.Name)
				pkgNameOfReturnV = pkgName
			case *ast.SelectorExpr:
				typeNameOfReturnV = types.TypeName(returnV.Sel.Name)
				pkgNameOfReturnV = types.PkgName(returnV.X.(*ast.Ident).Name)
			case *ast.StarExpr:
				switch returnV := returnV.X.(type) {
				case *ast.Ident:
					typeNameOfReturnV = types.TypeName(returnV.Name)
					pkgNameOfReturnV = pkgName
				case *ast.SelectorExpr:
					typeNameOfReturnV = types.TypeName(returnV.Sel.Name)
					pkgNameOfReturnV = types.PkgName(returnV.X.(*ast.Ident).Name)
				}
			}

			returns = append(returns, returnSet{
				TypeName: typeNameOfReturnV,
				PkgName:  pkgNameOfReturnV,
			})
		}
	}
	c.Funcs[pkgName] = append(c.Funcs[pkgName], funcSet{Name: types.DeclName(funcDecl.Name.Name), Description: description, Returns: returns})
}

func (c *candidates) processMethodDecl(pkgName types.PkgName, funcDecl *ast.FuncDecl) {
	if BuildStdPkgCandidatesMode && isPrivate(funcDecl.Name.Name) {
		return
	}

	var receiverTypeName types.ReceiverTypeName
	var returns []returnSet
	switch funcDeclRecvTypeV := funcDecl.Recv.List[0].Type.(type) {
	case *ast.Ident:
		receiverTypeName = types.ReceiverTypeName(funcDeclRecvTypeV.Name)
	case *ast.StarExpr:
		if _, ok := funcDeclRecvTypeV.X.(*ast.Ident); !ok {
			// ジェネリクス等だった場合は一旦スキップ
			return
		}
		receiverTypeName = types.ReceiverTypeName(funcDeclRecvTypeV.X.(*ast.Ident).Name)
	}
	var description string
	if funcDecl.Doc != nil {
		description = strings.ReplaceAll(funcDecl.Doc.Text(), "\n", "")
	}
	if funcDecl.Type.Results != nil {
		for _, returnElm := range funcDecl.Type.Results.List {
			var typeNameOfReturnV types.TypeName
			var pkgNameOfReturnV types.PkgName
			switch returnV := returnElm.Type.(type) {
			case *ast.Ident:
				typeNameOfReturnV = types.TypeName(returnV.Name)
				pkgNameOfReturnV = pkgName
			case *ast.SelectorExpr:
				typeNameOfReturnV = types.TypeName(returnV.Sel.Name)
				pkgNameOfReturnV = types.PkgName(returnV.X.(*ast.Ident).Name)
			case *ast.StarExpr:
				switch returnV := returnV.X.(type) {
				case *ast.Ident:
					typeNameOfReturnV = types.TypeName(returnV.Name)
					pkgNameOfReturnV = pkgName
				case *ast.SelectorExpr:
					typeNameOfReturnV = types.TypeName(returnV.Sel.Name)
					pkgNameOfReturnV = types.PkgName(returnV.X.(*ast.Ident).Name)
				}
			}
			returns = append(returns, returnSet{
				TypeName: typeNameOfReturnV,
				PkgName:  pkgNameOfReturnV,
			})
		}
	}
	c.Methods[pkgName] = append(c.Methods[pkgName], methodSet{Name: types.DeclName(funcDecl.Name.Name), Description: description, ReceiverTypeName: receiverTypeName, Returns: returns})
}

func (c *candidates) processGenDecl(pkgName types.PkgName, genDecl *ast.GenDecl) {
	switch genDecl.Tok {
	case token.VAR:
		c.processVarDecl(pkgName, genDecl)
	case token.CONST:
		c.processConstDecl(pkgName, genDecl)
	case token.TYPE:
		c.processTypeDecl(pkgName, genDecl)
	}
}

func (c *candidates) processVarDecl(pkgName types.PkgName, genDecl *ast.GenDecl) {
	var genDeclDescription string
	if genDecl.Doc != nil {
		genDeclDescription += strings.TrimSpace(genDecl.Doc.Text())
	}
	for _, spec := range genDecl.Specs {
		specV := spec.(*ast.ValueSpec)
		var specDescription string
		if specV.Doc != nil {
			specDescription += "   " + strings.TrimSpace(specV.Doc.Text())
		}

		for i, rhs := range specV.Values {
			name := types.DeclName(specV.Names[i].Name)

			if BuildStdPkgCandidatesMode && isPrivate(string(name)) {
				continue
			}

			switch rhsV := rhs.(type) {
			case *ast.CompositeLit:
				// 構造体リテラルの型を適切に処理
				var typeNameOfRHSV types.TypeName
				var pkgNameOfRHSV types.PkgName
				switch rhsTypeV := rhsV.Type.(type) {
				case *ast.Ident:
					// 単純な型名 (Type{})
					typeNameOfRHSV = types.TypeName(rhsTypeV.Name)
					pkgNameOfRHSV = pkgName // 現在のパッケージ名
				case *ast.SelectorExpr:
					// パッケージ名付きの型 (pkg.Type{})
					typeNameOfRHSV = types.TypeName(rhsTypeV.Sel.Name)
					pkgNameOfRHSV = types.PkgName(rhsTypeV.X.(*ast.Ident).Name)
				}
				c.Vars[pkgName] = append(c.Vars[pkgName], varSet{Name: name, Description: genDeclDescription + specDescription, TypeName: typeNameOfRHSV, PkgName: pkgNameOfRHSV})
			case *ast.UnaryExpr:
				var typeNameOfRHSV types.TypeName
				var pkgNameOfRHSV types.PkgName
				switch rhsV.Op {
				case token.AND:
					switch rhsUnaryExprV := rhsV.X.(type) {
					case *ast.CompositeLit:
						switch rhsCompositeLitTypeV := rhsUnaryExprV.Type.(type) {
						case *ast.Ident:
							// 単純な型名 (Type{})
							typeNameOfRHSV = types.TypeName(rhsCompositeLitTypeV.Name)
							pkgNameOfRHSV = pkgName // 現在のパッケージ名
						case *ast.SelectorExpr:
							// パッケージ名付きの型 (pkg.Type{})
							typeNameOfRHSV = types.TypeName(rhsCompositeLitTypeV.Sel.Name)
							pkgNameOfRHSV = types.PkgName(rhsCompositeLitTypeV.X.(*ast.Ident).Name)
						}
						c.Vars[pkgName] = append(c.Vars[pkgName], varSet{Name: name, Description: genDeclDescription + specDescription, TypeName: typeNameOfRHSV, PkgName: pkgNameOfRHSV})
					}
				case token.MUL:
					// TODO: デリファレンスの場合の処理
					// デリファレンスしている先の変数の型情報とかを取る必要がありそう。

				}
			case *ast.CallExpr:
				// funcSetから関数の戻り値を取得
				switch rhsFunV := rhsV.Fun.(type) {
				case *ast.SelectorExpr:
					switch rhsFunSelectorBaseV := rhsFunV.X.(type) {
					case *ast.Ident:
						selectorBase := rhsFunSelectorBaseV.Name
						if c.isVarDecl(selectorBase) {
							// トップレベルで宣言された変数がレシーバーとなっているメソッド呼び出し
							receiverVarDecl := types.DeclName(selectorBase)
							var receiverTypeName types.ReceiverTypeName
							var pkgNameOfReceiver types.PkgName
							for _, varSet := range c.Vars[pkgName] {
								if varSet.Name == receiverVarDecl {
									receiverTypeName = types.ReceiverTypeName(varSet.TypeName)
									pkgNameOfReceiver = varSet.PkgName
									break
								}
							}
							methodName := rhsFunV.Sel.Name
							if methodSets, ok := c.Methods[pkgNameOfReceiver]; ok {
								for _, methodSet := range methodSets {
									if methodSet.Name == types.DeclName(methodName) && methodSet.ReceiverTypeName == receiverTypeName {
										typeName := methodSet.Returns[i].TypeName
										pkgName := methodSet.Returns[i].PkgName
										c.Vars[pkgName] = append(c.Vars[pkgName], varSet{Name: name, Description: genDeclDescription + specDescription, TypeName: typeName, PkgName: pkgName})
										break
									}
								}
							}
							continue
						}
						funcPkgName := types.PkgName(rhsFunSelectorBaseV.Name)
						funcName := rhsFunV.Sel.Name
						if funcSets, ok := c.Funcs[funcPkgName]; ok {
							for _, funcSet := range funcSets {
								if funcSet.Name == types.DeclName(funcName) {
									typeName := funcSet.Returns[i].TypeName
									pkgName := funcSet.Returns[i].PkgName
									c.Vars[pkgName] = append(c.Vars[pkgName], varSet{Name: name, Description: genDeclDescription + specDescription, TypeName: typeName, PkgName: pkgName})
									break
								}
							}
						}
					}
				case *ast.Ident:
					// ローカル関数呼び出し (Func())
					funcName := rhsFunV.Name
					// 現在のパッケージから関数を探す
					if funcSets, ok := c.Funcs[pkgName]; ok {
						for _, funcSet := range funcSets {
							if funcSet.Name == types.DeclName(funcName) {
								typeName := funcSet.Returns[i].TypeName
								pkgName := funcSet.Returns[i].PkgName
								c.Vars[pkgName] = append(c.Vars[pkgName], varSet{Name: name, Description: genDeclDescription + specDescription, TypeName: typeName, PkgName: pkgName})
							}
						}
					}
				}
			case *ast.BasicLit:
				// 基本リテラル (文字列、数値など)
				var typeNameOfRHSV types.TypeName

				// リテラルの種類に基づいて型を推測
				switch rhsV.Kind {
				case token.INT:
					typeNameOfRHSV = "int"
				case token.FLOAT:
					typeNameOfRHSV = "float64"
				case token.IMAG:
					typeNameOfRHSV = "complex128"
				case token.CHAR:
					typeNameOfRHSV = "rune"
				case token.STRING:
					typeNameOfRHSV = "string"
				default:
					typeNameOfRHSV = "unknown"
				}

				c.Vars[pkgName] = append(c.Vars[pkgName], varSet{
					Name:        name,
					Description: genDeclDescription + specDescription,
					TypeName:    typeNameOfRHSV,
					PkgName:     "",
				})
			}
		}
	}
}

func (c *candidates) processConstDecl(pkgName types.PkgName, genDecl *ast.GenDecl) {
	var genDeclDescription string
	if genDecl.Doc != nil {
		genDeclDescription += strings.TrimSpace(genDecl.Doc.Text())
	}
	for _, spec := range genDecl.Specs {
		var specDescription string
		specV := spec.(*ast.ValueSpec)
		if specV.Doc != nil {
			specDescription += "   " + strings.TrimSpace(specV.Doc.Text())
		}
		// １つのValueSpecに複数の定数名がある場合に対応
		// 例: const A, B = 1, 2
		for _, name := range specV.Names {
			if BuildStdPkgCandidatesMode && isPrivate(name.Name) {
				continue
			}
			// 空の名前やアンダースコアのみの名前を除外
			if name.Name == "" || name.Name == "_" {
				continue
			}
			c.Consts[pkgName] = append(c.Consts[pkgName], constSet{Name: types.DeclName(name.Name), Description: genDeclDescription + specDescription})
		}
	}
}
func (c *candidates) processTypeDecl(pkgName types.PkgName, genDecl *ast.GenDecl) {
	var genDeclDescription string
	if genDecl.Doc != nil {
		genDeclDescription += strings.TrimSpace(genDecl.Doc.Text())
	}
	for _, spec := range genDecl.Specs {
		specV := spec.(*ast.TypeSpec)
		name := types.DeclName(specV.Name.Name)
		if BuildStdPkgCandidatesMode && isPrivate(specV.Name.Name) {
			continue
		}
		// 空の名前やアンダースコアのみの名前を除外
		if specV.Name.Name == "" || specV.Name.Name == "_" {
			continue
		}
		switch specTypeV := specV.Type.(type) {
		case *ast.StructType:
			var specDescription string
			if specV.Doc != nil {
				specDescription += "   " + strings.TrimSpace(specV.Doc.Text())
			}
			var fields []types.StructFieldName
			for _, field := range specTypeV.Fields.List {
				if len(field.Names) == 0 {
					// 埋め込み型（匿名フィールド）
					switch t := field.Type.(type) {
					case *ast.Ident:
						fields = append(fields, types.StructFieldName(t.Name))
					}
					continue
				}
				for _, fieldName := range field.Names {
					fields = append(fields, types.StructFieldName(fieldName.Name))
				}
			}
			c.Structs[pkgName] = append(c.Structs[pkgName], structSet{Name: name, Fields: fields, Description: genDeclDescription + specDescription})
		case *ast.InterfaceType:
			var methods []types.DeclName
			var descriptions []string
			for _, method := range specTypeV.Methods.List {
				if len(method.Names) > 0 {
					methods = append(methods, types.DeclName(method.Names[0].Name))
				}
				commentBuilder := strings.Builder{}
				if method.Doc != nil {
					commentBuilder.WriteString(strings.ReplaceAll(method.Doc.Text(), "\n", "") + "")
				}
				if method.Comment != nil {
					commentBuilder.WriteString(strings.ReplaceAll(method.Comment.Text(), "\n", ""))
				}
				descriptions = append(descriptions, commentBuilder.String())
			}
			c.Interfaces[pkgName] = append(c.Interfaces[pkgName], interfaceSet{Name: name, Methods: methods, Descriptions: descriptions})
		}

	}
}

func (c *candidates) isVarDecl(maybeVarDecl string) bool {
	for _, vars := range c.Vars {
		for _, v := range vars {
			if string(v.Name) == maybeVarDecl {
				return true
			}
		}
	}
	return false
}
