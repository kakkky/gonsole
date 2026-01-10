package executor

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kakkky/go-prompt"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
	"golang.org/x/mod/modfile"
)

func getGoModPath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errs.NewInternalError("failed to read go.mod file").Wrap(err)
	}
	mf, err := modfile.Parse("", data, nil)
	if err != nil {
		return "", errs.NewInternalError("failed to parse go.mod file").Wrap(err)
	}
	return mf.Module.Mod.Path, nil
}

// resolveImportPath はパッケージ名からインポートパスを解決する
// 現在のディレクトリからパッケージ名に一致するディレクトリを探索し、相対パスを返す
// 複数のパスが見つかった場合は、ユーザーに選択を促す
//
// MEMO: 現状はパッケージ名としてディレクトリ名が一致することを前提としている
func (e *Executor) resolveImportPathForAdd(pkgName types.PkgName) (string, error) {
	var importPaths []string
	if stdPkg, ok := isStandardPackage(pkgName); ok {
		importPaths = append(importPaths, string(stdPkg))
	}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// パッケージ名に一致するディレクトリか？
		base := filepath.Base(path)
		// 隠しディレクトリやvendor、tmp、node_modules、testdata、アンダースコアで始まるディレクトリはスキップ
		if base == "vendor" || base == "tmp" || base == "node_modules" || base == "testdata" || strings.HasPrefix(base, "_") || (strings.HasPrefix(base, ".") && base != ".") {
			// vendorディレクトリはスキップ
			return filepath.SkipDir
		}
		if base == string(pkgName) {
			relPath, err := filepath.Rel(".", path)
			if err != nil {
				return err
			}
			// モジュールルートのパスと相対パスを結合したものをインポートパスとする
			importPaths = append(importPaths, filepath.ToSlash(filepath.Join(e.modPath, relPath)))
		}
		return nil
	})
	if err != nil {
		return "", errs.NewInternalError("failed to walk directory").Wrap(err)
	}
	var importPath string
	if len(importPaths) == 1 {
		importPath = importPaths[0]
	}

	if len(importPaths) > 1 {
		// 複数の候補がある場合は選択を促す
		selectedPath, err := selectImportPathRepl(importPaths)
		if err != nil {
			return "", err
		}
		importPath = selectedPath
	}
	return importPath, nil
}

func selectImportPathRepl(paths []string) (string, error) {
	fmt.Print(toBlue("\nMultiple import candidates found.\n\nUse Tab key to select import path.\n\n"))
	for _, path := range paths {
		fmt.Println(toBlue("- " + path))
	}
	completer := func(d prompt.Document) []prompt.Suggest {
		s := make([]prompt.Suggest, len(paths))
		for i, path := range paths {
			s[i] = prompt.Suggest{Text: path}
		}
		return s
	}
	fmt.Print(toBlue("\n>>> "))
	result := prompt.Input(
		"",
		completer, prompt.OptionShowCompletionAtStart(),
		prompt.OptionPreviewSuggestionTextColor(prompt.Turquoise),
		prompt.OptionInputTextColor(prompt.Turquoise),
	)
	if result == "" {
		// Enterが押された場合はエラーとする
		return "", errs.NewBadInputError("no import path selected")
	}
	if !slices.Contains(paths, result) {
		// 基本はtabキーで選択肢にあるものを選べ部はずだが、なんらかの文字を入力してEnterを押した場合はエラーとする
		return "", errs.NewBadInputError(fmt.Sprintf("no existing import path: %s", result))
	}
	return result, nil
}

func toBlue(text string) string {
	return fmt.Sprintf("\033[94m%s\033[0m", text)
}

// resolveImportPath はパッケージ名からインポートパスを解決する
// 現在のディレクトリからパッケージ名に一致するディレクトリを探索し、相対パスを返す
// 複数のパスを返し、呼び出し元でハンドリングする
// resolveImportPathForAddと処理をまとめなかったは、削除ではrepl機能が必要なかったため
//
// MEMO: 現状はパッケージ名としてディレクトリ名が一致することを前提としている
func (e *Executor) resolveImportPathForDelete(pkgName types.PkgName) ([]string, error) {
	var importPaths []string
	if stdPkg, ok := isStandardPackage(pkgName); ok {
		importPaths = append(importPaths, string(stdPkg))
	}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// パッケージ名に一致するディレクトリか？
		base := filepath.Base(path)
		// 隠しディレクトリやvendor、tmp、node_modules、testdata、アンダースコアで始まるディレクトリはスキップ
		if base == "vendor" || base == "tmp" || base == "node_modules" || base == "testdata" || strings.HasPrefix(base, "_") || (strings.HasPrefix(base, ".") && base != ".") {
			// vendorディレクトリはスキップ
			return filepath.SkipDir
		}
		if base == string(pkgName) {
			relPath, err := filepath.Rel(".", path)
			if err != nil {
				return err
			}
			// モジュールルートのパスと相対パスを結合したものをインポートパスとする
			importPaths = append(importPaths, filepath.ToSlash(filepath.Join(e.modPath, relPath)))
		}
		return nil
	})
	if err != nil {
		return nil, errs.NewInternalError("failed to walk directory").Wrap(err)
	}
	return importPaths, nil
}
