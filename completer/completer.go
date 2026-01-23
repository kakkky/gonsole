package completer

import (
	"slices"
	"strings"
	"unicode"

	"github.com/kakkky/go-prompt"

	"github.com/kakkky/gonsole/decl_registry"
	"github.com/kakkky/gonsole/types"
)

type Completer struct {
	candidates   *candidates
	declRegistry *decl_registry.DeclRegistry
}

func NewCompleter(declRegistry *decl_registry.DeclRegistry) (*Completer, error) {
	candidates, err := newCandidates(".")
	if err != nil {
		return nil, err
	}
	return &Completer{
		candidates:   candidates,
		declRegistry: declRegistry,
	}, nil
}

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
	for _, pkg := range c.candidates.pkgs {
		if strings.HasPrefix(string(pkg), sb.input.text) {
			suggestions = append(suggestions, sb.build(string(pkg), suggestTypePackage, ""))
		}
	}
	return suggestions
}

func (c *Completer) findFunctionSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if funcSets, ok := c.candidates.funcs[types.PkgName(sb.input.basePart)]; ok {
		for _, funcSet := range funcSets {
			if strings.HasPrefix(string(funcSet.name), sb.input.selectorPart) && !isPrivate(string(funcSet.name)) {
				suggestions = append(suggestions, sb.build(string(funcSet.name), suggestTypeFunction, funcSet.description, "()"))
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
			for _, methodSet := range c.candidates.methods[decl.Rhs().PkgName()] {
				// メソッド名の前方一致フィルタと非公開フィルタ
				if strings.HasPrefix(string(methodSet.name), sb.input.selectorPart) && !isPrivate(string(methodSet.name)) {
					switch decl.Rhs().Kind() {
					case decl_registry.DeclRhsKindVar:
						suggestions = c.findMethodSuggestionsFromDeclRhsVar(suggestions, sb, decl, methodSet)
					case decl_registry.DeclRhsKindStruct:
						suggestions = c.findMethodSuggestionsFromDeclRhsStruct(suggestions, sb, decl, methodSet)
					case decl_registry.DeclRhsKindFunc:
						suggestions = c.findMethodSuggestionsFromDeclRhsFuncReturnValues(suggestions, sb, decl, methodSet)
					case decl_registry.DeclRhsKindMethod:
						suggestions = c.findMethodSuggestionsFromDeclRhsMethodReturnValues(suggestions, sb, decl, methodSet)
					case decl_registry.DeclRhsKindUnknown:
					}
				}
			}
			if len(suggestions) > 0 {
				return suggestions
			}
			for _, interfaceSet := range c.candidates.interfaces[decl.Rhs().PkgName()] {
				if !isPrivate(string(interfaceSet.name)) {
					switch decl.Rhs().Kind() {
					case decl_registry.DeclRhsKindFunc:
						suggestions = c.findMethodSuggestionsFromDeclRhsFuncReturnInterface(suggestions, sb, decl, interfaceSet)
					case decl_registry.DeclRhsKindMethod:
						suggestions = c.findMethodSuggestionsFromDeclRhsMethodReturnInterface(suggestions, sb, decl, interfaceSet)
					}
				}
			}
		}
	}

	return suggestions
}

// 構造体リテラルから宣言された変数のメソッド候補を追加する
// その変数が構造体リテラルで宣言されたものである場合、レシーバの型が一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromDeclRhsStruct(
	suggestions []prompt.Suggest,
	sb *suggestionBuilder,
	decl decl_registry.Decl,
	methodSet methodSet) []prompt.Suggest {
	if string(decl.Rhs().Name()) == string(methodSet.receiverTypeName) {
		suggestions = append(suggestions, sb.build(string(methodSet.name), suggestTypeMethod, methodSet.description, "()"))
	}
	return suggestions
}

// 変数から宣言された変数のメソッド候補を追加する
// その変数が、ソースコード内で宣言された変数である場合、ソースコード内から得られた変数の補完候補をたどってパッケージ名と型情報を確定させ、
// その型に一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromDeclRhsVar(
	suggestions []prompt.Suggest,
	sb *suggestionBuilder,
	decl decl_registry.Decl,
	methodSet methodSet) []prompt.Suggest {

	declRhsVarName := decl.Rhs().Name()
	declRhsVarPkgName := decl.Rhs().PkgName()
	// 変数の補完候補を取得
	varSets, ok := c.candidates.vars[declRhsVarPkgName]
	if !ok {
		return suggestions
	}

	for _, varSet := range varSets {
		if declRhsVarPkgName == varSet.pkgName && // パッケージ名が一致
			declRhsVarName == varSet.name && // 変数名が一致
			varSet.typeName == types.TypeName(methodSet.receiverTypeName) { // 型名が一致
			suggestions = append(suggestions, sb.build(string(methodSet.name), suggestTypeMethod, methodSet.description, "()"))
		}
	}
	return suggestions
}

