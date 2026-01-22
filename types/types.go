package types

import (
	"go/ast"
)

// PkgName はパッケージ名を表す。
type PkgName string

// GoAstNodes は Go の抽象構文木ノードの集合を表す。
// キーはパッケージ名、値はそのパッケージに属する ast.Package ノードのスライス.
type GoAstNodes map[PkgName][]*ast.Package

// DeclName は宣言名を表す。
type DeclName string

// StructFieldName は構造体のフィールド名を表す。
type StructFieldName string

// TypeName は型名を表す。
type TypeName string

// ReceiverName はメソッドのレシーバ型名を表す。
type ReceiverTypeName TypeName

type ImportPath string
