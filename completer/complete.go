package completer

import (
	"slices"
	"strings"
	"unicode"

	"github.com/kakkky/go-prompt"
	"github.com/kakkky/gonsole/decls"
)

type Completer struct {
	candidates *candidates
	declEntry  *decls.DeclEntry
}

func NewCompleter(candidates *candidates, declEntry *decls.DeclEntry) *Completer {
	return &Completer{
		candidates: candidates,
		declEntry:  declEntry,
	}
}

func (c *Completer) Complete(input prompt.Document) []prompt.Suggest {
	inputStr := input.Text
	// 先頭に&が含まれていたかどうか
	var isAndOperandInclude bool

	// & が含まれている場合は、& を除去してフラグをtrueにしておく
	// フラグは後続の処理で、& をつけるかどうかを決定するため
	if strings.Contains(inputStr, "&") {
		isAndOperandInclude = true
		inputStr = strings.ReplaceAll(inputStr, "&", "")
	}

	// 変数宣言の場合、"= "の後の文字列を補完対象とする
	if equalAndSpacePos, found := findEqualAndSpacePos(inputStr); found {
		inputStr = inputStr[equalAndSpacePos+2:]
	}

	// . を含んでいない場合は、パッケージ名の候補を返す
	// . を打つ前は基本的にパッケージ名を打とうとしていると想定している
	if !strings.Contains(inputStr, ".") {
		return c.findPackageSuggestions(inputStr, isAndOperandInclude)
	}

	// . 以降の文字列で補完候補を探す

	// 入力値からメソッドの候補があればそれを返す
	methodSuggests := c.findMethodSuggestions(inputStr)
	if len(methodSuggests) > 0 {
		return methodSuggests
	}

	// 補完候補の検索をしやすくするために、パッケージ名とその後の文字列を分ける
	pkgAndInput := buildPkgAndInput(inputStr, isAndOperandInclude)

	suggestions := c.findSuggestions(pkgAndInput)

	return suggestions
}

func (c *Completer) findSuggestions(pai pkgAndInput) []prompt.Suggest {
	// 先頭に&が含まれていたら構造体リテラルを入力しようとしていると想定
	if pai.isAndOperandInclude {
		return c.findStructSuggestions(pai)
	}
	functionSuggests := c.findFunctionSuggestions(pai)
	variableSuggests := c.findVariableSuggestions(pai)
	constantSuggets := c.findConstantSuggestions(pai)
	structSuggests := c.findStructSuggestions(pai)

	return slices.Concat(functionSuggests, variableSuggests, constantSuggets, structSuggests)
}

