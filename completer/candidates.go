package completer

import (
	"go/ast"
	gotypes "go/types"
	"slices"
	"strings"

	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
	"golang.org/x/tools/go/packages"
)

// BuildStdPkgCandidatesModeは標準パッケージの候補を構築するモードのフラグ
var BuildStdPkgCandidatesMode bool

// SkipStdPkgMergeはテスト時に標準パッケージのマージをスキップするフラグ
var SkipStdPkgMergeMode bool

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
		TypePkgName types.PkgName
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
	TypeName    types.TypeName
	TypePkgName types.PkgName
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func NewCandidates(projectRootPath string) (*candidates, error) {
	pkgs, err := loadProject(projectRootPath)
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

	// パッケージスコープごとに処理
	for _, pkg := range pkgs {
		pkgName := types.PkgName(pkg.Name)
		c.Pkgs = append(c.Pkgs, pkgName)
		c.processScope(pkgName, pkg.Types.Scope(), pkg.Syntax)
	}

	// 標準パッケージの候補とマージ（テスト時、標準パッケージ候補の生成スクリプト実行時はスキップ）
	if !SkipStdPkgMergeMode {
		c.mergeCandidates(stdPkgCandidates)
	}

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

func loadProject(path string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax, // コメント情報などはASTからしか取れない
		Dir: path,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, errs.NewInternalError("failed to load packages").Wrap(err)
	}
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return nil, errs.NewInternalError("failed to load packages: ").Wrap(pkg.Errors[0])
		}
	}

	pkgs = slices.DeleteFunc(pkgs, func(pkg *packages.Package) bool {
		if pkg.Name == "main" {
			return true
		}
		return false
	})
	return pkgs, nil
}

func (c *candidates) processScope(pkgName types.PkgName, scope *gotypes.Scope, astFiles []*ast.File) {
	for _, declName := range scope.Names() {
		declObj := scope.Lookup(declName)
		if !declObj.Exported() {
			continue
		}
		if declName == "" || declName == "_" || strings.HasPrefix(declName, "_") {
			continue
		}

		declAst := detectDecl(declName, astFiles)
		if declAst == nil {
			continue
		}

		switch declObjV := declObj.(type) {
		case *gotypes.Func:
			funcDecl, ok := declAst.(*ast.FuncDecl)
			if !ok {
				continue
			}
			c.processFuncDeclObj(pkgName, declObjV, funcDecl)
		case *gotypes.TypeName:
			genDecl, ok := declAst.(*ast.GenDecl)
			if !ok {
				continue
			}
			c.processTypeDeclObj(pkgName, declObjV, genDecl)
			switch decTypeObjV := declObjV.Type().(type) {
			case *gotypes.Named:
				for i := 0; i < decTypeObjV.NumMethods(); i++ {
					methodObj := decTypeObjV.Method(i)
					methodDeclAst := detectDecl(methodObj.Name(), astFiles)
					if methodDeclAst == nil {
						continue
					}
					methodDecl, ok := methodDeclAst.(*ast.FuncDecl)
					if !ok {
						continue
					}
					c.processMethodDeclObj(pkgName, methodObj, methodDecl)
				}
			}
		case *gotypes.Var:
			genDecl, ok := declAst.(*ast.GenDecl)
			if !ok {
				continue
			}
			c.processVarDeclObj(pkgName, declObjV, genDecl)
		case *gotypes.Const:
			genDecl, ok := declAst.(*ast.GenDecl)
			if !ok {
				continue
			}
			c.processConstDeclObj(pkgName, declObjV, genDecl)
		}
	}
}

