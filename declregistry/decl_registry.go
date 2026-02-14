package declregistry

import (
	"fmt"
	"go/ast"

	gotypes "go/types"

	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/filer"
	"github.com/kakkky/gonsole/types"
	"golang.org/x/tools/go/packages"
)

// DeclRegistry はReplセッション中に宣言された変数の情報を管理する
type DeclRegistry struct {
	Decls []Decl
	filer.Filer
}

// NewRegistry はDeclRegistryのインスタンスを生成する
func NewRegistry() *DeclRegistry {
	return &DeclRegistry{
		Decls: []Decl{},
		Filer: filer.NewDefaultFiler(),
	}
}

// Register は入力されたコードを解析し、宣言された変数情報を登録する
func (dr *DeclRegistry) Register(input string, importPath types.ImportPath) error {
	tmpFile, tmpFileName, cleanup, err := dr.Filer.CreateTmpFile()
	if err != nil {
		return errs.NewInternalError("failed to create temp file").Wrap(err)
	}
	defer cleanup()

	var importStmt string
	if importPath != "" {
		importStmt = fmt.Sprintf("import %s\n\n", importPath)
	}
	wrappedSrc := "package main\n" + importStmt + "func tmp() {\n" + input + "\n}"
	if _, err := tmpFile.WriteString(wrappedSrc); err != nil {
		return errs.NewInternalError("failed to write temp file").Wrap(err)
	}
	if err := tmpFile.Close(); err != nil {
		return errs.NewInternalError("failed to close temp file").Wrap(err)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  "",
	}

	pkgs, err := packages.Load(cfg, tmpFileName)
	if err != nil || len(pkgs) == 0 {
		return errs.NewInternalError("failed to load package").Wrap(err)
	}

	pkg := pkgs[0]
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if funcDecl.Name.Name != "tmp" {
				continue
			}
			for _, stmt := range funcDecl.Body.List {
				switch stmtV := stmt.(type) {
				case *ast.AssignStmt:
					dr.registerAssimentStmt(stmtV, pkg.TypesInfo)
				case *ast.DeclStmt:
					dr.registerDeclStmt(stmtV, pkg.TypesInfo)
				}
			}
		}
	}
	return nil
}

func (dr *DeclRegistry) registerAssimentStmt(assignmentStmt *ast.AssignStmt, typesInfo *gotypes.Info) {
	for _, stmtLHS := range assignmentStmt.Lhs {
		lhsIdent, ok := stmtLHS.(*ast.Ident)
		if !ok {
			continue
		}
		typ := typesInfo.TypeOf(stmtLHS)
		var typeName types.TypeName
		var typePkgName types.PkgName
		var pointered bool

		switch typV := typ.(type) {
		case *gotypes.Named:
			typeName = types.TypeName(typV.Obj().Name())
			if typV.Obj().Pkg() != nil {
				typePkgName = types.PkgName(typV.Obj().Pkg().Name())
			}
		case *gotypes.Pointer:
			pointered = true
			switch pointeredTypV := typV.Elem().(type) {
			case *gotypes.Named:
				typeName = types.TypeName(pointeredTypV.Obj().Name())
				if pointeredTypV.Obj().Pkg() != nil {
					typePkgName = types.PkgName(pointeredTypV.Obj().Pkg().Name())
				}
			default:
				typeName = types.TypeName(pointeredTypV.String())
			}
		default:
			typeName = types.TypeName(typ.String())
		}
		dr.register(Decl{
			Name:        types.DeclName(lhsIdent.Name),
			Pointered:   pointered,
			TypeName:    typeName,
			TypePkgName: typePkgName,
		})
	}
}

func (dr *DeclRegistry) registerDeclStmt(declStmt *ast.DeclStmt, typesInfo *gotypes.Info) {
	switch stmtDeclV := declStmt.Decl.(type) {
	case *ast.GenDecl:
		for _, stmtDeclSpec := range stmtDeclV.Specs {
			switch stmtDeclSpecV := stmtDeclSpec.(type) {
			case *ast.ValueSpec:
				for _, name := range stmtDeclSpecV.Names {
					typ := typesInfo.TypeOf(name)
					var typeName types.TypeName
					var typePkgName types.PkgName
					var pointered bool

					switch typV := typ.(type) {
					case *gotypes.Named:
						typeName = types.TypeName(typV.Obj().Name())
						if typV.Obj().Pkg() != nil {
							typePkgName = types.PkgName(typV.Obj().Pkg().Name())
						}
					case *gotypes.Pointer:
						pointered = true
						switch pointeredTypV := typV.Elem().(type) {
						case *gotypes.Named:
							typeName = types.TypeName(pointeredTypV.Obj().Name())
							if pointeredTypV.Obj().Pkg() != nil {
								typePkgName = types.PkgName(pointeredTypV.Obj().Pkg().Name())
							}
						default:
							typeName = types.TypeName(pointeredTypV.String())
						}
					default:
						typeName = types.TypeName(typ.String())
					}
					dr.register(Decl{
						Name:        types.DeclName(name.Name),
						Pointered:   pointered,
						TypeName:    typeName,
						TypePkgName: typePkgName,
					})
				}
			}
		}
	}
}

func (dr *DeclRegistry) register(decl Decl) {
	dr.Decls = append(dr.Decls, decl)
}

// IsRegisteredDecl は指定された名前の宣言が登録されているかを返す
func (dr *DeclRegistry) IsRegisteredDecl(name types.DeclName) bool {
	for _, decl := range dr.Decls {
		if decl.Name == name {
			return true
		}
	}
	return false
}
