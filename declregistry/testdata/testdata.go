package testdata

var Var int = 123

type Struct struct {
	X int
}

func Func() int {
	return 42
}

func MultiReturn() (int, string) {
	return 1, "a"
}

func (s Struct) Method1() Struct { return s }
func (s Struct) Method2() string { return "ok" }
