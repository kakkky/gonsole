package main

import (
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/decl_registry"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/executor"
	"github.com/kakkky/gonsole/repl"
)

func main() {
	// candidates, err := completer.NewCandidates(".")
	// if err != nil {
	// 	errs.HandleError(err)
	// }
	registry := decl_registry.NewRegistry()
	executor, err := executor.NewExecutor(registry)
	if err != nil {
		errs.HandleError(err)
	}
	completer, err := completer.NewCompleter(registry)
	if err != nil {
		errs.HandleError(err)
	}
	repl := repl.NewRepl(completer, executor)
	if err := repl.Run(); err != nil {
		errs.HandleError(err)
	}
}
