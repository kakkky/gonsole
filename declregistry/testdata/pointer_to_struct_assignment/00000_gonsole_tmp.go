package main

import "github.com/kakkky/gonsole/declregistry/testdata/pointer_to_struct_assignment/sample"

func main() {
	p := &sample.Struct{}
	_ = p
}
