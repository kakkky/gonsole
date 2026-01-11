package types

import (
	"go/ast"
)

type GoAstNodes map[PkgName][]*ast.Package

type DeclName string
type StructFieldName string
type PkgName string
type TypeName string
