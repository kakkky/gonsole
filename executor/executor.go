package executor

import (
	"fmt"

	"github.com/kakkky/gonsole/completer"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(input string) {
	fmt.Println("Executing command:", input)
	completer.DeclVarInTerminalList = append(completer.DeclVarInTerminalList, completer.DeclVarInTerminal{Name: "sc", Type: "SubComplexType", Pkg: "subcomplex"})
}
