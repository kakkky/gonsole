package completer

import (
	"strings"

	"github.com/c-bata/go-prompt"
)

type Completer struct {
	candidates *candidates
}

func NewCompleter(candidates *candidates) *Completer {
	return &Completer{
		candidates: candidates,
	}
}

type pkgAndInput struct {
	pkg   string
	input string
}

func (c *Completer) Complete(input prompt.Document) []prompt.Suggest {
	inputBeforeCursor := input.TextBeforeCursor()
	inputStr := input.Text
	suggestions := make([]prompt.Suggest, 0)

	// 変数宣言の際に = が含まれていたら、その後の文字列を補完の検索に使用する。ただ、これのせいで、＝より前がそのまま残って補完を適用してしまう。要KAIZEN
	equalPos, found := findEqualPos(inputStr)
	if found {
		inputStr = inputStr[equalPos+1:]
	}

	// . までは、packageの候補を表示する
	if !strings.Contains(inputStr, ".") {
		for _, pkg := range c.candidates.pkgs {
			if strings.HasPrefix(string(pkg), inputStr) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        string(pkg),
					Description: "Package",
				})
			}
		}
		return suggestions
	}

	pkgAndInput := separatePkgAndInput(inputStr)
	findedSuggestions := c.findSuggestions(pkgAndInput, inputBeforeCursor)
	suggestions = append(suggestions, findedSuggestions...)

	return suggestions
}

func (c *Completer) findSuggestions(pai pkgAndInput, ibc string) []prompt.Suggest {
	var suggestions []prompt.Suggest
	if funcSets, ok := c.candidates.funcs[pkgName(pai.pkg)]; ok {
		for _, funcSet := range funcSets {
			if strings.HasPrefix(funcSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        ibc + funcSet.name + "()",
					Description: "Function: " + funcSet.description,
				})
			}
		}
	}
	if varSets, ok := c.candidates.vars[pkgName(pai.pkg)]; ok {
		for _, varSet := range varSets {
			if strings.HasPrefix(varSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        ibc + varSet.name,
					Description: "Variable: " + varSet.description,
				})
			}
		}
	}
	if constSets, ok := c.candidates.consts[pkgName(pai.pkg)]; ok {
		for _, constSet := range constSets {
			if strings.HasPrefix(constSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        ibc + constSet.name,
					Description: "Constant: " + constSet.description,
				})
			}
		}
	}
	if typeSets, ok := c.candidates.types[pkgName(pai.pkg)]; ok {
		for _, typeSet := range typeSets {
			if strings.HasPrefix(typeSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        ibc + typeSet.name,
					Description: "Type: " + typeSet.description,
				})
			}
		}
	}
	return suggestions
}

// {pkg名}. まで入力されている場合は、pkg名とその後の文字列を構造体にまとめる
func separatePkgAndInput(input string) pkgAndInput {
	var pkgAndInput pkgAndInput
	if strings.Contains(input, ".") {
		parts := strings.SplitN(input, ".", 2)
		pkgAndInput.pkg = parts[0]
		if len(parts) == 2 {
			pkgAndInput.input = parts[1]
		}
	}
	return pkgAndInput
}

func findEqualPos(input string) (int, bool) {
	equalPos := strings.LastIndex(input, "=")
	if equalPos == -1 {
		return -1, false
	}
	return equalPos, true
}
