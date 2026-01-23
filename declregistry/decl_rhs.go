package declregistry

import "github.com/kakkky/gonsole/types"

// いずれかのフィールドのみがセットされる
type declRHS struct {
	name    types.DeclName
	kind    rhsKind
	pkgName types.PkgName
}

type rhsKind int

// rhsKind の種類
const (
	DeclRHSKindUnknown rhsKind = iota // 右辺の種類が不明な場合
	DeclRHSKindVar                    // 右辺が変数の場合
	DeclRHSKindStruct                 // 右辺が構造体の場合
	DeclRHSKindFunc                   // 右辺が関数の場合
	DeclRHSKindMethod                 // 右辺がメソッドの場合
)

// Name は右辺の名前を返す
func (rhs declRHS) Name() types.DeclName {
	return rhs.name
}

// Kind は右辺の種類を返す
func (rhs declRHS) Kind() rhsKind {
	return rhs.kind
}

// PkgName は右辺の属するパッケージ名を返す
func (rhs declRHS) PkgName() types.PkgName {
	return rhs.pkgName
}
