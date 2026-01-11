package registry

import "github.com/kakkky/gonsole/types"

type declRhs struct {
	declStruct declStruct
	declVar    declVar
	declFunc   declFunc
	declMethod declMethod
}

type declStruct struct {
	name types.DeclName
}
type declVar struct {
	name types.DeclName
}

type declFunc struct {
	name  types.DeclName
	order int
}
type declMethod struct {
	name  types.DeclName
	order int
}

func (r declRhs) Struct() declStruct {
	return r.declStruct
}
func (s declStruct) Name() types.DeclName {
	return s.name
}

func (r declRhs) Var() declVar {
	return r.declVar
}
func (v declVar) Name() types.DeclName {
	return v.name
}

func (r declRhs) Func() declFunc {
	return r.declFunc
}
func (f declFunc) Name() types.DeclName {
	return f.name
}
func (f declFunc) ReturnedOrder() int {
	return f.order
}

func (r declRhs) Method() declMethod {
	return r.declMethod
}
func (m declMethod) Name() types.DeclName {
	return m.name
}
func (m declMethod) ReturnedOrder() int {
	return m.order
}
