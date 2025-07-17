package executor

import (
	"go/ast"
	"os"
	"strings"

	"golang.org/x/mod/modfile"
)

func ExtractImportPaths(nodes map[string]*ast.Package, modPath string) []string {
	importPaths := make([]string, 0)
	for _, pkg := range nodes {
		for _, file := range pkg.Files {
			for _, imp := range file.Imports {
				modPath, err := getGoModPath(modPath)
				if err != nil || modPath == "" {
					continue // エラーが発生した場合はスキップ
				}
				importPath := strings.ReplaceAll(imp.Path.Value, modPath+"/", "")
				importPaths = append(importPaths, importPath)
			}
		}
	}
	return uniqueImportPaths(importPaths)
}

func getGoModPath(path string) (string, error) {
	// モジュール名を除去
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mf, err := modfile.Parse("path", data, nil)
	if err != nil {
		return "", err
	}
	return mf.Module.Mod.Path, nil
}

func uniqueImportPaths(paths []string) []string {
	seen := make(map[string]struct{})
	uniquePaths := make([]string, 0, len(paths))

	for _, path := range paths {
		if _, exists := seen[path]; !exists {
			seen[path] = struct{}{}
			uniquePaths = append(uniquePaths, path)
		}
	}

	return uniquePaths
}
