package executor

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"math/rand"
	"os"
	"time"

	"github.com/kakkky/gonsole/errs"
)

//go:generate mockgen -package=executor -source=./filer.go -destination=./filer_mock.go
type filer interface {
	createTmpFile() (tmpFile *os.File, tmpFileName string, cleanup func(), err error)
	flush(ast *ast.File, targetFile *os.File, fset *token.FileSet) error
}

type defaultFiler struct{}

func newDefaultFiler() *defaultFiler {
	return &defaultFiler{}
}

func (df *defaultFiler) createTmpFile() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	prefix := r.Int63n(1e10)
	tmpFileName = fmt.Sprintf("%d_gonsole_tmp.go", prefix)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return nil, "", nil, errs.NewInternalError("failed to create temporary file").Wrap(err)
	}

	cleanup = func() {
		if err := os.Remove(tmpFileName); err != nil {
			errs.HandleError(err)
		}
	}

	return file, tmpFileName, cleanup, nil
}

func (df *defaultFiler) flush(ast *ast.File, targetFile *os.File, fset *token.FileSet) error {
	if _, err := targetFile.Seek(0, 0); err != nil {
		return errs.NewInternalError("failed to seek file").Wrap(err)
	}
	if err := targetFile.Truncate(0); err != nil {
		return errs.NewInternalError("failed to truncate file").Wrap(err)
	}
	if err := format.Node(targetFile, fset, ast); err != nil {
		return errs.NewInternalError("failed to format AST node").Wrap(err)
	}
	return nil
}
