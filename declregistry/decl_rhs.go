package declregistry

import "github.com/kakkky/gonsole/types"

// DeclRHS は宣言の右辺を表す
type DeclRHS struct {
	name    types.DeclName
	kind    RHSKind
	pkgName types.PkgName
}

// RHSKind は右辺の種類を表す
type RHSKind int

// RHSKind の種類
const (
	DeclRHSKindUnknown RHSKind = iota // 右辺の種類が不明な場合
	DeclRHSKindVar                    // 右辺が変数の場合
	DeclRHSKindStruct                 // 右辺が構造体の場合
	DeclRHSKindFunc                   // 右辺が関数の場合
	DeclRHSKindMethod                 // 右辺がメソッドの場合
)

// Name は右辺の名前を返す
func (rhs DeclRHS) Name() types.DeclName {
	return rhs.name
}

// Kind は右辺の種類を返す
func (rhs DeclRHS) Kind() RHSKind {
	return rhs.kind
}

// PkgName は右辺の属するパッケージ名を返す
func (rhs DeclRHS) PkgName() types.PkgName {
	return rhs.pkgName
}
