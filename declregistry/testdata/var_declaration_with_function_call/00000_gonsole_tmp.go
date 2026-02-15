package main

import "github.com/kakkky/gonsole/declregistry/testdata/var_declaration_with_function_call/sample"

func main() {
	var f = sample.Func()
	_ = f
}
