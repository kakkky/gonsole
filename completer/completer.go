package completer

import (
	"slices"
	"strings"
	"unicode"

	"github.com/kakkky/go-prompt"

	"github.com/kakkky/gonsole/declregistry"
	"github.com/kakkky/gonsole/types"
)

// Completer は補完エンジンを担う
// go-promptのCompleterインターフェースを実装している
type Completer struct {
	candidates   *candidates
	declRegistry *declregistry.DeclRegistry
}

// NewCompleter はCompleterのインスタンスを生成する
func NewCompleter(declRegistry *declregistry.DeclRegistry) (*Completer, error) {
	candidates, err := NewCandidates(".")
	if err != nil {
		return nil, err
	}
	return &Completer{
		candidates:   candidates,
		declRegistry: declRegistry,
	}, nil
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

	return slices.Concat(functionSuggests, methodSuggests, variableSuggests, constantSuggets, structSuggests)
}

func (c *Completer) findPackageSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	for _, pkg := range c.candidates.Pkgs {
		if strings.HasPrefix(string(pkg), sb.input.text) {
			suggestions = append(suggestions, sb.build(string(pkg), suggestTypePackage, ""))
		}
	}
	return suggestions
}

func (c *Completer) findFunctionSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if funcSets, ok := c.candidates.Funcs[types.PkgName(sb.input.basePart)]; ok {
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

	// repl内で宣言された変数名エントリを回す
	for _, decl := range c.declRegistry.Decls() {
		if sb.input.basePart == string(decl.Name()) {
			for _, methodSet := range c.candidates.Methods[decl.RHS().PkgName()] {
				// メソッド名の前方一致フィルタと非公開フィルタ
				if strings.HasPrefix(string(methodSet.Name), sb.input.selectorPart) && !isPrivate(string(methodSet.Name)) {
					switch decl.RHS().Kind() {
					case declregistry.DeclRHSKindVar:
						suggestions = c.findMethodSuggestionsFromDeclRHSVar(suggestions, sb, decl, methodSet)
					case declregistry.DeclRHSKindStruct:
						suggestions = c.findMethodSuggestionsFromDeclRHSStruct(suggestions, sb, decl, methodSet)
					case declregistry.DeclRHSKindFunc:
						suggestions = c.findMethodSuggestionsFromDeclRHSFuncReturnValues(suggestions, sb, decl, methodSet)
					case declregistry.DeclRHSKindMethod:
						suggestions = c.findMethodSuggestionsFromDeclRHSMethodReturnValues(suggestions, sb, decl, methodSet)
					case declregistry.DeclRHSKindUnknown:
					}
				}
			}
			if len(suggestions) > 0 {
				return suggestions
			}
			for _, interfaceSet := range c.candidates.Interfaces[decl.RHS().PkgName()] {
				if !isPrivate(string(interfaceSet.Name)) {
					switch decl.RHS().Kind() {
					case declregistry.DeclRHSKindFunc:
						suggestions = c.findMethodSuggestionsFromDeclRHSFuncReturnInterface(suggestions, sb, decl, interfaceSet)
					case declregistry.DeclRHSKindMethod:
						suggestions = c.findMethodSuggestionsFromDeclRHSMethodReturnInterface(suggestions, sb, decl, interfaceSet)
					}
				}
			}
		}
	}

	return suggestions
}

// 構造体リテラルから宣言された変数のメソッド候補を追加する
// その変数が構造体リテラルで宣言されたものである場合、レシーバの型が一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromDeclRHSStruct(
	suggestions []prompt.Suggest,
	sb *suggestionBuilder,
	decl declregistry.Decl,
	methodSet methodSet) []prompt.Suggest {
	if string(decl.RHS().Name()) == string(methodSet.ReceiverTypeName) {
		suggestions = append(suggestions, sb.build(string(methodSet.Name), suggestTypeMethod, methodSet.Description, "()"))
	}
	return suggestions
}

