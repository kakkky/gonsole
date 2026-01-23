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

type candidates struct {
	pkgs       []types.PkgName
	funcs      map[types.PkgName][]funcSet
	methods    map[types.PkgName][]methodSet
	vars       map[types.PkgName][]varSet
	consts     map[types.PkgName][]constSet
	structs    map[types.PkgName][]structSet
	interfaces map[types.PkgName][]interfaceSet
}

type (
	funcSet struct {
		name        types.DeclName
		description string // TODO: descriptionにも型をつけたい
		returns     []returnSet
	}
	methodSet struct {
		name             types.DeclName
		description      string
		receiverTypeName types.ReceiverTypeName
		returns          []returnSet
	}
	varSet struct {
		name        types.DeclName
		description string
		typeName    types.TypeName
		pkgName     types.PkgName
	}
	constSet struct {
		name        types.DeclName
		description string
	}
	structSet struct {
		name        types.DeclName
		fields      []types.StructFieldName // 型情報を持たせてSetにしても良さそう。
		description string
	}
	// interfaceの候補を返すわけではなく、関数がinterfaceを返す場合に、
	// そのinterfaceのメソッドを候補として返すためのもの
	interfaceSet struct {
		name         types.DeclName
		methods      []types.DeclName
		descriptions []string
	}
)

type returnSet struct {
	typeName types.TypeName
	pkgName  types.PkgName
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func newCandidates(projectRootPath string) (*candidates, error) {
	nodes, err := parseProject(projectRootPath)
	if err != nil {
		return nil, err
	}
	c := candidates{
		pkgs:       make([]types.PkgName, 0),
		funcs:      make(map[types.PkgName][]funcSet),
		methods:    make(map[types.PkgName][]methodSet),
		vars:       make(map[types.PkgName][]varSet),
		consts:     make(map[types.PkgName][]constSet),
		structs:    make(map[types.PkgName][]structSet),
		interfaces: make(map[types.PkgName][]interfaceSet),
	}
	for pkgName, pkgAsts := range nodes {
		for _, pkgAst := range pkgAsts {
			c.processPackageAst(pkgName, pkgAst)
		}
	}

	return &c, nil
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func parseProject(path string) (types.GoAstNodes, error) {
	fset := token.NewFileSet()
	mode := parser.ParseComments | parser.AllErrors
	// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
	nodes := make(types.GoAstNodes)
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "vendor" {
			return filepath.SkipDir
		}
		node, err := parser.ParseDir(fset, path, nil, mode)
		for pkgName, pkg := range node {
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
	if !slices.Contains(c.pkgs, pkgName) {
		c.pkgs = append(c.pkgs, pkgName)
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
				typeName: typeNameOfReturnV,
				pkgName:  pkgNameOfReturnV,
			})
		}
	}
	c.funcs[pkgName] = append(c.funcs[pkgName], funcSet{name: types.DeclName(funcDecl.Name.Name), description: description, returns: returns})
}