func (c *Completer) findPackageSuggestions(inputStr string, isAndOperandInclude bool) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0)
	for _, pkg := range c.candidates.pkgs {
		if strings.HasPrefix(string(pkg), inputStr) {
			var text string
			if isAndOperandInclude {
				text = "&" + string(pkg)
			} else {
				text = string(pkg)
			}
			suggestions = append(suggestions, prompt.Suggest{
				Text:        text,
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
			var text string
			if pai.isAndOperandInclude {
				text = "&" + pai.pkg + "." + funcSet.name + "()"
			} else {
				text = pai.pkg + "." + funcSet.name + "()"
			}
			if strings.HasPrefix(funcSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        text,
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
	// 重複を避けるためのマップ
	seenMethods := make(map[string]bool)

	// repl内で宣言された変数名エントリを回す
	for _, decl := range c.declEntry.Decls() {
		// 変数名.メソッド名の入力に対応（例: foo.Do）
		if strings.HasPrefix(inputStr, decl.Name()+".") {
			// 入力値からメソッド名部分を抽出
			methodPrefix := inputStr[len(decl.Name())+1:]
			for _, methodSet := range c.candidates.methods[pkgName(decl.Pkg())] {
				// メソッド名の前方一致フィルタ
				if methodPrefix == "" || strings.HasPrefix(methodSet.name, methodPrefix) {
					suggestions = c.findMethodSuggestionsFromVarRhsStructLit(
						suggestions, seenMethods, decl.Name()+".", decl, methodSet)
					suggestions = c.findMethodSuggestionsFromVarRhsDeclVar(
						suggestions, seenMethods, decl.Name()+".", decl, methodSet)
					suggestions = c.findMethodSuggestionsFromVarRhsFuncReturnValues(
						suggestions, seenMethods, decl.Name()+".", decl, methodSet)
					suggestions = c.findMethodSuggestionsFromVarRhsMethodReturnValues(
						suggestions, seenMethods, decl.Name()+".", decl, methodSet)
				}
			}
		}
	}

	// メソッドチェーン
	if strings.Contains(inputStr, ").") {
		return c.findMethodSuggestionsFromChain(suggestions, inputStr)
	}

	return suggestions
}

// 構造体リテラルから宣言された変数のメソッド候補を追加する
// その変数が構造体リテラルで宣言されたものである場合、レシーバの型が一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromVarRhsStructLit(
	suggestions []prompt.Suggest,
	seenMethods map[string]bool,
	inputStr string,
	decl decls.Decl,
	methodSet methodSet) []prompt.Suggest {
	if decl.Rhs().Struct().Type() == methodSet.receiverTypeName {
		// 重複チェック
		methodKey := inputStr + methodSet.name
		if seenMethods[methodKey] {
			return suggestions
		}
		seenMethods[methodKey] = true

		suggestions = append(suggestions, prompt.Suggest{
			Text:        inputStr + methodSet.name + "()",
			DisplayText: methodSet.name + "()",
			Description: "Method: " + methodSet.description,
		})
	}
	return suggestions
}

// 変数から宣言された変数のメソッド候補を追加する
// その変数が、ソースコード内で宣言された変数である場合、ソースコード内から得られた変数の補完候補をたどってパッケージ名と型情報を確定させ、
// その型に一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromVarRhsDeclVar(
	suggestions []prompt.Suggest,
	seenMethods map[string]bool,
	inputStr string,
	decl decls.Decl,
	methodSet methodSet) []prompt.Suggest {

	// 右辺の変数名を取得
	declRhsVarName := decl.Rhs().Var().Name()
	if declRhsVarName == "" {
		return suggestions
	}

	// 変数の補完候補を取得
	rhsVarSets, ok := c.candidates.vars[pkgName(decl.Pkg())]
	if !ok {
		return suggestions
	}

	// 変数の補完候補を回す
	for _, rhsVarSet := range rhsVarSets {
		if (decl.Pkg() == rhsVarSet.typePkgName) && // パッケージ名が一致
			(declRhsVarName == rhsVarSet.name) && // 変数名が一致
			(rhsVarSet.typeName == methodSet.receiverTypeName) { // 型名が一致
			// 重複チェック
			methodKey := inputStr + methodSet.name
			if seenMethods[methodKey] {
				continue
			}
			seenMethods[methodKey] = true

			suggestions = append(suggestions, prompt.Suggest{
				Text:        inputStr + methodSet.name + "()",
				DisplayText: methodSet.name + "()",
				Description: "Method: " + methodSet.description,
			})
		}
	}
	return suggestions
}

// 関数の戻り値から宣言された変数のメソッド候補を追加する
// その変数が、関数の戻り値である場合、ソースコード内から得られた関数の補完候補をたどって、その関数のパッケージ名と型情報を確定させ、
// その型に一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromVarRhsFuncReturnValues(
	suggestions []prompt.Suggest,
	seenMethods map[string]bool,
	inputStr string,
	decl decls.Decl,
	methodSet methodSet) []prompt.Suggest {

	// 右辺の関数名と戻り値の順序を取得
	declRhsFuncName := decl.Rhs().Func().Name()
	if declRhsFuncName == "" {
		return suggestions
	}

	declRhsFuncReturnVarOrder := decl.Rhs().Func().ReturnedOrder()
	rhsFuncSets, ok := c.candidates.funcs[pkgName(decl.Pkg())]
	if !ok {
		return suggestions
	}
	// 関数の補完候補を回す
	for _, rhsFuncSet := range rhsFuncSets {
		// 関数名が一致
		if declRhsFuncName == rhsFuncSet.name {
			// 関数の戻り値（複数）の型情報を確認
			for i, returnTypeName := range rhsFuncSet.returnTypeNames {
				if (i == declRhsFuncReturnVarOrder) && // 何個目の戻り値かが一致
					(returnTypeName == methodSet.receiverTypeName) { // 型名が一致
					// 重複チェック
					methodKey := inputStr + methodSet.name
					if seenMethods[methodKey] {
						continue
					}
					seenMethods[methodKey] = true

					suggestions = append(suggestions, prompt.Suggest{
						Text:        inputStr + methodSet.name + "()",
						DisplayText: methodSet.name + "()",
						Description: "Method: " + methodSet.description,
					})
				}
			}
		}
	}

	// メソッドの戻り値がinterfaceの場合、interfaceのメソッドを候補として表示する
	rhsInterfaceSets, ok := c.candidates.interfaces[pkgName(decl.Pkg())]
	if !ok {
		return suggestions
	}
	for _, rhsFuncSet := range rhsFuncSets {
		// 関数名が一致
		if declRhsFuncName == rhsFuncSet.name {
			for i, returnTypeName := range rhsFuncSet.returnTypeNames {
				if i != declRhsFuncReturnVarOrder {
					continue // 何個目の戻り値かが一致しない場合はスキップ
				}
				for _, rhsInterfaceSet := range rhsInterfaceSets {
					if returnTypeName == rhsInterfaceSet.name {
						for mi, method := range rhsInterfaceSet.methods {
							// 重複チェック
							methodKey := inputStr + method
							if seenMethods[methodKey] {
								continue
							}
							seenMethods[methodKey] = true

							suggestions = append(suggestions, prompt.Suggest{
								Text:        inputStr + method + "()",
								DisplayText: method + "()",
								Description: "Method: " + rhsInterfaceSet.descriptions[mi],
							})
						}
					}
				}
			}
		}
	}
	return suggestions
}

// メソッドの戻り値から宣言された変数のメソッド候補を追加する
// その変数が、メソッドの戻り値である場合、ソースコード内から得られたメソッドの補完候補をたどって、そのメソッドのパッケージ名と型情報を確定させ、
// その型に一致する場合は、補完候補として追加する
func (c *Completer) findMethodSuggestionsFromVarRhsMethodReturnValues(
	suggestions []prompt.Suggest,
	seenMethods map[string]bool,
	inputStr string,
	decl decls.Decl,
	methodSet methodSet) []prompt.Suggest {

	// 右辺のメソッド名と戻り値の順序を取得
	declRhsMethodName := decl.Rhs().Method().Name()
	if declRhsMethodName == "" {
		return suggestions
	}

	declRhsMethodReturnVarOrder := decl.Rhs().Method().ReturnedOrder()
	rhsMethodSets, ok := c.candidates.methods[pkgName(decl.Pkg())]
	if !ok {
		return suggestions
	}

	// メソッドの補完候補を回す
	for _, rhsMethodSet := range rhsMethodSets {
		// メソッド名が一致
		if declRhsMethodName == rhsMethodSet.name {
			// メソッドの戻り値（複数）の型情報を確認
			for i, returnTypeName := range rhsMethodSet.returnTypeNames {
				if (i == declRhsMethodReturnVarOrder) && // 何個目の戻り値かが一致
					(returnTypeName == methodSet.receiverTypeName) { // 型名が一致
					// 重複チェック
					methodKey := inputStr + methodSet.name
					if seenMethods[methodKey] {
						continue
					}
					seenMethods[methodKey] = true

					suggestions = append(suggestions, prompt.Suggest{
						Text:        inputStr + methodSet.name + "()",
						DisplayText: methodSet.name + "()",
						Description: "Method: " + methodSet.description,
					})
				}
			}
		}
	}

	// メソッドの戻り値がinterfaceの場合、interfaceのメソッドを候補として表示する
	rhsInterfaceSets, ok := c.candidates.interfaces[pkgName(decl.Pkg())]
	if !ok {
		return suggestions
	}
	for _, rhsMethodSet := range rhsMethodSets {
		// メソッド名が一致
		if declRhsMethodName == rhsMethodSet.name {
			// メソッドの戻り値（複数）の型情報を確認
			for i, returnTypeName := range rhsMethodSet.returnTypeNames {
				if i != declRhsMethodReturnVarOrder {
					continue // 何個目の戻り値かが一致しない場合はスキップ
				}
				for _, rhsInterfaceSet := range rhsInterfaceSets {
					if returnTypeName == rhsInterfaceSet.name {
						for mi, method := range rhsInterfaceSet.methods {
							// 重複チェック
							methodKey := inputStr + method
							if seenMethods[methodKey] {
								continue
							}
							seenMethods[methodKey] = true

							suggestions = append(suggestions, prompt.Suggest{
								Text:        inputStr + method + "()",
								DisplayText: method + "()",
								Description: "Method: " + rhsInterfaceSet.descriptions[mi],
							})
						}
					}
				}
			}
		}
	}
	return suggestions
}

func (c *Completer) findMethodSuggestionsFromChain(suggestions []prompt.Suggest, inputStr string) []prompt.Suggest {
	pkg, isRecv := c.getPkgAndIsRecv(inputStr)
	funcOrMethodName := getPrevFuncOrMethodName(inputStr)

	funcSetPtr := c.findFuncSetPtr(pkg, funcOrMethodName)
	methodSetPtr := c.findMethodSetPtr(pkg, funcOrMethodName, isRecv, inputStr)

	methodSets := c.candidates.methods[pkgName(pkg)]
	interfaceSets := c.candidates.interfaces[pkgName(pkg)]

	// 重複排除用マップ
	seen := make(map[string]struct{})
	for _, s := range suggestions {
		seen[s.Text] = struct{}{}
	}

	if funcSetPtr != nil && len(funcSetPtr.returnTypeNames) == 1 {
		for _, s := range c.findMethodSuggestionsFromTypeOrInterface(inputStr, funcSetPtr.returnTypeNames[0], methodSets, interfaceSets) {
			if _, ok := seen[s.Text]; !ok {
				suggestions = append(suggestions, s)
				seen[s.Text] = struct{}{}
			}
		}
	}
	if methodSetPtr != nil && len(methodSetPtr.returnTypeNames) == 1 {
		for _, s := range c.findMethodSuggestionsFromTypeOrInterface(inputStr, methodSetPtr.returnTypeNames[0], methodSets, interfaceSets) {
			if _, ok := seen[s.Text]; !ok {
				suggestions = append(suggestions, s)
				seen[s.Text] = struct{}{}
			}
		}
	}
	return suggestions
}

// パッケージ名とレシーバかどうかを取得
func (c *Completer) getPkgAndIsRecv(inputStr string) (string, bool) {
	firstDotIdx := strings.Index(inputStr, ".")
	pkgOrRecvName := inputStr[:firstDotIdx]
	if c.declEntry.IsRegisteredDecl(pkgOrRecvName) {
		for _, decl := range c.declEntry.Decls() {
			if decl.Name() == pkgOrRecvName {
				return decl.Pkg(), true
			}
		}
	}
	return pkgOrRecvName, false
}

// 直前の関数orメソッド名を取得
func getPrevFuncOrMethodName(inputStr string) string {
	lastOpeningParenthesisIdx := strings.LastIndex(inputStr, "(")
	if lastOpeningParenthesisIdx == -1 {
		return ""
	}
	secondLastDotIdx := strings.LastIndex(inputStr[:lastOpeningParenthesisIdx], ".")
	if secondLastDotIdx == -1 {
		return ""
	}
	return inputStr[secondLastDotIdx+1 : lastOpeningParenthesisIdx]
}

// 関数セットを取得
func (c *Completer) findFuncSetPtr(pkg, name string) *funcSet {
	funcSets := c.candidates.funcs[pkgName(pkg)]
	for i := range funcSets {
		if funcSets[i].name == name {
			return &funcSets[i]
		}
	}
	return nil
}

// メソッドセットを取得
func (c *Completer) findMethodSetPtr(pkg, name string, isRecv bool, inputStr string) *methodSet {
	if !isRecv && strings.Count(inputStr, "(") == 1 {
		return nil
	}
	methodSets := c.candidates.methods[pkgName(pkg)]
	for i := range methodSets {
		if methodSets[i].name == name {
			return &methodSets[i]
		}
	}
	return nil
}

// 指定型のメソッド・インターフェースメソッドを補完候補として返す
func (c *Completer) findMethodSuggestionsFromTypeOrInterface(inputStr, typeName string, methodSets []methodSet, interfaceSets []interfaceSet) []prompt.Suggest {
	var suggestions []prompt.Suggest
	var inputingMethodName string
	if dotIdx := strings.LastIndex(inputStr, "."); dotIdx != -1 && dotIdx+1 < len(inputStr) {
		inputingMethodName = inputStr[dotIdx+1:]
	}

	for _, method := range methodSets {
		if method.receiverTypeName == typeName && !isPrivate(method.name) {
			if strings.HasPrefix(method.name, inputingMethodName) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        inputStr + method.name + "()",
					DisplayText: method.name + "()",
					Description: "Method: " + method.description,
				})
			}
		}
	}
	for _, interfaceSet := range interfaceSets {
		if interfaceSet.name == typeName {
			for mi, method := range interfaceSet.methods {
				if isPrivate(method) {
					continue
				}
				if strings.HasPrefix(method, inputingMethodName) {
					desc := ""
					if mi < len(interfaceSet.descriptions) {
						desc = interfaceSet.descriptions[mi]
					}
					suggestions = append(suggestions, prompt.Suggest{
						Text:        inputStr + method + "()",
						DisplayText: method + "()",
						Description: "Method: " + desc,
					})
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
				if isPrivate(varSet.name) {
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
			if isPrivate(constSet.name) {
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
			if isPrivate(structSet.name) {
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
			var text string
			if pai.isAndOperandInclude {
				text = "&" + pai.pkg + "." + structSet.name + field
			} else {
				text = pai.pkg + "." + structSet.name + field
			}
			if strings.HasPrefix(structSet.name, pai.input) {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        text,
					DisplayText: structSet.name,
					Description: "Struct: " + structSet.description,
				})
			}
		}
	}
	return suggestions
}

// 補完候補の検索をしやすくするための構造体
type pkgAndInput struct {
	pkg                 string
	input               string
	isAndOperandInclude bool
}

// {pkg名}. まで入力されている場合は、pkg名とその後の文字列を構造体にまとめる
func buildPkgAndInput(input string, isAndOperandInclude bool) pkgAndInput {
	var pkgAndInput pkgAndInput
	if strings.Contains(input, ".") {
		parts := strings.SplitN(input, ".", 2)
		pkgAndInput.pkg = parts[0]
		if len(parts) == 2 {
			pkgAndInput.input = parts[1]
		}
	}
	pkgAndInput.isAndOperandInclude = isAndOperandInclude
	return pkgAndInput
}

// "= "の位置を探し、見つかったらその位置とtrueを返す
func findEqualAndSpacePos(input string) (int, bool) {
	equalPos := strings.LastIndex(input, "= ")
	if equalPos == -1 {
		return -1, false
	}
	return equalPos, true
}

// 非公開の関数や変数を非表示にする
func isPrivate(decl string) bool {
	return unicode.IsLower([]rune(decl)[0])
}
