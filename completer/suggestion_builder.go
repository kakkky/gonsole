package completer

import (
	"go/token"
	"strings"

	"github.com/kakkky/go-prompt"
)

type suggestionBuilder struct {
	prefixOperand token.Token
	input         input
}

type input struct {
	raw          string // 元の入力全体
	text         string //オペランド以降の入力全体
	basePart     string //セレクタ式のベース部分
	selectorPart string //セレクタ式のセレクタ部分
}

type suggestType int

const (
	suggestTypeUnknown suggestType = iota
	suggestTypePackage
	suggestTypeVariable
	suggestTypeFunction
	suggestTypeStruct
	suggestTypeMethod
	suggestTypeConstant
)

var and = token.AND.String()

func newSuggestionBuilder(rawInput string) *suggestionBuilder {
	// 変数宣言をしようとしている場合、"= "以降の部分を補完対象とする
	if pos, found := findEqualAndSpacePos(rawInput); found {
		startIdx := pos + 2 // "= "の長さは2なのでそれ以降
		rawInput = rawInput[startIdx:]
	}

	sb := &suggestionBuilder{
		input: input{
			raw:  rawInput,
			text: rawInput,
		},
	}

	switch {
	case strings.HasPrefix(rawInput, and):
		sb.prefixOperand = token.AND
		sb.input.text = strings.TrimPrefix(rawInput, and)
	}

	if strings.Contains(sb.input.text, ".") {
		parts := strings.SplitN(sb.input.text, ".", 2)
		sb.input.basePart = parts[0]
		sb.input.selectorPart = parts[1]
	}
	return sb
}

// "= "の位置を探し、見つかったらその位置とtrueを返す
func findEqualAndSpacePos(input string) (pos int, found bool) {
	equalAndSpace := "= "
	equalPos := strings.LastIndex(input, equalAndSpace)
	if equalPos == -1 {
		return -1, false
	}
	return equalPos, true
}

func (sb *suggestionBuilder) build(candidate string, suggestType suggestType, desctiption string, appendSuggestText ...string) prompt.Suggest {
	return prompt.Suggest{
		Text:        sb.buildSuggestText(candidate) + strings.Join(appendSuggestText, ""),
		DisplayText: candidate,
		Description: sb.buildSuggestDescription(suggestType, desctiption),
	}
}

func (sb *suggestionBuilder) buildSuggestText(candidateStr string) string {
	// 最長一致する prefix を見つける
	maxLen := min(len(sb.input.raw), len(candidateStr))
	matchLen := 0

	for i := 1; i <= maxLen; i++ {
		if strings.HasSuffix(sb.input.raw, candidateStr[:i]) {
			matchLen = i
		}
	}

	if matchLen > 0 {
		// 一致部分を除去して candidateStr に置き換え
		return strings.TrimSuffix(sb.input.raw, candidateStr[:matchLen]) + candidateStr
	}

	// 一致なしの場合
	return sb.input.raw + candidateStr
}

func (sb *suggestionBuilder) buildSuggestDescription(suggestType suggestType, description string) string {
	suggestTypeStr := convertSuggestTypeToString(suggestType)
	return suggestTypeStr + ": " + description
}

func convertSuggestTypeToString(suggestType suggestType) string {
	switch suggestType {
	case suggestTypePackage:
		return "Package"
	case suggestTypeVariable:
		return "Variable"
	case suggestTypeFunction:
		return "Function"
	case suggestTypeStruct:
		return "Struct"
	case suggestTypeMethod:
		return "Method"
	case suggestTypeConstant:
		return "Constant"
	default:
		return "Unknown"
	}
}

func (sb *suggestionBuilder) isSelector() bool {
	return strings.Contains(sb.input.text, ".")
}