// 関数の戻り値から宣言された変数のメソッド候補を追加する
// その変数が、関数の戻り値である場合、ソースコード内から得られた関数の補完候補をたどって、その関数のパッケージ名と型情報を確定させ、
// その型に一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromDeclRhsFuncReturnValues(
	suggestions []prompt.Suggest,
	sb *suggestionBuilder,
	decl decl_registry.Decl,
	methodSet methodSet) []prompt.Suggest {

	// 右辺の関数名と戻り値の順序を取得
	declRhsFuncName := decl.Rhs().Name()
	declRhsFuncPkgName := decl.Rhs().PkgName()
	ok, returnedIdx := decl.IsReturnVal()
	if !ok {
		return suggestions
	}

	funcSets, ok := c.candidates.funcs[declRhsFuncPkgName]
	if !ok {
		return suggestions
	}

	var returnElm *returnSet
	for _, funcSet := range funcSets {
		// 関数名が一致
		if declRhsFuncName == funcSet.name {
			if returnedIdx >= len(funcSet.returns) {
				continue // 戻り値の順序が範囲外の場合はスキップ
			}
			returnElm = &funcSet.returns[returnedIdx]
			break
		}
	}
	if returnElm == nil {
		return suggestions
	}

	if returnElm.typeName == types.TypeName(methodSet.receiverTypeName) {
		suggestions = append(suggestions, sb.build(string(methodSet.name), suggestTypeMethod, methodSet.description, "()"))
		return suggestions
	}

	return suggestions
}

// メソッドの戻り値から宣言された変数のメソッド候補を追加する
// その変数が、メソッドの戻り値である場合、ソースコード内から得られたメソッドの補完候補をたどって、そのメソッドのパッケージ名と型情報を確定させ、
// その型に一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromDeclRhsMethodReturnValues(
	suggestions []prompt.Suggest,
	sb *suggestionBuilder,
	decl decl_registry.Decl,
	methodSet methodSet) []prompt.Suggest {

	declRhsMethodName := decl.Rhs().Name()
	declRhsMethodPkgName := decl.Rhs().PkgName()

	ok, returnedIdx := decl.IsReturnVal()
	if !ok {
		return suggestions
	}

	methodSets, ok := c.candidates.methods[declRhsMethodPkgName]
	if !ok {
		return suggestions
	}

	// 1. メソッドを探して戻り値の型を取得
	var returnElm *returnSet
	for _, candidateMethodSet := range methodSets {
		if declRhsMethodName == candidateMethodSet.name {
			if returnedIdx >= len(candidateMethodSet.returns) {
				continue
			}
			returnElm = &candidateMethodSet.returns[returnedIdx]
			break
		}
	}

	if returnElm == nil {
		return suggestions
	}

	// 2. 具体的な型(struct等)のレシーバとして一致するか確認
	if returnElm.typeName == types.TypeName(methodSet.receiverTypeName) {
		suggestions = append(suggestions, sb.build(string(methodSet.name), suggestTypeMethod, methodSet.description, "()"))
		return suggestions
	}

	return suggestions
}

func isMethodChain(selectorPart string) bool {
	return strings.Contains(selectorPart, ").")
}

func (c *Completer) findMethodSuggestionsFromDeclRhsFuncReturnInterface(suggestions []prompt.Suggest, sb *suggestionBuilder, decl decl_registry.Decl, interfaceSet interfaceSet) []prompt.Suggest {
	// 右辺の関数名と戻り値の順序を取得
	declRhsFuncName := decl.Rhs().Name()
	declRhsFuncPkgName := decl.Rhs().PkgName()
	ok, returnedIdx := decl.IsReturnVal()
	if !ok {
		return suggestions
	}

	funcSets, ok := c.candidates.funcs[declRhsFuncPkgName]
	if !ok {
		return suggestions
	}

	var returnElm *returnSet
	for _, funcSet := range funcSets {
		// 関数名が一致
		if declRhsFuncName == funcSet.name {
			if returnedIdx >= len(funcSet.returns) {
				continue // 戻り値の順序が範囲外の場合はスキップ
			}
			returnElm = &funcSet.returns[returnedIdx]
			break
		}
	}
	if returnElm == nil {
		return suggestions
	}

	if returnElm.typeName == types.TypeName(interfaceSet.name) {
		for i, method := range interfaceSet.methods {
			if strings.HasPrefix(string(method), sb.input.selectorPart) && !isPrivate(string(method)) {
				suggestions = append(suggestions, sb.build(string(method), suggestTypeMethod, interfaceSet.descriptions[i], "()"))
			}
		}
	}
	return suggestions
}

