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

func (c *Completer) Complete(input prompt.Document) []prompt.Suggest {
	inputStr := input.Text
	suggestions := make([]prompt.Suggest, 0)

	if strings.Contains(inputStr, "&") {
		isAmpersandInclude = true
		inputStr = strings.ReplaceAll(inputStr, "&", "")
	}

	equalAndSpacePos, found := findEqualAndSpacePos(inputStr)
	if found {
		inputStr = inputStr[equalAndSpacePos+2:]
		// . までは、packageの候補を表示する
		if !strings.Contains(inputStr, ".") {
			c.findAndAppendPackage(suggestions, inputStr)
			return suggestions
		}
	}

	if !strings.Contains(inputStr, ".") {
		c.findAndAppendPackage(suggestions, inputStr)
		return suggestions
	}

	c.findAndAppendMethod(suggestions, inputStr)

	pkgAndInput := buildPkgAndInput(inputStr)
	findedSuggestions := c.findSuggestions(pkgAndInput)
	suggestions = append(suggestions, findedSuggestions...)

	return suggestions
}

func (c *Completer) findSuggestions(pai pkgAndInput) []prompt.Suggest {
	var suggestions []prompt.Suggest
	if isAmpersandInclude {
		c.findAndAppendStruct(suggestions, pai)
		return suggestions
	}
	c.findAndAppendPackage(suggestions, pai.input)
	c.findAndAppendFunction(suggestions, pai)
	c.findAndAppendVariable(suggestions, pai)
	c.findAndAppendConstant(suggestions, pai)
	c.findAndAppendStruct(suggestions, pai)

	return suggestions
}

func (c *Completer) findAndAppendPackage(suggestions []prompt.Suggest, inputStr string) {
	for _, pkg := range c.candidates.pkgs {
		if strings.HasPrefix(string(pkg), inputStr) {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        addAmpersand() + string(pkg),
				DisplayText: string(pkg),
				Description: "Package",
			})
		}
	}
}

func (c *Completer) findAndAppendFunction(suggestions []prompt.Suggest, pai pkgAndInput) {
	if funcSets, ok := c.candidates.funcs[pkgName(pai.pkg)]; ok {
		for _, funcSet := range funcSets {
			if strings.HasPrefix(funcSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        pai.pkg + "." + funcSet.name + "()",
					DisplayText: funcSet.name + "()",
					Description: "Function: " + funcSet.description,
				})
			}
		}
	}
}

func (c *Completer) findAndAppendMethod(suggestions []prompt.Suggest, inputStr string) {
	for _, decl := range DeclVarRecords {
		if (decl.Name + ".") == inputStr {
			for _, methodSet := range c.candidates.methods[pkgName(decl.Pkg)] {
				if decl.Type == methodSet.receiverTypeName {
					suggestions = append(suggestions, prompt.Suggest{
						Text:        inputStr + methodSet.name + "()",
						DisplayText: methodSet.name + "()",
						Description: "Method: " + methodSet.description,
					})
				}
			}
		}
	}
}

func (c *Completer) findAndAppendVariable(suggestions []prompt.Suggest, pai pkgAndInput) {
	if varSets, ok := c.candidates.vars[pkgName(pai.pkg)]; ok {
		for _, varSet := range varSets {
			if strings.HasPrefix(varSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        pai.pkg + "." + varSet.name,
					DisplayText: varSet.name,
					Description: "Variable: " + varSet.description,
				})
			}
		}
	}
}

func (c *Completer) findAndAppendConstant(suggestions []prompt.Suggest, pai pkgAndInput) {
	if constSets, ok := c.candidates.consts[pkgName(pai.pkg)]; ok {
		for _, constSet := range constSets {
			if strings.HasPrefix(constSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        pai.pkg + "." + constSet.name,
					DisplayText: constSet.name,
					Description: "Constant: " + constSet.description,
				})
			}
		}
	}
}

func (c *Completer) findAndAppendStruct(suggestions []prompt.Suggest, pai pkgAndInput) {
	if structSets, ok := c.candidates.structs[pkgName(pai.pkg)]; ok {
		for _, structSet := range structSets {
			var field string
			if len(structSet.fields) > 0 {
				field += "{"
				for _, name := range structSet.fields {
					field += name + ": ,"
				}
				field = strings.TrimSuffix(field, ",") + "}"
			}
			if strings.HasPrefix(structSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        addAmpersand() + pai.pkg + "." + structSet.name + field,
					DisplayText: structSet.name,
					Description: "Struct: " + structSet.description,
				})
			}
		}
	}
}

type pkgAndInput struct {
	pkg   string
	input string
}

// {pkg名}. まで入力されている場合は、pkg名とその後の文字列を構造体にまとめる
func buildPkgAndInput(input string) pkgAndInput {
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

func findEqualAndSpacePos(input string) (int, bool) {
	equalPos := strings.LastIndex(input, "= ")
	if equalPos == -1 {
		return -1, false
	}
	return equalPos, true
}

var isAmpersandInclude bool

func addAmpersand() string {
	if isAmpersandInclude {
		return "&"
	}
	return ""
}
