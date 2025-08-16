package main

import (
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/decls"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/executor"
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
	declEntry := decls.NewDeclEntry()
	executor, err := executor.NewExecutor(declEntry, nodes, fset)
	if err != nil {
		errs.HandleError(err)
	}
	completer := completer.NewCompleter(candidates, declEntry)
	repl := repl.NewRepl(completer, executor)
	repl.Run()
}
