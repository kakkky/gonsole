package repl

import (
	"fmt"

	// go:embedディレクティブ用
	_ "embed"
	"os"

	"github.com/kakkky/go-prompt"
	"github.com/kakkky/gonsole/completer"
	"github.com/kakkky/gonsole/executor"
	"github.com/kakkky/gonsole/version"
)

// Repl は対話型コンソールの実現を担う
// 実際は go-prompt をラップしているだけ
type Repl struct {
	pt *prompt.Prompt
}

// NewRepl はReplのインスタンスを生成する
func NewRepl(completer *completer.Completer, executor *executor.Executor) *Repl {
	pt := prompt.New(
		executor.Execute,
		completer.Complete,
		prompt.OptionTitle("gonsole"),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn: func(buf *prompt.Buffer) {
				os.Exit(0)
			},
		}),
	)
	return &Repl{
		pt: pt,
	}
}

// Run はREPLセッションを開始する
func (r *Repl) Run() error {
	printGonsoleASCIIArt()
	version.PrintVersion()
	fmt.Print("\n\n Interactive Golang Execution Console\n\n")
	ok, latestVersion, err := version.IsLatestVersion()
	if err != nil {
		return err
	}
	if !ok {
		version.PrintNoteLatestVersion(latestVersion)
	}

	r.pt.Run()
	return nil
}

//go:embed gonsole_ascii.txt
var gonsoleASCIIArt []byte

func printGonsoleASCIIArt() {
	// Print the ASCII art to the console
	fmt.Print(string(gonsoleASCIIArt))
}
