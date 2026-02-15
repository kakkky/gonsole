package declregistry

import (
	"go/ast"
	"slices"
	"strings"

	gotypes "go/types"

	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
	"golang.org/x/tools/go/packages"
)

// DeclRegistry はReplセッション中に宣言された変数の情報を管理する
type DeclRegistry struct {
	Decls []Decl
}

// NewRegistry はDeclRegistryのインスタンスを生成する
func NewRegistry() *DeclRegistry {
	return &DeclRegistry{
		Decls: []Decl{},
	}
}

// Register は入力された最後の文を解析して、宣言された変数の情報をDeclRegistryに登録する
func (dr *DeclRegistry) Register(tmpFileName string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  "",
	}

	pkgs, err := packages.Load(cfg, tmpFileName)
	if err != nil || len(pkgs) == 0 {
		return errs.NewInternalError("failed to load package").Wrap(err)
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
		return errs.NewBadInputError("failed to parse input: " + strings.Join(errMsgs, "; "))
	}

	var mainFunc *ast.FuncDecl
	for _, decl := range pkg.Syntax[0].Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == "main" {
			mainFunc = funcDecl
			break
		}
	}
	if mainFunc == nil {
		return errs.NewBadInputError("main function not found")
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

	switch lastStmtV := lastStmt.(type) {
	case *ast.AssignStmt:
		dr.registerAssimentStmt(lastStmtV, pkg.TypesInfo)
	case *ast.DeclStmt:
		dr.registerDeclStmt(lastStmtV, pkg.TypesInfo)
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
