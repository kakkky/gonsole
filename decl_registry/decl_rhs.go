package decl_registry

import "github.com/kakkky/gonsole/types"

// いずれかのフィールドのみがセットされる
type declRHS struct {
	name    types.DeclName
	kind    rhsKind
	pkgName types.PkgName
}

type rhsKind int

const (
	DeclRHSKindUnknown rhsKind = iota
	DeclRHSKindVar
	DeclRHSKindStruct
	DeclRHSKindFunc
	DeclRHSKindMethod
)

func (rhs declRHS) Name() types.DeclName {
	return rhs.name
}

func (rhs declRHS) Kind() rhsKind {
	return rhs.kind
}

func (rhs declRHS) PkgName() types.PkgName {
	return rhs.pkgName
}
