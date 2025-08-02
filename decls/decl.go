package decls

type Decl struct {
	name string
	pkg  string
	rhs  declRhs
}

func (d Decl) Name() string {
	return d.name
}

func (d Decl) Pkg() string {
	return d.pkg
}

func (d Decl) Rhs() declRhs {
	return d.rhs
}
