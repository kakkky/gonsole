package main

import (
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/executor"
	"github.com/kakkky/gonsole/repl"
)

func main() {
	repl := repl.NewRepl(completer.NewCompleter(), executor.NewExecutor())
	repl.Run()
}
