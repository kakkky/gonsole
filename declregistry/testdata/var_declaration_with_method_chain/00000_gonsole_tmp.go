package main

import "github.com/kakkky/gonsole/declregistry/testdata/var_declaration_with_method_chain/sample"

func main() {
	a := sample.Struct{}
	_ = a
	var b = a.Method1().Method2()
	_ = b
}