func (c *Completer) findMethodSuggestionsFromDeclRhsMethodReturnInterface(suggestions []prompt.Suggest, sb *suggestionBuilder, decl decl_registry.Decl, interfaceSet interfaceSet) []prompt.Suggest {
	// 右辺のメソッド名と戻り値の順序を取得
	declRhsMethodName := decl.Rhs().Name()
	declRhsMethodPkgName := decl.Rhs().PkgName()

	ok, returnedIdx := decl.IsReturnVal()
	if !ok {
		return suggestions
	}

	methodSets, ok := c.candidates.methods[declRhsMethodPkgName]
	if !ok {
		return suggestions
	}

	// 1. メソッドを探して戻り値の型を取得
	var returnElm *returnSet
	for _, candidateMethodSet := range methodSets {
		if declRhsMethodName == candidateMethodSet.name {
			if returnedIdx >= len(candidateMethodSet.returns) {
				continue
			}
			returnElm = &candidateMethodSet.returns[returnedIdx]
			break
		}
	}

	if returnElm == nil {
		return suggestions
	}

	if returnElm.typeName == types.TypeName(interfaceSet.name) {
		for i, method := range interfaceSet.methods {
			if strings.HasPrefix(string(method), sb.input.selectorPart) && !isPrivate(string(method)) {
				suggestions = append(suggestions, sb.build(string(method), suggestTypeMethod, interfaceSet.descriptions[i], "()"))
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
				switch decl.Rhs().Kind() {
				case decl_registry.DeclRhsKindVar:
					if varSets, ok := c.candidates.vars[decl.Rhs().PkgName()]; ok {
						for _, varSet := range varSets {
							if varSet.name == decl.Rhs().Name() {
								firstRecvTypeName = varSet.typeName
								firstRecvPkgName = varSet.pkgName
								break
							}
						}
					}
				case decl_registry.DeclRhsKindStruct:
					firstRecvTypeName = types.TypeName(decl.Rhs().Name())
					firstRecvPkgName = decl.Rhs().PkgName()
				case decl_registry.DeclRhsKindFunc:
					ok, _ := decl.IsReturnVal()
					if !ok {
						break
					}
					funcSets, ok := c.candidates.funcs[decl.Rhs().PkgName()]
					if !ok {
						break
					}
					var returnElm *returnSet
					for _, funcSet := range funcSets {
						if funcSet.name == decl.Rhs().Name() {
							returnElm = &funcSet.returns[0]
							break
						}
					}
					if returnElm == nil {
						break
					}
					firstRecvTypeName = returnElm.typeName
					firstRecvPkgName = returnElm.pkgName
				case decl_registry.DeclRhsKindMethod:
					declRhsMethodName := decl.Rhs().Name()
					declRhsMethodPkgName := decl.Rhs().PkgName()
					ok, _ := decl.IsReturnVal()
					if !ok {
						break
					}
					methodSets, ok := c.candidates.methods[declRhsMethodPkgName]
					if !ok {
						break
					}

					var returnElm *returnSet
					for _, methodSet := range methodSets {
						if declRhsMethodName == methodSet.name && len(methodSet.returns) == 1 {
							returnElm = &methodSet.returns[0]
							break
						}
					}
					if returnElm == nil {
						break
					}

					firstRecvTypeName = returnElm.typeName
					firstRecvPkgName = returnElm.pkgName

				}
			}
		}
		var firstReturnElm returnSet
		for _, methodSet := range c.candidates.methods[firstRecvPkgName] {
			if strings.HasPrefix(string(methodSet.name), selectorParts[0]) && types.TypeName(methodSet.receiverTypeName) == firstRecvTypeName && len(methodSet.returns) == 1 {
				firstReturnElm = returnSet{
					typeName: methodSet.returns[0].typeName,
					pkgName:  methodSet.returns[0].pkgName,
				}
				break
			}
		}
		last := c.detectReturnElmFromMethodChainRecursive(sb, firstReturnElm.typeName, firstReturnElm.pkgName, selectorParts[1:len(selectorParts)-1])
		if last == nil {
			return suggestions
		}
		lastReturElm = *last
	} else {
		// 最初の呼び出し要素が関数
		if funcSets, ok := c.candidates.funcs[types.PkgName(sb.input.basePart)]; ok {
			for _, funcSet := range funcSets {
				if string(funcSet.name) == selectorParts[0] && len(funcSet.returns) == 1 {
					firstReturnElm := funcSet.returns[0]
					last := c.detectReturnElmFromMethodChainRecursive(sb, firstReturnElm.typeName, firstReturnElm.pkgName, selectorParts[1:len(selectorParts)-1])
					if last == nil {
						return suggestions
					}
					lastReturElm = *last
					break
				}
			}
		}
	}

	for _, methodSet := range c.candidates.methods[lastReturElm.pkgName] {
		if strings.HasPrefix(string(methodSet.name), lastSelectorPart) && !isPrivate(string(methodSet.name)) && types.TypeName(methodSet.receiverTypeName) == lastReturElm.typeName && len(methodSet.returns) == 1 {
			suggestions = append(suggestions, sb.build(string(methodSet.name), suggestTypeMethod, methodSet.description, "()"))
		}
	}
	if len(suggestions) > 0 {
		return suggestions
	}
	for _, interfaceSet := range c.candidates.interfaces[lastReturElm.pkgName] {
		if lastReturElm.typeName == types.TypeName(interfaceSet.name) {
			for i, method := range interfaceSet.methods {
				if strings.HasPrefix(string(method), lastSelectorPart) && !isPrivate(string(method)) {
					suggestions = append(suggestions, sb.build(string(method), suggestTypeMethod, interfaceSet.descriptions[i], "()"))
				}
			}
		}
	}
	return suggestions
}

