package sample

type Struct struct{}

func (s Struct) Method1() Struct { return s }
func (s Struct) Method2() string { return "ok" }
