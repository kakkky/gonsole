package decls

type decl struct {
	name string
	pkg  string
	rhs  declRhs
}

func (d decl) Name() string {
	return d.name
}

func (d decl) Pkg() string {
	return d.pkg
}

func (d decl) Rhs() declRhs {
	return d.rhs
}
