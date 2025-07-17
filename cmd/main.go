package main

import (
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/executor"
	"github.com/kakkky/gonsole/repl"
	"github.com/kakkky/gonsole/utils"
)

func main() {
	nodes, err := utils.AnalyzeGoAst(".")
	if err != nil {
		panic(err)
	}
	candidates := completer.ConvertFromNodeToCandidates(nodes)
	importPaths := executor.ExtractImportPaths(nodes, "go.mod")
	repl := repl.NewRepl(completer.NewCompleter(candidates), executor.NewExecutor(importPaths))
	repl.Run()
}
