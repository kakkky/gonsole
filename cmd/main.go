package main

import (
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/executor"
	"github.com/kakkky/gonsole/repl"
)

func main() {
	candidates, err := completer.GenerateCandidates(".")
	if err != nil {
		panic(err)
	}
	repl := repl.NewRepl(completer.NewCompleter(candidates), executor.NewExecutor())
	repl.Run()
}
