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
		name               types.DeclName
		description        string
		returnTypeNames    []string
		returnTypePkgNames []types.PkgName
	}
	methodSet struct {
		name               types.DeclName
		description        string
		receiverName       types.DeclName
		returnTypeNames    []string
		returnTypePkgNames []types.PkgName
	}
	varSet struct {
		name        types.DeclName
		description string
		typeName    string
		typePkgName types.PkgName
	}
	constSet struct {
		name        types.DeclName
		description string
	}
	structSet struct {
		name        types.DeclName
		fields      []string
		description string
	}
	// interfaceの候補を返すわけではなく、関数がinterfaceを返す場合に、
	// そのinterfaceのメソッドを候補として返すためのもの
	interfaceSet struct {
		name         types.DeclName
		methods      []string
		descriptions []string
	}
)

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
		}
	}
	for _, decl := range fileAst.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			c.processGenDecl(pkgName, d)
		}
	}
}

func isMethod(funcDecl *ast.FuncDecl) bool {
	return funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0
}

func (c *candidates) processFuncDecl(pkgName types.PkgName, funcDecl *ast.FuncDecl) {
	var description string
	var returnTypeName []string
	var returnTypePkgName []types.PkgName
	if funcDecl.Doc != nil {
		description = strings.ReplaceAll(funcDecl.Doc.Text(), "\n", "")
	}
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			var typeName string
			var typePkgName types.PkgName
			switch resultType := result.Type.(type) {
			case *ast.Ident:
				typeName = resultType.Name
				typePkgName = pkgName
			case *ast.SelectorExpr:
				typeName = resultType.Sel.Name
				typePkgName = types.PkgName(resultType.X.(*ast.Ident).Name)
			case *ast.StarExpr:
				typePkgName = pkgName
				if ident, ok := resultType.X.(*ast.Ident); ok {
					typeName = ident.Name
				}
			}
			returnTypeName = append(returnTypeName, typeName)
			returnTypePkgName = append(returnTypePkgName, typePkgName)
		}
	}
	c.funcs[pkgName] = append(c.funcs[pkgName], funcSet{name: types.DeclName(funcDecl.Name.Name), description: description, returnTypeNames: returnTypeName, returnTypePkgNames: returnTypePkgName})
}

