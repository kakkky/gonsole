package main

import "github.com/kakkky/gonsole/declregistry/testdata/method_assignment/sample"

func main() {
	a := sample.Struct{}
	_ = a
	b := a.Method1()
	_ = b
}
