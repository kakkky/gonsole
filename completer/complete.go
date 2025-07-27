package completer

import (
	"slices"
	"strings"
	"unicode"

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

	if strings.Contains(inputStr, "&") {
		isAmpersandInclude = true
		inputStr = strings.ReplaceAll(inputStr, "&", "")
	}

	if equalAndSpacePos, found := findEqualAndSpacePos(inputStr); found {
		inputStr = inputStr[equalAndSpacePos+2:]
	}

	if !strings.Contains(inputStr, ".") {
		return c.findPackageSuggestions(inputStr)
	}

	methodSuggests := c.findMethodSuggestions(inputStr)
	if len(methodSuggests) > 0 {
		return methodSuggests
	}

	pkgAndInput := buildPkgAndInput(inputStr)
	suggestions := c.findSuggestions(pkgAndInput)

	return suggestions
}

func (c *Completer) findSuggestions(pai pkgAndInput) []prompt.Suggest {
	if isAmpersandInclude {
		return c.findStructSuggestions(pai)
	}
	functionSuggests := c.findFunctionSuggestions(pai)
	variableSuggests := c.findVariableSuggestions(pai)
	constantSuggets := c.findConstantSuggestions(pai)
	structSuggests := c.findStructSuggestions(pai)

	return slices.Concat(functionSuggests, variableSuggests, constantSuggets, structSuggests)
}

func (c *Completer) findPackageSuggestions(inputStr string) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	for _, pkg := range c.candidates.pkgs {
		if strings.HasPrefix(string(pkg), inputStr) {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        addAmpersand() + string(pkg),
				DisplayText: string(pkg),
				Description: "Package",
			})
		}
	}
	return suggestions
}

func (c *Completer) findFunctionSuggestions(pai pkgAndInput) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if funcSets, ok := c.candidates.funcs[pkgName(pai.pkg)]; ok {
		for _, funcSet := range funcSets {
			if isPrivateDecl(funcSet.name) {
				continue
			}
			if strings.HasPrefix(funcSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        pai.pkg + "." + funcSet.name + "()",
					DisplayText: funcSet.name + "()",
					Description: "Function: " + funcSet.description,
				})
			}
		}
	}
	return suggestions
}

func (c *Completer) findMethodSuggestions(inputStr string) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	for _, decl := range DeclVarRecords {
		if (decl.Name + ".") == inputStr {
			for _, methodSet := range c.candidates.methods[pkgName(decl.Pkg)] {
				if decl.Rhs.Struct.Type == methodSet.receiverTypeName {
					if isPrivateDecl(methodSet.name) {
						continue
					}
					suggestions = append(suggestions, prompt.Suggest{
						Text:        inputStr + methodSet.name + "()",
						DisplayText: methodSet.name + "()",
						Description: "Method: " + methodSet.description,
					})
				}
				varPkgName := decl.Pkg

				declRhsVarName := decl.Rhs.Var.Name
				rhsVarSets, ok := c.candidates.vars[pkgName(varPkgName)]
				if ok {
					for _, rhsVarSet := range rhsVarSets {
						if varPkgName == rhsVarSet.typePkgName && declRhsVarName == rhsVarSet.name && rhsVarSet.typeName == methodSet.receiverTypeName {
							if isPrivateDecl(methodSet.name) {
								continue
							}
							suggestions = append(suggestions, prompt.Suggest{
								Text:        inputStr + methodSet.name + "()",
								DisplayText: methodSet.name + "()",
								Description: "Method: " + methodSet.description,
							})
						}
					}
				}

				declRhsFuncName := decl.Rhs.Func.Name
				declRhsFuncReturnVarOrder := decl.Rhs.Func.Order
				rhsFuncSets, ok := c.candidates.funcs[pkgName(varPkgName)]
				if ok {
					for _, rhsFuncSet := range rhsFuncSets {
						if declRhsFuncName == rhsFuncSet.name {
							for i, typeName := range rhsFuncSet.returnTypeName {
								if i == declRhsFuncReturnVarOrder {
									if typeName == methodSet.receiverTypeName {
										if isPrivateDecl(methodSet.name) {
											continue
										}
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
				}

				declRhsMethodName := decl.Rhs.Method.Name
				declRhsMethodReturnVarOrder := decl.Rhs.Method.Order
				rhsMethodSets, ok := c.candidates.methods[pkgName(varPkgName)]
				if ok {
					for _, rhsMethodSet := range rhsMethodSets {
						if declRhsMethodName == rhsMethodSet.name {
							for i, typeName := range rhsMethodSet.returnTypeName {
								if i == declRhsMethodReturnVarOrder {
									if typeName == methodSet.receiverTypeName {
										if isPrivateDecl(methodSet.name) {
											continue
										}
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
				}

			}
		}
	}
	return suggestions
}

func (c *Completer) findVariableSuggestions(pai pkgAndInput) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if varSets, ok := c.candidates.vars[pkgName(pai.pkg)]; ok {
		for _, varSet := range varSets {
			if strings.HasPrefix(varSet.name, pai.input) {
				if isPrivateDecl(varSet.name) {
					continue
				}
				suggestions = append(suggestions, prompt.Suggest{
					Text:        pai.pkg + "." + varSet.name,
					DisplayText: varSet.name,
					Description: "Variable: " + varSet.description,
				})
			}
		}
	}
	return suggestions
}

func (c *Completer) findConstantSuggestions(pai pkgAndInput) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if constSets, ok := c.candidates.consts[pkgName(pai.pkg)]; ok {
		for _, constSet := range constSets {
			if isPrivateDecl(constSet.name) {
				continue
			}
			if strings.HasPrefix(constSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        pai.pkg + "." + constSet.name,
					DisplayText: constSet.name,
					Description: "Constant: " + constSet.description,
				})
			}
		}
	}
	return suggestions
}

func (c *Completer) findStructSuggestions(pai pkgAndInput) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if structSets, ok := c.candidates.structs[pkgName(pai.pkg)]; ok {
		for _, structSet := range structSets {
			if isPrivateDecl(structSet.name) {
				continue
			}
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
	return suggestions
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

// 非公開の関数や変数を非表示にする
func isPrivateDecl(decl string) bool {
	return unicode.IsLower([]rune(decl)[0])
}