func (c *candidates) processMethodDecl(pkgName types.PkgName, funcDecl *ast.FuncDecl) {
	var receiverName types.DeclName
	var returnTypeName []string
	var returnTypePkgName []types.PkgName
	switch receiverType := funcDecl.Recv.List[0].Type.(type) {
	case *ast.Ident:
		receiverName = types.DeclName(receiverType.Name)
	case *ast.StarExpr:
		if ident, ok := receiverType.X.(*ast.Ident); ok {
			receiverName = types.DeclName(ident.Name)
		}
	}
	var description string
	if funcDecl.Doc != nil {
		description = strings.ReplaceAll(funcDecl.Doc.Text(), "\n", "")
	}
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			var typeName string
			var typePkgName types.PkgName
			switch resultType := result.Type.(type) {
			case *ast.Ident:
				typeName = resultType.Name
				typePkgName = pkgName
			case *ast.SelectorExpr:
				typeName = resultType.Sel.Name
				typePkgName = types.PkgName(resultType.X.(*ast.Ident).Name)
			case *ast.StarExpr:
				typePkgName = pkgName
				if ident, ok := resultType.X.(*ast.Ident); ok {
					typeName = ident.Name
				}
			}
			returnTypeName = append(returnTypeName, typeName)
			returnTypePkgName = append(returnTypePkgName, typePkgName)
		}
	}
	c.methods[pkgName] = append(c.methods[pkgName], methodSet{name: types.DeclName(funcDecl.Name.Name), description: description, receiverName: receiverName, returnTypeNames: returnTypeName, returnTypePkgNames: returnTypePkgName})
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
			switch rhs := val.(type) {
			case *ast.CompositeLit:
				// 構造体リテラルの型を適切に処理
				var typeName string
				var typePkgName types.PkgName
				switch typeExpr := rhs.Type.(type) {
				case *ast.SelectorExpr:
					// パッケージ名付きの型 (pkg.Type{})
					typeName = typeExpr.Sel.Name
					typePkgName = types.PkgName(typeExpr.X.(*ast.Ident).Name)
				case *ast.Ident:
					// 単純な型名 (Type{})
					typeName = typeExpr.Name
					typePkgName = pkgName // 現在のパッケージ名
				}
				c.vars[pkgName] = append(c.vars[pkgName], varSet{name: types.DeclName(name), description: genDeclDescription + specDescription, typeName: typeName, typePkgName: typePkgName})
			case *ast.UnaryExpr:
				if rhs.Op == token.AND {
					// & 演算子の場合
					if compLit, ok := rhs.X.(*ast.CompositeLit); ok {
						// 構造体リテラルの型を適切に処理
						var typeName string
						var typePkgName types.PkgName
						switch typeExpr := compLit.Type.(type) {
						case *ast.SelectorExpr:
							// パッケージ名付きの型 (pkg.Type{})
							typeName = typeExpr.Sel.Name
							typePkgName = types.PkgName(typeExpr.X.(*ast.Ident).Name)
						case *ast.Ident:
							// 単純な型名 (Type{})
							typeName = typeExpr.Name
							typePkgName = pkgName // 現在のパッケージ名
						}
						c.vars[pkgName] = append(c.vars[pkgName], varSet{name: types.DeclName(name), description: genDeclDescription + specDescription, typeName: typeName, typePkgName: typePkgName})
					}
				}
			case *ast.CallExpr:
				// funcSetから関数の戻り値を取得
				switch funExpr := rhs.Fun.(type) {
				case *ast.SelectorExpr:
					// パッケージ名付きの関数呼び出し (pkg.Func())
					if pkgIdent, ok := funExpr.X.(*ast.Ident); ok {
						funcPkgName := types.PkgName(pkgIdent.Name)
						funcName := funExpr.Sel.Name
						if funcSets, ok := c.funcs[funcPkgName]; ok {
							for _, funcSet := range funcSets {
								if funcSet.name == types.DeclName(funcName) {
									typeName := funcSet.returnTypeNames[i]
									typePkgName := funcSet.returnTypePkgNames[i]
									c.vars[pkgName] = append(c.vars[pkgName], varSet{name: types.DeclName(name), description: genDeclDescription + specDescription, typeName: typeName, typePkgName: typePkgName})
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
								typeName := funcSet.returnTypeNames[i]
								typePkgName := funcSet.returnTypePkgNames[i]
								c.vars[pkgName] = append(c.vars[pkgName], varSet{name: types.DeclName(name), description: genDeclDescription + specDescription, typeName: typeName, typePkgName: typePkgName})
							}
						}
					}
				}
			case *ast.BasicLit:
				// 基本リテラル (文字列、数値など)
				var typeName string
				var typePkgName types.PkgName

				// リテラルの種類に基づいて型を推測
				switch rhs.Kind {
				case token.INT:
					typeName = "int"
				case token.FLOAT:
					typeName = "float64"
				case token.IMAG:
					typeName = "complex128"
				case token.CHAR:
					typeName = "rune"
				case token.STRING:
					typeName = "string"
				default:
					typeName = "unknown"
				}

				typePkgName = "" // 組み込み型なのでパッケージ名はなし
				c.vars[pkgName] = append(c.vars[pkgName], varSet{
					name:        types.DeclName(name),
					description: genDeclDescription + specDescription,
					typeName:    typeName,
					typePkgName: typePkgName,
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
			var fields []string
			for _, field := range typespecV.Fields.List {
				if len(field.Names) > 0 {
					fields = append(fields, field.Names[0].Name)
				} else {
					// 埋め込み型（匿名フィールド）
					switch t := field.Type.(type) {
					case *ast.Ident:
						fields = append(fields, t.Name)
					}
				}
			}
			var specDescription string
			if typespec.Doc != nil {
				specDescription += "   " + strings.TrimSpace(typespec.Doc.Text())
			}
			c.structs[pkgName] = append(c.structs[pkgName], structSet{name: types.DeclName(typespec.Name.Name), fields: fields, description: genDeclDescription + specDescription})
		case *ast.InterfaceType:
			var methods []string
			var descriptions []string
			for _, method := range typespecV.Methods.List {
				if len(method.Names) > 0 {
					methods = append(methods, method.Names[0].Name)
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
