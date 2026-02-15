package completer

import (
	"slices"
	"strings"
	"unicode"

	"github.com/kakkky/go-prompt"

	"github.com/kakkky/gonsole/declregistry"
	"github.com/kakkky/gonsole/symbols"
	"github.com/kakkky/gonsole/types"
)

// Completer は補完エンジンを担う
// go-promptのCompleterインターフェースを実装している
type Completer struct {
	symbolIndex  *symbols.SymbolIndex
	declRegistry *declregistry.DeclRegistry
}

// NewCompleter はCompleterのインスタンスを生成する
func NewCompleter(declRegistry *declregistry.DeclRegistry, symbolIndex *symbols.SymbolIndex) *Completer {
	return &Completer{
		symbolIndex:  symbolIndex,
		declRegistry: declRegistry,
	}
}

// Complete はgo-promptのCompleterインターフェースを実装するメソッドで、補完候補を返す
func (c *Completer) Complete(input prompt.Document) []prompt.Suggest {
	sb := newSuggestionBuilder(input.Text)

	if !sb.isSelector() {
		// TODO: repl内で宣言された変数の補完も出すようにしたい
		return c.findPackageSuggestions(sb)
	}

	suggestions := c.findSuggestions(sb)

	return suggestions
}

func (c *Completer) findSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	methodSuggests := c.findMethodSuggestions(sb)
	functionSuggests := c.findFunctionSuggestions(sb)
	variableSuggests := c.findVariableSuggestions(sb)
	constantSuggets := c.findConstantSuggestions(sb)
	structSuggests := c.findStructSuggestions(sb)
	definedTypeSuggests := c.findDefinedTypeSuggestions(sb)

	return slices.Concat(functionSuggests, methodSuggests, variableSuggests, constantSuggets, structSuggests, definedTypeSuggests)
}

func (c *Completer) findPackageSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	for _, pkg := range c.symbolIndex.Pkgs {
		if strings.HasPrefix(string(pkg), sb.input.text) {
			suggestions = append(suggestions, sb.build(string(pkg), suggestTypePackage, ""))
		}
	}
	return suggestions
}

func (c *Completer) findFunctionSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if funcSets, ok := c.symbolIndex.Funcs[types.PkgName(sb.input.basePart)]; ok {
		for _, funcSet := range funcSets {
			if strings.HasPrefix(string(funcSet.Name), sb.input.selectorPart) && !isPrivate(string(funcSet.Name)) {
				suggestions = append(suggestions, sb.build(string(funcSet.Name), suggestTypeFunction, funcSet.Description, "()"))
			}
		}
	}
	return suggestions
}
func (c *Completer) findMethodSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)

	// メソッドチェーンの場合は専用処理に移行
	if isMethodChain(sb.input.selectorPart) {
		return c.findMethodSuggestionsFromChain(suggestions, sb)
	}

	for _, decl := range c.declRegistry.Decls {
		if sb.input.basePart == string(decl.Name) {
			for _, methodSet := range c.symbolIndex.Methods[decl.TypePkgName] {
				if strings.HasPrefix(string(methodSet.Name), sb.input.selectorPart) && !isPrivate(string(methodSet.Name)) {
					if decl.TypeName == types.TypeName(methodSet.ReceiverTypeName) {
						suggestions = append(suggestions, sb.build(string(methodSet.Name), suggestTypeMethod, methodSet.Description, "()"))
					}
				}
			}
		}
		if len(suggestions) > 0 {
			return suggestions
		}
		for _, interfaceSet := range c.symbolIndex.Interfaces[decl.TypePkgName] {
			if !isPrivate(string(interfaceSet.Name)) {
				if decl.TypeName == types.TypeName(interfaceSet.Name) {
					for i, method := range interfaceSet.Methods {
						if strings.HasPrefix(string(method), sb.input.selectorPart) && !isPrivate(string(method)) {
							suggestions = append(suggestions, sb.build(string(method), suggestTypeMethod, interfaceSet.Descriptions[i], "()"))
						}
					}
				}
			}
		}
	}

	return suggestions
}

func isMethodChain(selectorPart string) bool {
	return strings.Contains(selectorPart, ").")
}

