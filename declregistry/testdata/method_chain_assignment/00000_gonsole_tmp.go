package main

import "github.com/kakkky/gonsole/declregistry/testdata/method_chain_assignment/sample"

func main() {
	a := sample.Struct{}
	_ = a
	b := a.Method1().Method2()
	_ = b
}
