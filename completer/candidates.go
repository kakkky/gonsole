package completer

import (
	"go/ast"
	"go/token"
	"strings"
)

type pkgName string

type (
	funcSet struct {
		name              string
		description       string
		returnTypeName    []string
		returnTypePkgName []string
	}
	methodSet struct {
		name              string
		description       string
		receiverTypeName  string
		returnTypeName    []string
		returnTypePkgName []string
	}
	varSet struct {
		name        string
		description string
		typeName    string
		typePkgName string
	}
	constSet struct {
		name        string
		description string
	}
	structSet struct {
		name        string
		fields      []string
		description string
	}
)

type candidates struct {
	pkgs    []pkgName
	funcs   map[pkgName][]funcSet
	methods map[pkgName][]methodSet
	vars    map[pkgName][]varSet
	consts  map[pkgName][]constSet
	structs map[pkgName][]structSet
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func NewCandidates(path string) (*candidates, error) {
	c := candidates{
		pkgs:    make([]pkgName, 0),
		funcs:   make(map[pkgName][]funcSet),
		methods: make(map[pkgName][]methodSet),
		vars:    make(map[pkgName][]varSet),
		consts:  make(map[pkgName][]constSet),
		structs: make(map[pkgName][]structSet),
	}
	node, err := analyzeGoAst(path)
	if err != nil {
		return nil, err
	}
	for pkg, pkgAst := range node {
		c.processPackageAst(pkg, pkgAst)
	}

	return &c, nil
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func (c *candidates) processPackageAst(pkg string, pkgAst *ast.Package) {
	c.pkgs = append(c.pkgs, pkgName(pkg))
	for _, fileAst := range pkgAst.Files {
		c.processFileAst(pkg, fileAst)
	}
}

func (c *candidates) processFileAst(pkg string, fileAst *ast.File) {
	for _, decl := range fileAst.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if isMethod(d) {
				c.processMethodDecl(pkg, d)
				continue
			}
			c.processFuncDecl(pkg, d)
		}
	}
	for _, decl := range fileAst.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			c.processGenDecl(pkg, d)
		}
	}
}

func (c *candidates) processFuncDecl(pkg string, funcDecl *ast.FuncDecl) {
	var description string
	var returnTypeName []string
	var returnTypePkgName []string
	if funcDecl.Doc != nil {
		description = strings.ReplaceAll(funcDecl.Doc.Text(), "\n", "")
	}
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			var typeName string
			var typePkgName string
			switch resultType := result.Type.(type) {
			case *ast.Ident:
				typeName = resultType.Name
				typePkgName = pkg
			case *ast.SelectorExpr:
				typeName = resultType.Sel.Name
				typePkgName = resultType.X.(*ast.Ident).Name
			case *ast.StarExpr:
				typePkgName = pkg
				if ident, ok := resultType.X.(*ast.Ident); ok {
					typeName = ident.Name
				}
			}
			returnTypeName = append(returnTypeName, typeName)
			returnTypePkgName = append(returnTypePkgName, typePkgName)
		}
	}
	c.funcs[pkgName(pkg)] = append(c.funcs[pkgName(pkg)], funcSet{name: funcDecl.Name.Name, description: description, returnTypeName: returnTypeName, returnTypePkgName: returnTypePkgName})
}

func isMethod(funcDecl *ast.FuncDecl) bool {
	return funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0
}

// TODO: こっちも返り値の型を取得するようにする
func (c *candidates) processMethodDecl(pkg string, funcDecl *ast.FuncDecl) {
	var receiverTypeName string
	var returnTypeName []string
	var returnTypePkgName []string
	switch receiverType := funcDecl.Recv.List[0].Type.(type) {
	case *ast.Ident:
		receiverTypeName = receiverType.Name
	case *ast.StarExpr:
		if ident, ok := receiverType.X.(*ast.Ident); ok {
			receiverTypeName = ident.Name
		}
	}
	var description string
	if funcDecl.Doc != nil {
		description = strings.ReplaceAll(funcDecl.Doc.Text(), "\n", "")
	}
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			var typeName string
			var typePkgName string
			switch resultType := result.Type.(type) {
			case *ast.Ident:
				typeName = resultType.Name
				typePkgName = pkg
			case *ast.SelectorExpr:
				typeName = resultType.Sel.Name
				typePkgName = resultType.X.(*ast.Ident).Name
			case *ast.StarExpr:
				typePkgName = pkg
				if ident, ok := resultType.X.(*ast.Ident); ok {
					typeName = ident.Name
				}
			}
			returnTypeName = append(returnTypeName, typeName)
			returnTypePkgName = append(returnTypePkgName, typePkgName)
		}
		c.methods[pkgName(pkg)] = append(c.methods[pkgName(pkg)], methodSet{name: funcDecl.Name.Name, description: description, receiverTypeName: receiverTypeName, returnTypeName: returnTypeName, returnTypePkgName: returnTypePkgName})
	}
}

