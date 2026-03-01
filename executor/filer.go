package executor

import (
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"path/filepath"

	"github.com/kakkky/gonsole/errs"
)

//go:generate mockgen -package=executor -source=./filer.go -destination=./filer_mock.go
type filer interface {
	createSessionSrcDir() (sessionSrcDir string, cleanup func(), err error)
	createSessionSrcFile() (sessionSrcFile *os.File, sessionSrcFilePath string, cleanup func(), err error)
	flush(ast *ast.File, targetFile *os.File, fset *token.FileSet) error
	prepareGoModFile(targetDir string, content []byte) (goModFilePath string, cleanup func(), err error)
	cleanupGoSumFile(targetDir string) error
}

type defaultFiler struct{}

func newDefaultFiler() *defaultFiler {
	return &defaultFiler{}
}

const SESSION_SRC_FILE_PATH = "gonsole_session_src/session_src.go"
const SESSION_SRC_DIR_PERMISSION = 0755

func (df *defaultFiler) createSessionSrcDir() (sessionSrcDir string, cleanup func(), err error) {
	sessionSrcDir = filepath.Dir(SESSION_SRC_FILE_PATH)
	if err := os.MkdirAll(sessionSrcDir, SESSION_SRC_DIR_PERMISSION); err != nil {
		return "", nil, errs.NewInternalError("failed to create session src dir").Wrap(err)
	}

	cleanup = func() {
		if err := os.RemoveAll(sessionSrcDir); err != nil {
			errs.HandleError(err)
		}
	}

	return sessionSrcDir, cleanup, nil
}

func (df *defaultFiler) createSessionSrcFile() (sessionSrcFile *os.File, sessionSrcFilePath string, cleanup func(), err error) {
	file, err := os.Create(SESSION_SRC_FILE_PATH)
	if err != nil {
		return nil, "", nil, errs.NewInternalError("failed to create session src file").Wrap(err)
	}

	cleanup = func() {
		if err := os.Remove(SESSION_SRC_FILE_PATH); err != nil {
			errs.HandleError(err)
		}
	}

	return file, SESSION_SRC_FILE_PATH, cleanup, nil
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

const GO_MOD_FILE_NAME = "go.mod"
const GO_MOD_FILE_PERMISSION = 0644

func (df *defaultFiler) prepareGoModFile(targetDir string, content []byte) (goModFilePath string, cleanup func(), err error) {
	goModFilePath = filepath.Join(targetDir, GO_MOD_FILE_NAME)
	if _, err := os.Create(goModFilePath); err != nil {
		return "", nil, errs.NewInternalError("failed to create go.mod file").Wrap(err)
	}

	if err := os.WriteFile(goModFilePath, content, GO_MOD_FILE_PERMISSION); err != nil {
		return "", nil, errs.NewInternalError("failed to write go.mod file").Wrap(err)
	}

	cleanup = func() {
		if err := os.Remove(goModFilePath); err != nil {
			errs.HandleError(err)
		}
	}

	return goModFilePath, cleanup, nil
}

const GO_SUM_FILE_NAME = "go.sum"

func (df *defaultFiler) cleanupGoSumFile(targetDir string) error {
	goSumFilePath := filepath.Join(targetDir, GO_SUM_FILE_NAME)
	if _, err := os.Stat(goSumFilePath); err == nil {
		if err := os.Remove(goSumFilePath); err != nil {
			return errs.NewInternalError("failed to remove go.sum file").Wrap(err)
		}
	}
	return nil
}
