package completer

import "github.com/c-bata/go-prompt"

type Completer struct{}

func NewCompleter() *Completer {
	return &Completer{}
}

func (c *Completer) Complete(input prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{
		{Text: "help", Description: "Show help"},
	}
}