func (c *candidates) processGenDecl(pkg string, genDecl *ast.GenDecl) {
	switch genDecl.Tok {
	case token.VAR:
		c.processVarDecl(pkg, genDecl)
	case token.CONST:
		c.processConstDecl(pkg, genDecl)
	case token.TYPE:
		c.processTypeDecl(pkg, genDecl)
	}
}

func (c *candidates) processVarDecl(pkg string, genDecl *ast.GenDecl) {
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
				var typePkgName string
				switch typeExpr := rhs.Type.(type) {
				case *ast.SelectorExpr:
					// パッケージ名付きの型 (pkg.Type{})
					typeName = typeExpr.Sel.Name
					typePkgName = typeExpr.X.(*ast.Ident).Name
				case *ast.Ident:
					// 単純な型名 (Type{})
					typeName = typeExpr.Name
					typePkgName = pkg // 現在のパッケージ名
				}
				c.vars[pkgName(pkg)] = append(c.vars[pkgName(pkg)], varSet{name: name, description: genDeclDescription + specDescription, typeName: typeName, typePkgName: typePkgName})
			case *ast.UnaryExpr:
				if rhs.Op == token.AND {
					// & 演算子の場合
					if compLit, ok := rhs.X.(*ast.CompositeLit); ok {
						// 構造体リテラルの型を適切に処理
						var typeName string
						var typePkgName string
						switch typeExpr := compLit.Type.(type) {
						case *ast.SelectorExpr:
							// パッケージ名付きの型 (pkg.Type{})
							typeName = typeExpr.Sel.Name
							typePkgName = typeExpr.X.(*ast.Ident).Name
						case *ast.Ident:
							// 単純な型名 (Type{})
							typeName = typeExpr.Name
							typePkgName = pkg // 現在のパッケージ名
						}
						c.vars[pkgName(pkg)] = append(c.vars[pkgName(pkg)], varSet{name: name, description: genDeclDescription + specDescription, typeName: typeName, typePkgName: typePkgName})
					}
				}
			case *ast.CallExpr:
				// funcSetから関数の戻り値を取得
				switch funExpr := rhs.Fun.(type) {
				case *ast.SelectorExpr:
					// パッケージ名付きの関数呼び出し (pkg.Func())
					if pkgIdent, ok := funExpr.X.(*ast.Ident); ok {
						funcPkgName := pkgIdent.Name
						funcName := funExpr.Sel.Name
						if funcSets, ok := c.funcs[pkgName(funcPkgName)]; ok {
							for _, funcSet := range funcSets {
								if funcSet.name == funcName {
									typeName := funcSet.returnTypeName[i]
									typePkgName := funcSet.returnTypePkgName[i]
									c.vars[pkgName(pkg)] = append(c.vars[pkgName(pkg)], varSet{name: name, description: genDeclDescription + specDescription, typeName: typeName, typePkgName: typePkgName})
								}
							}
						}
					}
				case *ast.Ident:
					// ローカル関数呼び出し (Func())
					funcName := funExpr.Name
					// 現在のパッケージから関数を探す
					if funcSets, ok := c.funcs[pkgName(pkg)]; ok {
						for _, funcSet := range funcSets {
							if funcSet.name == funcName {
								typeName := funcSet.returnTypeName[i]
								typePkgName := funcSet.returnTypePkgName[i]
								c.vars[pkgName(pkg)] = append(c.vars[pkgName(pkg)], varSet{name: name, description: genDeclDescription + specDescription, typeName: typeName, typePkgName: typePkgName})
							}
						}
					}
				}
			case *ast.BasicLit:
				// 基本リテラル (文字列、数値など)
				var typeName string
				var typePkgName string

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
				c.vars[pkgName(pkg)] = append(c.vars[pkgName(pkg)], varSet{
					name:        name,
					description: genDeclDescription + specDescription,
					typeName:    typeName,
					typePkgName: typePkgName,
				})
			}
		}
	}
}

func (c *candidates) processConstDecl(pkg string, genDecl *ast.GenDecl) {
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
			c.consts[pkgName(pkg)] = append(c.consts[pkgName(pkg)], constSet{name: constname.Name, description: genDeclDescription + specDescription})
		}
	}
}
func (c *candidates) processTypeDecl(pkg string, genDecl *ast.GenDecl) {
	var genDeclDescription string
	if genDecl.Doc != nil {
		genDeclDescription += strings.TrimSpace(genDecl.Doc.Text())
	}
	for _, spec := range genDecl.Specs {
		typespec := spec.(*ast.TypeSpec)
		var fields []string
		structType, ok := typespec.Type.(*ast.StructType)
		if ok {
			for _, field := range structType.Fields.List {
				if len(field.Names) > 0 {
					fields = append(fields, field.Names[0].Name)
				}
			}
			var specDescription string
			if typespec.Doc != nil {
				specDescription += "   " + strings.TrimSpace(typespec.Doc.Text())
			}
			c.structs[pkgName(pkg)] = append(c.structs[pkgName(pkg)], structSet{name: typespec.Name.Name, fields: fields, description: genDeclDescription + specDescription})
		}
	}
}
