package registry

type declRhs struct {
	declStruct declStruct
	declVar    declVar
	declFunc   declFunc
	declMethod declMethod
}

type declStruct struct {
	typeName string
}
type declVar struct {
	name string
}

type declFunc struct {
	name  string
	order int
}
type declMethod struct {
	name  string
	order int
}

func (r declRhs) Struct() declStruct {
	return r.declStruct
}
func (s declStruct) Type() string {
	return s.typeName
}

func (r declRhs) Var() declVar {
	return r.declVar
}
func (v declVar) Name() string {
	return v.name
}

func (r declRhs) Func() declFunc {
	return r.declFunc
}
func (f declFunc) Name() string {
	return f.name
}
func (f declFunc) ReturnedOrder() int {
	return f.order
}

func (r declRhs) Method() declMethod {
	return r.declMethod
}
func (m declMethod) Name() string {
	return m.name
}
func (m declMethod) ReturnedOrder() int {
	return m.order
}
