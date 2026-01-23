package decl_registry

import "github.com/kakkky/gonsole/types"

// いずれかのフィールドのみがセットされる
type declRhs struct {
	name    types.DeclName
	kind    rhsKind
	pkgName types.PkgName
}

type rhsKind int

const (
	DeclRhsKindUnknown rhsKind = iota
	DeclRhsKindVar
	DeclRhsKindStruct
	DeclRhsKindFunc
	DeclRhsKindMethod
)

func (rhs declRhs) Name() types.DeclName {
	return rhs.name
}

func (rhs declRhs) Kind() rhsKind {
	return rhs.kind
}

func (rhs declRhs) PkgName() types.PkgName {
	return rhs.pkgName
}
