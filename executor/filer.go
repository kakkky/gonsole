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
	createSessionSrcFile() (sessionSrcFile *os.File, sessionSrcFileName string, cleanup func(), err error)
	flush(ast *ast.File, targetFile *os.File, fset *token.FileSet) error
}

type defaultFiler struct{}

func newDefaultFiler() *defaultFiler {
	return &defaultFiler{}
}

func (df *defaultFiler) createSessionSrcFile() (sessionSrcFile *os.File, sessionSrcFileName string, cleanup func(), err error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	prefix := r.Int63n(1e10)
	sessionSrcFileName = fmt.Sprintf("%d_gonsole_session_src.go", prefix)

	file, err := os.Create(sessionSrcFileName)
	if err != nil {
		return nil, "", nil, errs.NewInternalError("failed to create session src file").Wrap(err)
	}

	cleanup = func() {
		if err := os.Remove(sessionSrcFileName); err != nil {
			errs.HandleError(err)
		}
	}

	return file, sessionSrcFileName, cleanup, nil
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
