package decl_registry

import "github.com/kakkky/gonsole/types"

type Decl struct {
	name        types.DeclName
	isReturnVal bool    // 関数やメソッドの戻り値かどうか
	returnedIdx int     // 関数やメソッドの戻り値の場合、その戻り値の順番
	rhs         declRHS // 宣言の右辺
}

func (d Decl) Name() types.DeclName {
	return d.name
}

func (d Decl) RHS() declRHS {
	return d.rhs
}

func (d Decl) IsReturnVal() (ok bool, returnedIdx int) {
	return d.isReturnVal, d.returnedIdx
}
