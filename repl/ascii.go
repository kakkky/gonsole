package repl

import (
	_ "embed"
	"fmt"
)

//go:embed gonsole_ascii.txt
var ascii []byte

func printAscii() {
	// Print the ASCII art to the console
	fmt.Println(string(ascii))
}
