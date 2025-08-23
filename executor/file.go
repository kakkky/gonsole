package executor

import (
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"path/filepath"

	"github.com/kakkky/gonsole/errs"
)

var src = "package main\n\nimport()\n\nfunc main() {\n\t// 初期化コード\n}\n"

func makeTmpMainFile() (string, func(), error) {
	if err := os.Mkdir("tmp", 0755); err != nil && !os.IsExist(err) {
		return "", nil, errs.NewInternalError("failed to create tmp directory").Wrap(err)
	}
	gonsoleDir, err := os.MkdirTemp("tmp", "gonsole")
	if err != nil {
		return "", nil, errs.NewInternalError("failed to create temporary directory").Wrap(err)
	}
	if _, err := os.Create(filepath.Join(gonsoleDir, "main.go")); err != nil {
		return "", nil, errs.NewInternalError("failed to create temporary main file").Wrap(err)
	}
	tmpFilePath := filepath.Join(gonsoleDir, "main.go")
	if err := os.WriteFile(tmpFilePath, []byte(src), 0644); err != nil {
		return "", nil, errs.NewInternalError("failed to write temporary main file").Wrap(err)
	}
	cleaner := func() {
		os.Remove(tmpFilePath)
		os.RemoveAll(gonsoleDir)
		entries, err := os.ReadDir("tmp")
		if err != nil {
			errs.HandleError(errs.NewInternalError("failed to read tmp directory").Wrap(err))
			return
		}
		if len(entries) == 0 {
			if err := os.Remove("tmp"); err != nil {
				errs.HandleError(errs.NewInternalError("failed to remove tmp directory").Wrap(err))
			}
		}
	}
	return tmpFilePath, cleaner, nil
}

func makeTmpFile(dir string) (string, func(), error) {
	tmpFile, err := os.CreateTemp(dir, "tmp_gonsole_*.go")
	if err != nil {
		return "", nil, errs.NewInternalError("failed to create temporary file").Wrap(err)
	}
	cleaner := func() {
		os.Remove(tmpFile.Name())
	}
	return tmpFile.Name(), cleaner, nil
}

func outputToFile(outputPath string, fileAst *ast.File) error {
	outFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return errs.NewInternalError("failed to open temporary file for writing").Wrap(err)
	}
	defer outFile.Close()
	fset := token.NewFileSet()
	if err := format.Node(outFile, fset, fileAst); err != nil {
		return errs.NewInternalError("failed to format temporary file").Wrap(err)
	}
	return nil
}
