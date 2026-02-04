package executor

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/kakkky/go-prompt"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
)

//go:generate mockgen -package=executor -source=./import_path_resolver.go -destination=./import_path_resolver_mock.go
type importPathResolver interface {
	resolve(pkgName types.PkgName) (importPath types.ImportPath, err error)
}

type defaultImportPathResolver struct {
	commander
}

func newDefaultImportPathResolver(cmd commander) *defaultImportPathResolver {
	return &defaultImportPathResolver{
		commander: cmd,
	}
}

func (dipr *defaultImportPathResolver) resolve(pkgName types.PkgName) (types.ImportPath, error) {
	var importPathCandidates []types.ImportPath

	if stdpkgImportPaths, ok := stdPkgImportPathMap[pkgName]; ok {
		importPathCandidates = append(importPathCandidates, stdpkgImportPaths...)
	}

	cmdOut, err := dipr.execGoListAll()
	if err != nil {
		return "", errs.NewInternalError("failed to resolve import path").Wrap(err)
	}

	allImportPaths := strings.Split(string(cmdOut), "\n")
	for _, importPath := range allImportPaths {
		if importPath == "" {
			continue
		}
		if types.PkgName(path.Base(importPath)) == pkgName {
			quoted := fmt.Sprintf(`"%s"`, importPath)
			importPathCandidates = append(importPathCandidates, types.ImportPath(quoted))
		}
	}

	if len(importPathCandidates) == 1 {
		return importPathCandidates[0], nil
	}

	// 複数候補がある場合はユーザーに選択させる
	selectedImportPath, err := selectImportPathRepl(importPathCandidates)
	if err != nil {
		return "", err
	}
	return selectedImportPath, nil
}

func selectImportPathRepl(importPathCandidates []types.ImportPath) (types.ImportPath, error) {
	toBlue := func(s string) string {
		colorBlue := "\033[94m"
		colorReset := "\033[0m"
		return fmt.Sprintf("%s%s%s", colorBlue, s, colorReset)
	}
	completer := func(d prompt.Document) []prompt.Suggest {
		suggests := make([]prompt.Suggest, len(importPathCandidates))
		for i, importPath := range importPathCandidates {
			suggests[i] = prompt.Suggest{Text: string(importPath)}
		}
		return suggests
	}

	fmt.Println(toBlue("\nMultiple import candidates found.\n\nUse Tab key to select import path.\n\n"))
	for _, importPathCandidate := range importPathCandidates {
		fmt.Printf(toBlue("- %s\n"), importPathCandidate)
	}
	fmt.Print(toBlue("\n>>> "))
	selectedImportPath := prompt.Input(
		"",
		completer,
		prompt.OptionShowCompletionAtStart(),
		prompt.OptionPreviewSuggestionTextColor(prompt.Turquoise),
		prompt.OptionInputTextColor(prompt.Turquoise),
	)
	if selectedImportPath == "" {
		return "", errs.NewBadInputError("no import path selected")
	}
	if !slices.Contains(importPathCandidates, types.ImportPath(selectedImportPath)) {
		return "", errs.NewBadInputError("invalid import path selected")
	}

	sip := types.ImportPath(selectedImportPath)
	return sip, nil
}
