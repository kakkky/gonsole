package executor

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func extractFilePaths() ([]string, error) {
	var filepaths []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// ディレクトリの場合
		if d.IsDir() {
			// 隠しディレクトリのみスキップ
			if path != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			// testdataディレクトリはスキップ
			if d.Name() == "testdata" {
				return fs.SkipDir
			}
			return nil
		}

		// ファイルの場合、フィルタリング
		name := d.Name()
		if name == "go.mod" || name == "go.sum" || strings.HasPrefix(name, ".") || strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
			return nil // 特定のファイルはスキップするが、ディレクトリはスキップしない
		}
		filepaths = append(filepaths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filepaths, nil
}
