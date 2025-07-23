package executor

import (
	"fmt"
	"os"
	"path/filepath"
)

var src = "package main\n\nimport()\n\nfunc main() {\n\t// 初期化コード\n}\n"

func makeTmpMainFile() (string, func(), error) {
	if err := os.Mkdir("tmp", 0755); err != nil && !os.IsExist(err) {
		return "", nil, err
	}
	gonsoleDir, err := os.MkdirTemp("tmp", "gonsole")
	if err != nil {
		return "", nil, err
	}
	if _, err := os.Create(filepath.Join(gonsoleDir, "main.go")); err != nil {
		return "", nil, err
	}
	tmpFilePath := filepath.Join(gonsoleDir, "main.go")
	if err := os.WriteFile(tmpFilePath, []byte(src), 0644); err != nil {
		return "", nil, err
	}
	cleaner := func() {
		os.Remove(tmpFilePath)
		os.RemoveAll(gonsoleDir)
		entries, err := os.ReadDir("tmp")
		if err != nil {
			fmt.Printf("Failed to read tmp directory: %v\n", err)
		}
		if len(entries) == 0 {
			os.Remove("tmp")
		}
	}
	return tmpFilePath, cleaner, nil
}
