package executor

import "fmt"

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(input string) {
	fmt.Println("Executing command:", input)
}