// 変数から宣言された変数のメソッド候補を追加する
// その変数が、ソースコード内で宣言された変数である場合、ソースコード内から得られた変数の補完候補をたどってパッケージ名と型情報を確定させ、
// その型に一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromDeclRHSVar(
	suggestions []prompt.Suggest,
	sb *suggestionBuilder,
	decl declregistry.Decl,
	methodSet methodSet) []prompt.Suggest {

	declRHSVarName := decl.RHS().Name()
	declRHSVarPkgName := decl.RHS().PkgName()
	// 変数の補完候補を取得
	varSets, ok := c.candidates.Vars[declRHSVarPkgName]
	if !ok {
		return suggestions
	}

	for _, varSet := range varSets {
		if declRHSVarPkgName == varSet.PkgName && // パッケージ名が一致
			declRHSVarName == varSet.Name && // 変数名が一致
			varSet.TypeName == types.TypeName(methodSet.ReceiverTypeName) { // 型名が一致
			suggestions = append(suggestions, sb.build(string(methodSet.Name), suggestTypeMethod, methodSet.Description, "()"))
		}
	}
	return suggestions
}

// 関数の戻り値から宣言された変数のメソッド候補を追加する
// その変数が、関数の戻り値である場合、ソースコード内から得られた関数の補完候補をたどって、その関数のパッケージ名と型情報を確定させ、
// その型に一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromDeclRHSFuncReturnValues(
	suggestions []prompt.Suggest,
	sb *suggestionBuilder,
	decl declregistry.Decl,
	methodSet methodSet) []prompt.Suggest {

	// 右辺の関数名と戻り値の順序を取得
	declRHSFuncName := decl.RHS().Name()
	declRHSFuncPkgName := decl.RHS().PkgName()
	ok, returnedIdx := decl.IsReturnVal()
	if !ok {
		return suggestions
	}

	funcSets, ok := c.candidates.Funcs[declRHSFuncPkgName]
	if !ok {
		return suggestions
	}

	var returnElm *returnSet
	for _, funcSet := range funcSets {
		// 関数名が一致
		if declRHSFuncName == funcSet.Name {
			if returnedIdx >= len(funcSet.Returns) {
				continue // 戻り値の順序が範囲外の場合はスキップ
			}
			returnElm = &funcSet.Returns[returnedIdx]
			break
		}
	}
	if returnElm == nil {
		return suggestions
	}

	if returnElm.TypeName == types.TypeName(methodSet.ReceiverTypeName) {
		suggestions = append(suggestions, sb.build(string(methodSet.Name), suggestTypeMethod, methodSet.Description, "()"))
		return suggestions
	}

	return suggestions
}

// メソッドの戻り値から宣言された変数のメソッド候補を追加する
// その変数が、メソッドの戻り値である場合、ソースコード内から得られたメソッドの補完候補をたどって、そのメソッドのパッケージ名と型情報を確定させ、
// その型に一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromDeclRHSMethodReturnValues(
	suggestions []prompt.Suggest,
	sb *suggestionBuilder,
	decl declregistry.Decl,
	methodSet methodSet) []prompt.Suggest {

	declRHSMethodName := decl.RHS().Name()
	declRHSMethodPkgName := decl.RHS().PkgName()

	ok, returnedIdx := decl.IsReturnVal()
	if !ok {
		return suggestions
	}

	methodSets, ok := c.candidates.Methods[declRHSMethodPkgName]
	if !ok {
		return suggestions
	}

	// 1. メソッドを探して戻り値の型を取得
	var returnElm *returnSet
	for _, candidateMethodSet := range methodSets {
		if declRHSMethodName == candidateMethodSet.Name {
			if returnedIdx >= len(candidateMethodSet.Returns) {
				continue
			}
			returnElm = &candidateMethodSet.Returns[returnedIdx]
			break
		}
	}

	if returnElm == nil {
		return suggestions
	}

	// 2. 具体的な型(struct等)のレシーバとして一致するか確認
	if returnElm.TypeName == types.TypeName(methodSet.ReceiverTypeName) {
		suggestions = append(suggestions, sb.build(string(methodSet.Name), suggestTypeMethod, methodSet.Description, "()"))
		return suggestions
	}

	return suggestions
}

func isMethodChain(selectorPart string) bool {
	return strings.Contains(selectorPart, ").")
}

