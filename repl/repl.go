package repl

import (
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
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn: func(buf *prompt.Buffer) {
				executor.Close()
				os.Exit(0)
			},
		}),
	)
	return &Repl{
		pt: pt,
	}
}

func (r *Repl) Run() {
	r.pt.Run()
}
