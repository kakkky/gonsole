package declregistry

import "github.com/kakkky/gonsole/types"

// Decl はReplセッション内で宣言された変数の情報を表す
type Decl struct {
	Name        types.DeclName
	Pointered   bool
	TypeName    types.TypeName
	TypePkgName types.PkgName
}

// IsPointered は宣言された変数がポインタ型かどうかを返す
func (d Decl) IsPointered() bool {
	return d.Pointered
}