func (c *Completer) findMethodSuggestionsFromDeclRHSFuncReturnInterface(suggestions []prompt.Suggest, sb *suggestionBuilder, decl declregistry.Decl, interfaceSet interfaceSet) []prompt.Suggest {
	// 右辺の関数名と戻り値の順序を取得
	declRHSFuncName := decl.RHS().Name()
	declRHSFuncPkgName := decl.RHS().PkgName()
	ok, returnedIdx := decl.IsReturnVal()
	if !ok {
		return suggestions
	}

	funcSets, ok := c.candidates.Funcs[declRHSFuncPkgName]
	if !ok {
		return suggestions
	}

	var returnElm *returnSet
	for _, funcSet := range funcSets {
		// 関数名が一致
		if declRHSFuncName == funcSet.Name {
			if returnedIdx >= len(funcSet.Returns) {
				continue // 戻り値の順序が範囲外の場合はスキップ
			}
			returnElm = &funcSet.Returns[returnedIdx]
			break
		}
	}
	if returnElm == nil {
		return suggestions
	}

	if returnElm.TypeName == types.TypeName(interfaceSet.Name) {
		for i, method := range interfaceSet.Methods {
			if strings.HasPrefix(string(method), sb.input.selectorPart) && !isPrivate(string(method)) {
				suggestions = append(suggestions, sb.build(string(method), suggestTypeMethod, interfaceSet.Descriptions[i], "()"))
			}
		}
	}
	return suggestions
}

