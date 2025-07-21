package executor

import (
	"os"

	"golang.org/x/mod/modfile"
)

func getGoModPath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mf, err := modfile.Parse("", data, nil)
	if err != nil {
		return "", err
	}
	return mf.Module.Mod.Path, nil
}
