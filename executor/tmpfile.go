package executor

import (
	"os"
	"path/filepath"
)

var src = "package main\n\nimport()\n\nfunc main() {\n\t// 初期化コード\n}\n"

func makeTmpMainFile() (string, func(), error) {
	if err := os.Mkdir("tmp", 0755); err != nil && !os.IsExist(err) {
		return "", nil, err
	}
	tmpDir, err := os.MkdirTemp("tmp", "gonsole")
	if err != nil {
		return "", nil, err
	}
	if _, err := os.Create(filepath.Join(tmpDir, "main.go")); err != nil {
		return "", nil, err
	}
	tmpFilePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpFilePath, []byte(src), 0644); err != nil {
		return "", nil, err
	}
	cleaner := func() {
		os.Remove(tmpFilePath)
		os.Remove(tmpDir)
	}
	return tmpFilePath, cleaner, nil
}
