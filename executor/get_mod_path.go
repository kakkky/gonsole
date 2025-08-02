package executor

import (
	"os"

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
