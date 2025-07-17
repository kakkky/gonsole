package executor

import (
	"fmt"

	"github.com/kakkky/gonsole/completer"
)

type Executor struct {
	importPaths []string
}

func NewExecutor(importPaths []string) *Executor {
	return &Executor{
		importPaths: importPaths,
	}
}

func (e *Executor) Execute(input string) {
	fmt.Println("Executing command:", input)
	completer.DeclVarRecords = append(completer.DeclVarRecords, completer.DeclVarRecord{Name: "sc", Type: "SubComplexType", Pkg: "subcomplex"})
}
