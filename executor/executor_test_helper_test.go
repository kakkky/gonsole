package executor

import (
	"go/ast"
	"go/token"
	"os"
	"testing"

	"github.com/kakkky/gonsole/types"
	gomock "go.uber.org/mock/gomock"
)

const testSessionSrcDir = "gonsole_session_src"
const testGoModFilePath = "gonsole_session_src/go.mod"

// resolveExpect は importPathResolver.resolve の期待値を表す。
type resolveExpect struct {
	pkgName    types.PkgName
	importPath types.ImportPath
}

// flushExpect は flush mock の登録パターンを表す。
// declFlush() または exprFlush() で生成する。
type flushExpect struct {
	kind    flushKind
	onFlush func(*ast.File, *os.File, *token.FileSet) error
}

type flushKind int

const (
	// flushKindDecl は宣言文のケース。flush が呼ばれることだけ確認する。
	flushKindDecl flushKind = iota
	// flushKindExpr は式文のケース。flush 直前の AST 中間状態も検証する。
	flushKindExpr
)

// declFlush は宣言文テスト用の flushExpect を返す。
func declFlush() flushExpect {
	return flushExpect{kind: flushKindDecl}
}

// exprFlush は式文テスト用の flushExpect を返す。fn で flush 直前の AST を検証できる。
func exprFlush(fn func(*ast.File, *os.File, *token.FileSet) error) flushExpect {
	return flushExpect{kind: flushKindExpr, onFlush: fn}
}

// setupMocks はテスト全体で共通して必要な filer/commander/importPathResolver の mock 設定を行うヘルパー。
//
// flusher が flushKindExpr の場合、flush は DoAndReturn(flusher.onFlush).Times(1) で登録される。
// flusher が flushKindDecl の場合、execGoRunErr が nil でなければ Times(2)、nil なら Times(1) で Return(nil) が登録される。
//
// resolveExpects が指定された場合、その順序で resolve の期待値が登録される（複数の場合は gomock.InOrder で順序保証）。
func setupMocks(
	t *testing.T,
	mockFiler *Mockfiler,
	mockCommander *Mockcommander,
	mockImportPathResolver *MockimportPathResolver,
	sessionSrcFilePath string,
	flusher flushExpect,
	execGoRunOut []byte,
	execGoRunErr error,
	resolveExpects ...resolveExpect,
) *os.File {
	t.Helper()

	// filer
	mockFiler.EXPECT().createSessionSrcDir().Return(testSessionSrcDir, func() {}, nil).Times(1)

	var sessionSrcFile *os.File
	mockFiler.EXPECT().createSessionSrcFile().DoAndReturn(func() (*os.File, string, func(), error) {
		r, w, _ := os.Pipe()
		sessionSrcFile = w
		return w, sessionSrcFilePath, func() {
			if err := r.Close(); err != nil {
				t.Errorf("failed to close pipe reader: %v", err)
			}
		}, nil
	}).Times(1)

	expectFlush := mockFiler.EXPECT().flush(
		gomock.AssignableToTypeOf(&ast.File{}),
		gomock.AssignableToTypeOf(&os.File{}),
		gomock.AssignableToTypeOf(&token.FileSet{}),
	)
	switch flusher.kind {
	case flushKindExpr:
		expectFlush.DoAndReturn(flusher.onFlush).Times(1)
	case flushKindDecl:
		flushTimes := 1
		if execGoRunErr != nil {
			flushTimes = 2
		}
		expectFlush.Return(nil).Times(flushTimes)
	}

	mockFiler.EXPECT().prepareGoModFile(testSessionSrcDir, gomock.AssignableToTypeOf([]byte{})).Return(testGoModFilePath, func() {}, nil).Times(1)
	mockFiler.EXPECT().cleanupGoSumFile(testSessionSrcDir).Return(nil).Times(1)

	// commander
	mockCommander.EXPECT().execGoListMod().Return([]byte("github.com/kakkky/gonsole"), nil).Times(1)
	mockCommander.EXPECT().execGoModTidy(testSessionSrcDir).Return(nil).Times(1)
	mockCommander.EXPECT().execGoRun(testGoModFilePath, sessionSrcFilePath).Return(execGoRunOut, execGoRunErr).Times(1)

	// importPathResolver
	if len(resolveExpects) == 1 {
		mockImportPathResolver.EXPECT().resolve(resolveExpects[0].pkgName).Return(resolveExpects[0].importPath, nil).Times(1)
	} else if len(resolveExpects) > 1 {
		calls := make([]any, len(resolveExpects))
		for i, e := range resolveExpects {
			calls[i] = mockImportPathResolver.EXPECT().resolve(e.pkgName).Return(e.importPath, nil).Times(1)
		}
		gomock.InOrder(calls...)
	}

	return sessionSrcFile
}