func (c *Completer) findMethodSuggestionsFromChain(suggestions []prompt.Suggest, sb *suggestionBuilder) []prompt.Suggest {
	// 入力例: "x.GetUser().GetProfile()." または "pkg.Func().Method()."
	// basePart = "x" or "pkg", selectorPart = "GetUser().GetProfile()." or "Func().Method()."

	selectorParts := strings.Split(sb.input.selectorPart, ".")
	for i, selectorPart := range selectorParts {
		idx := strings.Index(selectorPart, "(")
		if idx > 0 {
			selectorParts[i] = selectorPart[:idx]
			continue
		}
		selectorParts[i] = selectorPart

	}
	lastSelectorPart := selectorParts[len(selectorParts)-1]
	var lastReturElm symbols.ReturnSet

	if c.declRegistry.IsRegisteredDecl(types.DeclName(sb.input.basePart)) {
		// 最初の呼び出し要素がメソッド
		var firstRecvTypeName types.TypeName
		var firstRecvPkgName types.PkgName
		for _, decl := range c.declRegistry.Decls {
			if decl.Name == types.DeclName(sb.input.basePart) {
				firstRecvTypeName = decl.TypeName
				firstRecvPkgName = decl.TypePkgName
			}
		}
		var firstReturnElm symbols.ReturnSet
		for _, methodSet := range c.symbolIndex.Methods[firstRecvPkgName] {
			if strings.HasPrefix(string(methodSet.Name), selectorParts[0]) && types.TypeName(methodSet.ReceiverTypeName) == firstRecvTypeName && len(methodSet.Returns) == 1 {
				firstReturnElm = symbols.ReturnSet{
					TypeName:    methodSet.Returns[0].TypeName,
					TypePkgName: methodSet.Returns[0].TypePkgName,
				}
				break
			}
		}
		last := c.detectReturnElmFromMethodChainRecursive(sb, firstReturnElm.TypeName, firstReturnElm.TypePkgName, selectorParts[1:len(selectorParts)-1])
		if last == nil {
			return suggestions
		}
		lastReturElm = *last
	} else {
		// 最初の呼び出し要素が関数
		if funcSets, ok := c.symbolIndex.Funcs[types.PkgName(sb.input.basePart)]; ok {
			for _, funcSet := range funcSets {
				if string(funcSet.Name) == selectorParts[0] && len(funcSet.Returns) == 1 {
					firstReturnElm := funcSet.Returns[0]
					last := c.detectReturnElmFromMethodChainRecursive(sb, firstReturnElm.TypeName, firstReturnElm.TypePkgName, selectorParts[1:len(selectorParts)-1])
					if last == nil {
						return suggestions
					}
					lastReturElm = *last
					break
				}
			}
		}
	}

	for _, methodSet := range c.symbolIndex.Methods[lastReturElm.TypePkgName] {
		if strings.HasPrefix(string(methodSet.Name), lastSelectorPart) && !isPrivate(string(methodSet.Name)) && types.TypeName(methodSet.ReceiverTypeName) == lastReturElm.TypeName && len(methodSet.Returns) == 1 {
			suggestions = append(suggestions, sb.build(string(methodSet.Name), suggestTypeMethod, methodSet.Description, "()"))
		}
	}
	if len(suggestions) > 0 {
		return suggestions
	}
	for _, interfaceSet := range c.symbolIndex.Interfaces[lastReturElm.TypePkgName] {
		if lastReturElm.TypeName == types.TypeName(interfaceSet.Name) {
			for i, method := range interfaceSet.Methods {
				if strings.HasPrefix(string(method), lastSelectorPart) && !isPrivate(string(method)) {
					suggestions = append(suggestions, sb.build(string(method), suggestTypeMethod, interfaceSet.Descriptions[i], "()"))
				}
			}
		}
	}
	return suggestions
}

