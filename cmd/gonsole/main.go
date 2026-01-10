package main

import (
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/executor"
	"github.com/kakkky/gonsole/registry"
	"github.com/kakkky/gonsole/repl"
	"github.com/kakkky/gonsole/utils"
)

func main() {
	nodes, fset, err := utils.AnalyzeGoAst(".")
	if err != nil {
		errs.HandleError(err)
	}
	candidates, err := completer.NewCandidates(nodes)
	if err != nil {
		errs.HandleError(err)
	}
	registry := registry.NewRegistry()
	executor, err := executor.NewExecutor(registry, nodes, fset)
	if err != nil {
		errs.HandleError(err)
	}
	completer := completer.NewCompleter(candidates, registry)
	repl := repl.NewRepl(completer, executor)
	if err := repl.Run(); err != nil {
		errs.HandleError(err)
	}
}