// detectDecl は与えられた名前の宣言をASTファイル群から探し出す
func detectDecl(declName string, astFiles []*ast.File) ast.Decl {
	for _, astFile := range astFiles {
		for _, decl := range astFile.Decls {
			switch declV := decl.(type) {
			case *ast.FuncDecl:
				if declV.Name.Name == declName {
					return declV
				}
			case *ast.GenDecl:
				for _, spec := range declV.Specs {
					switch specV := spec.(type) {
					case *ast.TypeSpec:
						if specV.Name.Name == declName {
							return declV
						}
					case *ast.ValueSpec:
						for _, name := range specV.Names {
							if name.Name == declName {
								return declV
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// processFuncDeclObj は関数宣言オブジェクトを処理して候補に追加する
func (c *candidates) processFuncDeclObj(pkgName types.PkgName, funcDeclObj *gotypes.Func, funcDeclAst *ast.FuncDecl) {
	var description string
	if funcDeclAst.Doc != nil {
		description = funcDeclAst.Doc.Text()
	}

	var returns []returnSet
	results := funcDeclObj.Signature().Results()

	if results != nil {
		for i := 0; i < results.Len(); i++ {
			returnType := results.At(i).Type()
			var returnTypeName types.TypeName
			var returnTypePkgName types.PkgName

			switch returnTypeV := returnType.(type) {
			case *gotypes.Named:
				returnTypeName = types.TypeName(returnTypeV.Obj().Name())
				if returnTypeV.Obj().Pkg() != nil {
					returnTypePkgName = types.PkgName(returnTypeV.Obj().Pkg().Name())
				}
			case *gotypes.Pointer:
				switch pointedTypeV := returnTypeV.Elem().(type) {
				case *gotypes.Named:
					returnTypeName = types.TypeName(pointedTypeV.Obj().Name())
					if pointedTypeV.Obj().Pkg() != nil {
						returnTypePkgName = types.PkgName(pointedTypeV.Obj().Pkg().Name())
					}
				default:
					returnTypeName = types.TypeName(returnType.String())
				}
			default:
				returnTypeName = types.TypeName(returnType.String())
			}
			returns = append(returns, returnSet{
				TypeName:    returnTypeName,
				TypePkgName: returnTypePkgName,
			})
		}
	}

	c.Funcs[pkgName] = append(c.Funcs[pkgName], funcSet{Name: types.DeclName(funcDeclObj.Name()), Description: description, Returns: returns})
}

// processMethodDeclObj はメソッド宣言オブジェクトを処理して候補に追加する
func (c *candidates) processMethodDeclObj(pkgName types.PkgName, methodDeclObj *gotypes.Func, methodDeclAst *ast.FuncDecl) {
	var description string
	if methodDeclAst.Doc != nil {
		description = methodDeclAst.Doc.Text()
	}

	var receiverTypeName types.ReceiverTypeName
	recv := methodDeclObj.Signature().Recv()
	switch recvTypeV := recv.Type().(type) {
	case *gotypes.Named:
		receiverTypeName = types.ReceiverTypeName(recvTypeV.Obj().Name())
	case *gotypes.Pointer:
		switch pointedTypeV := recvTypeV.Elem().(type) {
		case *gotypes.Named:
			receiverTypeName = types.ReceiverTypeName(pointedTypeV.Obj().Name())
		}
	}

	var returns []returnSet
	results := methodDeclObj.Signature().Results()

	if results != nil {
		for i := 0; i < results.Len(); i++ {
			returnType := results.At(i).Type()
			var returnTypeName types.TypeName
			var returnTypePkgName types.PkgName

			switch returnTypeV := returnType.(type) {
			case *gotypes.Named:
				returnTypeName = types.TypeName(returnTypeV.Obj().Name())
				if returnTypeV.Obj().Pkg() != nil {
					returnTypePkgName = types.PkgName(returnTypeV.Obj().Pkg().Name())
				}
			case *gotypes.Pointer:
				switch pointedTypeV := returnTypeV.Elem().(type) {
				case *gotypes.Named:
					returnTypeName = types.TypeName(pointedTypeV.Obj().Name())
					if pointedTypeV.Obj().Pkg() != nil {
						returnTypePkgName = types.PkgName(pointedTypeV.Obj().Pkg().Name())
					}
				default:
					returnTypeName = types.TypeName(returnTypeV.String())
				}
			default:
				returnTypeName = types.TypeName(returnType.String())
			}

			returns = append(returns, returnSet{
				TypeName:    returnTypeName,
				TypePkgName: returnTypePkgName,
			})
		}
	}

	c.Methods[pkgName] = append(c.Methods[pkgName], methodSet{
		Name:             types.DeclName(methodDeclObj.Name()),
		Description:      description,
		ReceiverTypeName: receiverTypeName,
		Returns:          returns,
	})

}

// processTypeDeclObj は型宣言オブジェクトを処理して候補に追加する
func (c *candidates) processTypeDeclObj(pkgName types.PkgName, typeDeclObj *gotypes.TypeName, genDeclAst *ast.GenDecl) {
	declName := types.DeclName(typeDeclObj.Name())

	var typeDeclAst *ast.TypeSpec
	for _, spec := range genDeclAst.Specs {
		switch specV := spec.(type) {
		case *ast.TypeSpec:
			if specV.Name.Name == string(declName) {
				typeDeclAst = specV
				break
			}
		}
	}

	underlyingType := typeDeclObj.Type().Underlying()
	switch underlyingTypeV := underlyingType.(type) {
	case *gotypes.Struct:
		c.processStructTypeDeclObj(pkgName, declName, underlyingTypeV, genDeclAst)
	case *gotypes.Interface:
		c.processInterfaceTypeDeclObj(pkgName, declName, underlyingTypeV, typeDeclAst)
	default:
		// c.processDefinedTypeDeclObj(pkgName, declName, underlyingTypeV, typeDeclAst)
	}
}

// processStructTypeDeclObj は構造体型宣言オブジェクトを処理して候補に追加する
func (c *candidates) processStructTypeDeclObj(pkgName types.PkgName, declName types.DeclName, structDeclObj *gotypes.Struct, genDeclAst *ast.GenDecl) {
	var description string
	if genDeclAst != nil && genDeclAst.Doc != nil {
		description = genDeclAst.Doc.Text()
	}

	var fields []types.StructFieldName
	for i := 0; i < structDeclObj.NumFields(); i++ {
		fieldObj := structDeclObj.Field(i)
		fields = append(fields, types.StructFieldName(fieldObj.Name()))
	}

	c.Structs[pkgName] = append(c.Structs[pkgName], structSet{
		Name:        declName,
		Fields:      fields,
		Description: description,
	})
}

// processInterfaceTypeDeclObj はインターフェース型宣言オブジェクトを処理して候補に追加する
func (c *candidates) processInterfaceTypeDeclObj(pkgName types.PkgName, declName types.DeclName, interfaceDeclObj *gotypes.Interface, typeDeclAst *ast.TypeSpec) {
	var methods []types.DeclName
	var descriptions []string

	// 各メソッドのドキュメントを取得
	for i := 0; i < interfaceDeclObj.NumMethods(); i++ {
		methodObj := interfaceDeclObj.Method(i)
		methods = append(methods, types.DeclName(methodObj.Name()))

		// ASTからメソッドのドキュメントを探す
		var description string
		switch typeDeclAstV := typeDeclAst.Type.(type) {
		case *ast.InterfaceType:
			for _, field := range typeDeclAstV.Methods.List {
				for _, name := range field.Names {
					if name.Name == methodObj.Name() {
						if field.Doc != nil {
							description = field.Doc.Text()
						}
						break
					}
				}
			}
		}
		descriptions = append(descriptions, description)
	}

	c.Interfaces[pkgName] = append(c.Interfaces[pkgName], interfaceSet{
		Name:         declName,
		Methods:      methods,
		Descriptions: descriptions,
	})
}

func (c *candidates) processVarDeclObj(pkgName types.PkgName, varDeclObj *gotypes.Var, genDeclAst *ast.GenDecl) {
	declName := types.DeclName(varDeclObj.Name())

	var varDeclAst *ast.ValueSpec
	for _, spec := range genDeclAst.Specs {
		switch specV := spec.(type) {
		case *ast.ValueSpec:
			for _, name := range specV.Names {
				if name.Name == string(declName) {
					varDeclAst = specV
					break
				}
			}
		}
	}

	var description string
	if varDeclAst != nil && varDeclAst.Doc != nil {
		description = varDeclAst.Doc.Text()
	} else if genDeclAst != nil && genDeclAst.Doc != nil {
		description = genDeclAst.Doc.Text()
	}

	var typeName types.TypeName
	var typePkgName types.PkgName
	switch varTypeV := varDeclObj.Type().(type) {
	case *gotypes.Named:
		typeName = types.TypeName(varTypeV.Obj().Name())
		if varTypeV.Obj().Pkg() != nil {
			typePkgName = types.PkgName(varTypeV.Obj().Pkg().Name())
		}
	case *gotypes.Pointer:
		switch pointedTypeV := varTypeV.Elem().(type) {
		case *gotypes.Named:
			typeName = types.TypeName(pointedTypeV.Obj().Name())
			if pointedTypeV.Obj().Pkg() != nil {
				typePkgName = types.PkgName(pointedTypeV.Obj().Pkg().Name())
			}
		default:
			typeName = types.TypeName(varTypeV.String())
		}
	default:
		typeName = types.TypeName(varTypeV.String())
	}

	c.Vars[pkgName] = append(c.Vars[pkgName], varSet{
		Name:        declName,
		Description: description,
		TypeName:    typeName,
		TypePkgName: typePkgName,
	})
}

func (c *candidates) processConstDeclObj(pkgName types.PkgName, constDeclObj *gotypes.Const, genDeclAst *ast.GenDecl) {
	declName := types.DeclName(constDeclObj.Name())

	var constDeclAst *ast.ValueSpec
	for _, spec := range genDeclAst.Specs {
		switch specV := spec.(type) {
		case *ast.ValueSpec:
			for _, name := range specV.Names {
				if name.Name == string(declName) {
					constDeclAst = specV
					break
				}
			}
		}
	}

	var description string
	if constDeclAst != nil && constDeclAst.Doc != nil {
		description = constDeclAst.Doc.Text()
	} else if genDeclAst != nil && genDeclAst.Doc != nil {
		description = genDeclAst.Doc.Text()
	}

	c.Consts[pkgName] = append(c.Consts[pkgName], constSet{
		Name:        declName,
		Description: description,
	})
}
