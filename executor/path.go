package executor

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kakkky/gonsole/errs"
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
// 複数のパスが見つかる可能性があるので呼び出しもとで適切に選択する必要がある
//
// MEMO: 現状はパッケージ名としてディレクトリ名が一致することを前提としている
func (e *Executor) resolveImportPath(pkgName string) ([]string, error) {
	var importPaths []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// パッケージ名に一致するディレクトリか？
		base := filepath.Base(path)
		if base == pkgName {
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