func (c *Completer) detectReturnElmFromMethodChainRecursive(sb *suggestionBuilder, prevBasePartTypeName types.TypeName, prevBasePartPkgName types.PkgName, selectorParts []string) *symbols.ReturnSet {
	if len(selectorParts) == 0 {
		return &symbols.ReturnSet{
			TypeName:    prevBasePartTypeName,
			TypePkgName: prevBasePartPkgName,
		}
	}
	currentSelectorPart := selectorParts[0]
	selectorParts = selectorParts[1:]

	for _, methodSet := range c.symbolIndex.Methods[prevBasePartPkgName] {
		if strings.HasPrefix(string(methodSet.Name), currentSelectorPart) && types.TypeName(methodSet.ReceiverTypeName) == prevBasePartTypeName && len(methodSet.Returns) == 1 {
			returnElm := methodSet.Returns[0]
			if len(selectorParts) == 0 {
				return &returnElm
			}
			nextReturnElm := c.detectReturnElmFromMethodChainRecursive(sb, returnElm.TypeName, returnElm.TypePkgName, selectorParts)
			if nextReturnElm != nil {
				return nextReturnElm
			}
		}
	}

	for _, interfaceSet := range c.symbolIndex.Interfaces[prevBasePartPkgName] {
		if types.TypeName(interfaceSet.Name) == prevBasePartTypeName {
			for _, method := range interfaceSet.Methods {
				if strings.HasPrefix(string(method), currentSelectorPart) {
					for _, methodSet := range c.symbolIndex.Methods[prevBasePartPkgName] {
						if string(methodSet.Name) == string(method) && len(methodSet.Returns) == 1 {
							returnElm := methodSet.Returns[0]
							if len(selectorParts) == 0 {
								return &returnElm
							}
							nextReturnElm := c.detectReturnElmFromMethodChainRecursive(sb, returnElm.TypeName, returnElm.TypePkgName, selectorParts)
							if nextReturnElm != nil {
								return nextReturnElm
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (c *Completer) findVariableSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if varSets, ok := c.symbolIndex.Vars[types.PkgName(sb.input.basePart)]; ok {
		for _, varSet := range varSets {
			if strings.HasPrefix(string(varSet.Name), sb.input.selectorPart) && !isPrivate(string(varSet.Name)) {
				suggestions = append(suggestions, sb.build(string(varSet.Name), suggestTypeVariable, varSet.Description))
			}
		}
	}
	return suggestions
}

func (c *Completer) findConstantSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if constSets, ok := c.symbolIndex.Consts[types.PkgName(sb.input.basePart)]; ok {
		for _, constSet := range constSets {
			if strings.HasPrefix(string(constSet.Name), sb.input.selectorPart) && !isPrivate(string(constSet.Name)) {
				suggestions = append(suggestions, sb.build(string(constSet.Name), suggestTypeConstant, constSet.Description))
			}
		}
	}
	return suggestions
}

func (c *Completer) findStructSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if structSets, ok := c.symbolIndex.Structs[types.PkgName(sb.input.basePart)]; ok {
		for _, structSet := range structSets {
			if strings.HasPrefix(string(structSet.Name), sb.input.selectorPart) && !isPrivate(string(structSet.Name)) {
				var compositeLit string
				if len(structSet.Fields) > 0 {
					compositeLit = compositeLitStr(structSet.Fields)
				}
				suggestions = append(suggestions, sb.build(string(structSet.Name), suggestTypeStruct, structSet.Description, "", compositeLit))
			}
		}
	}
	return suggestions
}

func compositeLitStr(fields []types.StructFieldName) string {
	var compositeLit string
	compositeLit += "{"
	for _, field := range fields {
		compositeLit += string(field) + ": ,"
	}
	compositeLit = strings.TrimSuffix(compositeLit, ",") + "}"
	return compositeLit
}

// 非公開の関数や変数を非表示にする
func isPrivate(input string) bool {
	return unicode.IsLower([]rune(input)[0])
}

func (c *Completer) findDefinedTypeSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	for _, definedTypeSet := range c.symbolIndex.DefinedTypes[types.PkgName(sb.input.basePart)] {
		if strings.HasPrefix(string(definedTypeSet.Name), sb.input.selectorPart) && !isPrivate(string(definedTypeSet.Name)) {
			suggestions = append(suggestions, sb.build(string(definedTypeSet.Name), suggestTypeDefinedType, definedTypeSet.Description, "()"))
		}
	}
	return suggestions
}
