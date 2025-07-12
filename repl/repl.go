package repl

import (
	"fmt"
	"os"

	"github.com/c-bata/go-prompt"
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/executor"
)

type Repl struct {
	pt *prompt.Prompt
}

func NewRepl(completer *completer.Completer, executor *executor.Executor) *Repl {
	pt := prompt.New(
		executor.Execute,
		completer.Complete,
		prompt.OptionTitle("Gonsole"),
		prompt.OptionAddKeyBind(keyBinds...),
	)
	return &Repl{
		pt: pt,
	}
}

func (r *Repl) Run() {
	r.pt.Run()
}

var keyBinds = []prompt.KeyBind{
	{
		Key: prompt.ControlC,
		Fn: func(buf *prompt.Buffer) {
			fmt.Println("\nExit on Ctrl+C")
			os.Exit(0)
		},
	},
}