func (c *Completer) findMethodSuggestionsFromDeclRHSMethodReturnInterface(suggestions []prompt.Suggest, sb *suggestionBuilder, decl declregistry.Decl, interfaceSet interfaceSet) []prompt.Suggest {
	// 右辺のメソッド名と戻り値の順序を取得
	declRHSMethodName := decl.RHS().Name()
	declRHSMethodPkgName := decl.RHS().PkgName()

	ok, returnedIdx := decl.IsReturnVal()
	if !ok {
		return suggestions
	}

	methodSets, ok := c.candidates.Methods[declRHSMethodPkgName]
	if !ok {
		return suggestions
	}

	// 1. メソッドを探して戻り値の型を取得
	var returnElm *returnSet
	for _, candidateMethodSet := range methodSets {
		if declRHSMethodName == candidateMethodSet.Name {
			if returnedIdx >= len(candidateMethodSet.Returns) {
				continue
			}
			returnElm = &candidateMethodSet.Returns[returnedIdx]
			break
		}
	}

	if returnElm == nil {
		return suggestions
	}

	if returnElm.TypeName == types.TypeName(interfaceSet.Name) {
		for i, method := range interfaceSet.Methods {
			if strings.HasPrefix(string(method), sb.input.selectorPart) && !isPrivate(string(method)) {
				suggestions = append(suggestions, sb.build(string(method), suggestTypeMethod, interfaceSet.Descriptions[i], "()"))
			}
		}
	}
	return suggestions
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
	var lastReturElm returnSet

	if c.declRegistry.IsRegisteredDecl(types.DeclName(sb.input.basePart)) {
		// 最初の呼び出し要素がメソッド
		var firstRecvTypeName types.TypeName
		var firstRecvPkgName types.PkgName
		for _, decl := range c.declRegistry.Decls() {
			if decl.Name() == types.DeclName(sb.input.basePart) {
				switch decl.RHS().Kind() {
				case declregistry.DeclRHSKindVar:
					if varSets, ok := c.candidates.Vars[decl.RHS().PkgName()]; ok {
						for _, varSet := range varSets {
							if varSet.Name == decl.RHS().Name() {
								firstRecvTypeName = varSet.TypeName
								firstRecvPkgName = varSet.PkgName
								break
							}
						}
					}
				case declregistry.DeclRHSKindStruct:
					firstRecvTypeName = types.TypeName(decl.RHS().Name())
					firstRecvPkgName = decl.RHS().PkgName()
				case declregistry.DeclRHSKindFunc:
					ok, _ := decl.IsReturnVal()
					if !ok {
						break
					}
					funcSets, ok := c.candidates.Funcs[decl.RHS().PkgName()]
					if !ok {
						break
					}
					var returnElm *returnSet
					for _, funcSet := range funcSets {
						if funcSet.Name == decl.RHS().Name() {
							returnElm = &funcSet.Returns[0]
							break
						}
					}
					if returnElm == nil {
						break
					}
					firstRecvTypeName = returnElm.TypeName
					firstRecvPkgName = returnElm.PkgName
				case declregistry.DeclRHSKindMethod:
					declRHSMethodName := decl.RHS().Name()
					declRHSMethodPkgName := decl.RHS().PkgName()
					ok, _ := decl.IsReturnVal()
					if !ok {
						break
					}
					methodSets, ok := c.candidates.Methods[declRHSMethodPkgName]
					if !ok {
						break
					}

					var returnElm *returnSet
					for _, methodSet := range methodSets {
						if declRHSMethodName == methodSet.Name && len(methodSet.Returns) == 1 {
							returnElm = &methodSet.Returns[0]
							break
						}
					}
					if returnElm == nil {
						break
					}

					firstRecvTypeName = returnElm.TypeName
					firstRecvPkgName = returnElm.PkgName

				}
			}
		}
		var firstReturnElm returnSet
		for _, methodSet := range c.candidates.Methods[firstRecvPkgName] {
			if strings.HasPrefix(string(methodSet.Name), selectorParts[0]) && types.TypeName(methodSet.ReceiverTypeName) == firstRecvTypeName && len(methodSet.Returns) == 1 {
				firstReturnElm = returnSet{
					TypeName: methodSet.Returns[0].TypeName,
					PkgName:  methodSet.Returns[0].PkgName,
				}
				break
			}
		}
		last := c.detectReturnElmFromMethodChainRecursive(sb, firstReturnElm.TypeName, firstReturnElm.PkgName, selectorParts[1:len(selectorParts)-1])
		if last == nil {
			return suggestions
		}
		lastReturElm = *last
	} else {
		// 最初の呼び出し要素が関数
		if funcSets, ok := c.candidates.Funcs[types.PkgName(sb.input.basePart)]; ok {
			for _, funcSet := range funcSets {
				if string(funcSet.Name) == selectorParts[0] && len(funcSet.Returns) == 1 {
					firstReturnElm := funcSet.Returns[0]
					last := c.detectReturnElmFromMethodChainRecursive(sb, firstReturnElm.TypeName, firstReturnElm.PkgName, selectorParts[1:len(selectorParts)-1])
					if last == nil {
						return suggestions
					}
					lastReturElm = *last
					break
				}
			}
		}
	}

	for _, methodSet := range c.candidates.Methods[lastReturElm.PkgName] {
		if strings.HasPrefix(string(methodSet.Name), lastSelectorPart) && !isPrivate(string(methodSet.Name)) && types.TypeName(methodSet.ReceiverTypeName) == lastReturElm.TypeName && len(methodSet.Returns) == 1 {
			suggestions = append(suggestions, sb.build(string(methodSet.Name), suggestTypeMethod, methodSet.Description, "()"))
		}
	}
	if len(suggestions) > 0 {
		return suggestions
	}
	for _, interfaceSet := range c.candidates.Interfaces[lastReturElm.PkgName] {
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

func (c *Completer) detectReturnElmFromMethodChainRecursive(sb *suggestionBuilder, prevBasePartTypeName types.TypeName, prevBasePartPkgName types.PkgName, selectorParts []string) *returnSet {
	if len(selectorParts) == 0 {
		return &returnSet{
			TypeName: prevBasePartTypeName,
			PkgName:  prevBasePartPkgName,
		}
	}
	currentSelectorPart := selectorParts[0]
	selectorParts = selectorParts[1:]

	for _, methodSet := range c.candidates.Methods[prevBasePartPkgName] {
		if strings.HasPrefix(string(methodSet.Name), currentSelectorPart) && types.TypeName(methodSet.ReceiverTypeName) == prevBasePartTypeName && len(methodSet.Returns) == 1 {
			returnElm := methodSet.Returns[0]
			if len(selectorParts) == 0 {
				return &returnElm
			}
			nextReturnElm := c.detectReturnElmFromMethodChainRecursive(sb, returnElm.TypeName, returnElm.PkgName, selectorParts)
			if nextReturnElm != nil {
				return nextReturnElm
			}
		}
	}

	for _, interfaceSet := range c.candidates.Interfaces[prevBasePartPkgName] {
		if types.TypeName(interfaceSet.Name) == prevBasePartTypeName {
			for _, method := range interfaceSet.Methods {
				if strings.HasPrefix(string(method), currentSelectorPart) {
					for _, methodSet := range c.candidates.Methods[prevBasePartPkgName] {
						if string(methodSet.Name) == string(method) && len(methodSet.Returns) == 1 {
							returnElm := methodSet.Returns[0]
							if len(selectorParts) == 0 {
								return &returnElm
							}
							nextReturnElm := c.detectReturnElmFromMethodChainRecursive(sb, returnElm.TypeName, returnElm.PkgName, selectorParts)
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
	if varSets, ok := c.candidates.Vars[types.PkgName(sb.input.basePart)]; ok {
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
	if constSets, ok := c.candidates.Consts[types.PkgName(sb.input.basePart)]; ok {
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
	if structSets, ok := c.candidates.Structs[types.PkgName(sb.input.basePart)]; ok {
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
