package main

import "github.com/kakkky/gonsole/declregistry/testdata/multiple_return_values_from_function/sample"

func main() {
	a, b := sample.MultiReturn()
	_ = a
	_ = b
}