func (c *Completer) detectReturnElmFromMethodChainRecursive(sb *suggestionBuilder, prevBasePartTypeName types.TypeName, prevBasePartPkgName types.PkgName, selectorParts []string) *returnSet {
	if len(selectorParts) == 0 {
		return &returnSet{
			typeName: prevBasePartTypeName,
			pkgName:  prevBasePartPkgName,
		}
	}
	currentSelectorPart := selectorParts[0]
	selectorParts = selectorParts[1:]

	for _, methodSet := range c.candidates.methods[prevBasePartPkgName] {
		if strings.HasPrefix(string(methodSet.name), currentSelectorPart) && types.TypeName(methodSet.receiverTypeName) == prevBasePartTypeName && len(methodSet.returns) == 1 {
			returnElm := methodSet.returns[0]
			if len(selectorParts) == 0 {
				return &returnElm
			}
			nextReturnElm := c.detectReturnElmFromMethodChainRecursive(sb, returnElm.typeName, returnElm.pkgName, selectorParts)
			if nextReturnElm != nil {
				return nextReturnElm
			}
		}
	}

	for _, interfaceSet := range c.candidates.interfaces[prevBasePartPkgName] {
		if types.TypeName(interfaceSet.name) == prevBasePartTypeName {
			for _, method := range interfaceSet.methods {
				if strings.HasPrefix(string(method), currentSelectorPart) {
					for _, methodSet := range c.candidates.methods[prevBasePartPkgName] {
						if string(methodSet.name) == string(method) && len(methodSet.returns) == 1 {
							returnElm := methodSet.returns[0]
							if len(selectorParts) == 0 {
								return &returnElm
							}
							nextReturnElm := c.detectReturnElmFromMethodChainRecursive(sb, returnElm.typeName, returnElm.pkgName, selectorParts)
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
	if varSets, ok := c.candidates.vars[types.PkgName(sb.input.basePart)]; ok {
		for _, varSet := range varSets {
			if strings.HasPrefix(string(varSet.name), sb.input.selectorPart) && !isPrivate(string(varSet.name)) {
				suggestions = append(suggestions, sb.build(string(varSet.name), suggestTypeVariable, varSet.description))
			}
		}
	}
	return suggestions
}

func (c *Completer) findConstantSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if constSets, ok := c.candidates.consts[types.PkgName(sb.input.basePart)]; ok {
		for _, constSet := range constSets {
			if strings.HasPrefix(string(constSet.name), sb.input.selectorPart) && !isPrivate(string(constSet.name)) {
				suggestions = append(suggestions, sb.build(string(constSet.name), suggestTypeConstant, constSet.description))
			}
		}
	}
	return suggestions
}

func (c *Completer) findStructSuggestions(sb *suggestionBuilder) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	if structSets, ok := c.candidates.structs[types.PkgName(sb.input.basePart)]; ok {
		for _, structSet := range structSets {
			if strings.HasPrefix(string(structSet.name), sb.input.selectorPart) && !isPrivate(string(structSet.name)) {
				var compositeLit string
				if len(structSet.fields) > 0 {
					compositeLit = compositeLitStr(structSet.fields)
				}
				suggestions = append(suggestions, sb.build(string(structSet.name), suggestTypeStruct, structSet.description, "", compositeLit))
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
