package main

import (
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/decls"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/executor"
	"github.com/kakkky/gonsole/repl"
)

func main() {
	candidates, err := completer.NewCandidates(".")
	if err != nil {
		errs.HandleError(err)
	}
	declEntry := decls.NewDeclEntry()
	executor, err := executor.NewExecutor(declEntry)
	if err != nil {
		errs.HandleError(err)
	}
	completer := completer.NewCompleter(candidates, declEntry)
	repl := repl.NewRepl(completer, executor)
	repl.Run()
}
