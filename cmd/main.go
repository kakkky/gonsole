package main

import (
	"log"

	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/decls"
	"github.com/kakkky/gonsole/executor"
	"github.com/kakkky/gonsole/repl"
)

func main() {
	candidates, err := completer.NewCandidates(".")
	declEntry := decls.NewDeclEntry()
	if err != nil {
		log.Fatalf("failed to create candidates: %v", err)
	}
	repl := repl.NewRepl(completer.NewCompleter(candidates, declEntry), executor.NewExecutor(declEntry))
	repl.Run()
}
