package main

import (
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/declregistry"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/executor"
	"github.com/kakkky/gonsole/repl"
	"github.com/kakkky/gonsole/symbols"
)

func main() {
	symbolIndex, err := symbols.NewSymbolIndex(".")
	if err != nil {
		errs.HandleError(err)
	}
	registry := declregistry.NewRegistry()

	executor, err := executor.NewExecutor(registry)
	if err != nil {
		errs.HandleError(err)
	}
	completer := completer.NewCompleter(registry, symbolIndex)
	repl := repl.NewRepl(completer, executor)
	if err := repl.Run(); err != nil {
		errs.HandleError(err)
	}
}
