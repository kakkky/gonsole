package registry

import "github.com/kakkky/gonsole/types"

type Decl struct {
	name    string
	pkgName types.PkgName
	rhs     declRhs
}

func (d Decl) Name() string {
	return d.name
}

func (d Decl) PkgName() types.PkgName {
	return d.pkgName
}

func (d Decl) Rhs() declRhs {
	return d.rhs
}
