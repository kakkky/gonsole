package main

import "github.com/kakkky/gonsole/declregistry/testdata/var_declaration_with_pointer_to_struct/sample"

func main() {
	var p = &sample.Struct{}
	_ = p
}
