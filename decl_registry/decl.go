package decl_registry

import "github.com/kakkky/gonsole/types"

// Decl はReplセッション内で宣言された変数の情報を表す
type Decl struct {
	name        types.DeclName
	isReturnVal bool    // 関数やメソッドの戻り値かどうか
	returnedIdx int     // 関数やメソッドの戻り値の場合、その戻り値の順番
	rhs         declRHS // 宣言の右辺
}

// Name は宣言された変数の名前を返す
func (d Decl) Name() types.DeclName {
	return d.name
}

// RHS は宣言の右辺の情報を返す
func (d Decl) RHS() declRHS {
	return d.rhs
}

// IsReturnVal はその宣言が関数やメソッドの戻り値かどうかを返す
func (d Decl) IsReturnVal() (ok bool, returnedIdx int) {
	return d.isReturnVal, d.returnedIdx
}
