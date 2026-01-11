package completer

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

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
		description string
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
		fields      []types.StructFieldName
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
func NewCandidates(nodes types.GoAstNodes) (*candidates, error) {
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
	funcDeclRecvType := funcDecl.Recv.List[0].Type
	switch funcDeclRecvTypeV := funcDeclRecvType.(type) {
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
		varspec := spec.(*ast.ValueSpec)
		var specDescription string
		if varspec.Doc != nil {
			specDescription += "   " + strings.TrimSpace(varspec.Doc.Text())
		}

		for i, val := range varspec.Values {
			name := varspec.Names[i].Name
			switch rhsV := val.(type) {
			case *ast.CompositeLit:
				// 構造体リテラルの型を適切に処理
				var typeNameOfRhsV types.TypeName
				var pkgNameOfRhsV types.PkgName
				switch typeExpr := rhsV.Type.(type) {
				case *ast.SelectorExpr:
					// パッケージ名付きの型 (pkg.Type{})
					typeNameOfRhsV = types.TypeName(typeExpr.Sel.Name)
					pkgNameOfRhsV = types.PkgName(typeExpr.X.(*ast.Ident).Name)
				case *ast.Ident:
					// 単純な型名 (Type{})
					typeNameOfRhsV = types.TypeName(typeExpr.Name)
					pkgNameOfRhsV = pkgName // 現在のパッケージ名
				}
				c.vars[pkgName] = append(c.vars[pkgName], varSet{name: types.DeclName(name), description: genDeclDescription + specDescription, typeName: typeNameOfRhsV, pkgName: pkgNameOfRhsV})
			case *ast.UnaryExpr:
				if rhsV.Op == token.AND {
					// & 演算子の場合
					if compLit, ok := rhsV.X.(*ast.CompositeLit); ok {
						// 構造体リテラルの型を適切に処理
						var typeNameOfRhsV types.TypeName
						var pkgNameOfRhsV types.PkgName
						switch typeExpr := compLit.Type.(type) {
						case *ast.SelectorExpr:
							// パッケージ名付きの型 (pkg.Type{})
							typeNameOfRhsV = types.TypeName(typeExpr.Sel.Name)
							pkgNameOfRhsV = types.PkgName(typeExpr.X.(*ast.Ident).Name)
						case *ast.Ident:
							// 単純な型名 (Type{})
							typeNameOfRhsV = types.TypeName(typeExpr.Name)
							pkgNameOfRhsV = pkgName // 現在のパッケージ名
						}
						c.vars[pkgName] = append(c.vars[pkgName], varSet{name: types.DeclName(name), description: genDeclDescription + specDescription, typeName: typeNameOfRhsV, pkgName: pkgNameOfRhsV})
					}
				}
			case *ast.CallExpr:
				// funcSetから関数の戻り値を取得
				switch funExpr := rhsV.Fun.(type) {
				case *ast.SelectorExpr:
					// パッケージ名付きの関数呼び出し (pkg.Func())
					if pkgIdent, ok := funExpr.X.(*ast.Ident); ok {
						funcPkgName := types.PkgName(pkgIdent.Name)
						funcName := funExpr.Sel.Name
						if funcSets, ok := c.funcs[funcPkgName]; ok {
							for _, funcSet := range funcSets {
								if funcSet.name == types.DeclName(funcName) {
									typeName := funcSet.returns[i].typeName
									pkgName := funcSet.returns[i].pkgName
									c.vars[pkgName] = append(c.vars[pkgName], varSet{name: types.DeclName(name), description: genDeclDescription + specDescription, typeName: typeName, pkgName: pkgName})
								}
							}
						}
					}
				case *ast.Ident:
					// ローカル関数呼び出し (Func())
					funcName := funExpr.Name
					// 現在のパッケージから関数を探す
					if funcSets, ok := c.funcs[pkgName]; ok {
						for _, funcSet := range funcSets {
							if funcSet.name == types.DeclName(funcName) {
								typeName := funcSet.returns[i].typeName
								pkgName := funcSet.returns[i].pkgName
								c.vars[pkgName] = append(c.vars[pkgName], varSet{name: types.DeclName(name), description: genDeclDescription + specDescription, typeName: typeName, pkgName: pkgName})
							}
						}
					}
				}
			case *ast.BasicLit:
				// 基本リテラル (文字列、数値など)
				var typeNameOfRhsV types.TypeName
				var typePkgNameOfRhsV types.PkgName

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

				typePkgNameOfRhsV = "" // 組み込み型なのでパッケージ名はなし
				c.vars[pkgName] = append(c.vars[pkgName], varSet{
					name:        types.DeclName(name),
					description: genDeclDescription + specDescription,
					typeName:    typeNameOfRhsV,
					pkgName:     typePkgNameOfRhsV,
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
		constspec := spec.(*ast.ValueSpec)
		if constspec.Doc != nil {
			specDescription += "   " + strings.TrimSpace(constspec.Doc.Text())
		}
		for _, constname := range constspec.Names {
			c.consts[pkgName] = append(c.consts[pkgName], constSet{name: types.DeclName(constname.Name), description: genDeclDescription + specDescription})
		}
	}
}
func (c *candidates) processTypeDecl(pkgName types.PkgName, genDecl *ast.GenDecl) {
	var genDeclDescription string
	if genDecl.Doc != nil {
		genDeclDescription += strings.TrimSpace(genDecl.Doc.Text())
	}
	for _, spec := range genDecl.Specs {
		typespec := spec.(*ast.TypeSpec)
		switch typespecV := typespec.Type.(type) {
		case *ast.StructType:
			var fields []types.StructFieldName
			for _, field := range typespecV.Fields.List {
				if len(field.Names) > 0 {
					fields = append(fields, types.StructFieldName(field.Names[0].Name))
				} else {
					// 埋め込み型（匿名フィールド）
					switch t := field.Type.(type) {
					case *ast.Ident:
						fields = append(fields, types.StructFieldName(t.Name))
					}
				}
			}
			var specDescription string
			if typespec.Doc != nil {
				specDescription += "   " + strings.TrimSpace(typespec.Doc.Text())
			}
			c.structs[pkgName] = append(c.structs[pkgName], structSet{name: types.DeclName(typespec.Name.Name), fields: fields, description: genDeclDescription + specDescription})
		case *ast.InterfaceType:
			var methods []types.DeclName
			var descriptions []string
			for _, method := range typespecV.Methods.List {
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
			c.interfaces[pkgName] = append(c.interfaces[pkgName], interfaceSet{name: types.DeclName(typespec.Name.Name), methods: methods, descriptions: descriptions})
		}

	}
}
