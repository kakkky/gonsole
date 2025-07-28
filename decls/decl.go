package decls

type decl struct {
	Name string
	Pkg  string
	Rhs  Rhs
}

type Rhs struct {
	Struct Struct
	Var    Var
	Func   Func
	Method Method
}

type Struct struct {
	Type string
}
type Var struct {
	Name string
}

type Func struct {
	Name  string
	Order int
}
type Method struct {
	Name  string
	Order int
}
