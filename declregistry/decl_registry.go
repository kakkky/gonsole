package declregistry

import (
	gotypes "go/types"

	"github.com/kakkky/gonsole/types"
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
func (dr *DeclRegistry) Register(declName types.DeclName, declType gotypes.Type) error {
	var typeName types.TypeName
	var typePkgName types.PkgName
	var pointered bool

	switch declTypeV := declType.(type) {
	case *gotypes.Named:
		typeName = types.TypeName(declTypeV.Obj().Name())
		if declTypeV.Obj().Pkg() != nil {
			typePkgName = types.PkgName(declTypeV.Obj().Pkg().Name())
		}
	case *gotypes.Pointer:
		pointered = true
		switch pointeredTypV := declTypeV.Elem().(type) {
		case *gotypes.Named:
			typeName = types.TypeName(pointeredTypV.Obj().Name())
			if pointeredTypV.Obj().Pkg() != nil {
				typePkgName = types.PkgName(pointeredTypV.Obj().Pkg().Name())
			}
		default:
			typeName = types.TypeName(pointeredTypV.String())
		}
	default:
		typeName = types.TypeName(declType.String())
	}
	dr.register(Decl{
		Name:        declName,
		Pointered:   pointered,
		TypeName:    typeName,
		TypePkgName: typePkgName,
	})
	return nil
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