func (c *candidates) processMethodDecl(pkgName types.PkgName, funcDecl *ast.FuncDecl) {
	var receiverTypeName types.ReceiverTypeName
	var returns []returnSet
	switch funcDeclRecvTypeV := funcDecl.Recv.List[0].Type.(type) {
	case *ast.Ident:
		receiverTypeName = types.ReceiverTypeName(funcDeclRecvTypeV.Name)
	case *ast.StarExpr:
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
				typeName: typeNameOfReturnV,
				pkgName:  pkgNameOfReturnV,
			})
		}
	}
	c.methods[pkgName] = append(c.methods[pkgName], methodSet{name: types.DeclName(funcDecl.Name.Name), description: description, receiverTypeName: receiverTypeName, returns: returns})
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
			switch rhsV := rhs.(type) {
			case *ast.CompositeLit:
				// 構造体リテラルの型を適切に処理
				var typeNameOfRhsV types.TypeName
				var pkgNameOfRhsV types.PkgName
				switch rhsTypeV := rhsV.Type.(type) {
				case *ast.Ident:
					// 単純な型名 (Type{})
					typeNameOfRhsV = types.TypeName(rhsTypeV.Name)
					pkgNameOfRhsV = pkgName // 現在のパッケージ名
				case *ast.SelectorExpr:
					// パッケージ名付きの型 (pkg.Type{})
					typeNameOfRhsV = types.TypeName(rhsTypeV.Sel.Name)
					pkgNameOfRhsV = types.PkgName(rhsTypeV.X.(*ast.Ident).Name)
				}
				c.vars[pkgName] = append(c.vars[pkgName], varSet{name: name, description: genDeclDescription + specDescription, typeName: typeNameOfRhsV, pkgName: pkgNameOfRhsV})
			case *ast.UnaryExpr:
				var typeNameOfRhsV types.TypeName
				var pkgNameOfRhsV types.PkgName
				switch rhsV.Op {
				case token.AND:
					switch rhsUnaryExprV := rhsV.X.(type) {
					case *ast.CompositeLit:
						switch rhsCompositeLitTypeV := rhsUnaryExprV.Type.(type) {
						case *ast.Ident:
							// 単純な型名 (Type{})
							typeNameOfRhsV = types.TypeName(rhsCompositeLitTypeV.Name)
							pkgNameOfRhsV = pkgName // 現在のパッケージ名
						case *ast.SelectorExpr:
							// パッケージ名付きの型 (pkg.Type{})
							typeNameOfRhsV = types.TypeName(rhsCompositeLitTypeV.Sel.Name)
							pkgNameOfRhsV = types.PkgName(rhsCompositeLitTypeV.X.(*ast.Ident).Name)
						}
						c.vars[pkgName] = append(c.vars[pkgName], varSet{name: name, description: genDeclDescription + specDescription, typeName: typeNameOfRhsV, pkgName: pkgNameOfRhsV})
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
							for _, varSet := range c.vars[pkgName] {
								if varSet.name == receiverVarDecl {
									receiverTypeName = types.ReceiverTypeName(varSet.typeName)
									pkgNameOfReceiver = varSet.pkgName
									break
								}
							}
							methodName := rhsFunV.Sel.Name
							if methodSets, ok := c.methods[pkgNameOfReceiver]; ok {
								for _, methodSet := range methodSets {
									if methodSet.name == types.DeclName(methodName) && methodSet.receiverTypeName == receiverTypeName {
										typeName := methodSet.returns[i].typeName
										pkgName := methodSet.returns[i].pkgName
										c.vars[pkgName] = append(c.vars[pkgName], varSet{name: name, description: genDeclDescription + specDescription, typeName: typeName, pkgName: pkgName})
										break
									}
								}
							}
							continue
						}
						funcPkgName := types.PkgName(rhsFunSelectorBaseV.Name)
						funcName := rhsFunV.Sel.Name
						if funcSets, ok := c.funcs[funcPkgName]; ok {
							for _, funcSet := range funcSets {
								if funcSet.name == types.DeclName(funcName) {
									typeName := funcSet.returns[i].typeName
									pkgName := funcSet.returns[i].pkgName
									c.vars[pkgName] = append(c.vars[pkgName], varSet{name: name, description: genDeclDescription + specDescription, typeName: typeName, pkgName: pkgName})
									break
								}
							}
						}
					}
				case *ast.Ident:
					// ローカル関数呼び出し (Func())
					funcName := rhsFunV.Name
					// 現在のパッケージから関数を探す
					if funcSets, ok := c.funcs[pkgName]; ok {
						for _, funcSet := range funcSets {
							if funcSet.name == types.DeclName(funcName) {
								typeName := funcSet.returns[i].typeName
								pkgName := funcSet.returns[i].pkgName
								c.vars[pkgName] = append(c.vars[pkgName], varSet{name: name, description: genDeclDescription + specDescription, typeName: typeName, pkgName: pkgName})
							}
						}
					}
				}
			case *ast.BasicLit:
				// 基本リテラル (文字列、数値など)
				var typeNameOfRhsV types.TypeName

				// リテラルの種類に基づいて型を推測
				switch rhsV.Kind {
				case token.INT:
					typeNameOfRhsV = "int"
				case token.FLOAT:
					typeNameOfRhsV = "float64"
				case token.IMAG:
					typeNameOfRhsV = "complex128"
				case token.CHAR:
					typeNameOfRhsV = "rune"
				case token.STRING:
					typeNameOfRhsV = "string"
				default:
					typeNameOfRhsV = "unknown"
				}

				c.vars[pkgName] = append(c.vars[pkgName], varSet{
					name:        name,
					description: genDeclDescription + specDescription,
					typeName:    typeNameOfRhsV,
					pkgName:     "",
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
			c.consts[pkgName] = append(c.consts[pkgName], constSet{name: types.DeclName(name.Name), description: genDeclDescription + specDescription})
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
			c.structs[pkgName] = append(c.structs[pkgName], structSet{name: name, fields: fields, description: genDeclDescription + specDescription})
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
			c.interfaces[pkgName] = append(c.interfaces[pkgName], interfaceSet{name: name, methods: methods, descriptions: descriptions})
		}

	}
}

func (c *candidates) isVarDecl(maybeVarDecl string) bool {
	for _, vars := range c.vars {
		for _, v := range vars {
			if string(v.name) == maybeVarDecl {
				return true
			}
		}
	}
	return false
}
