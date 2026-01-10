package executor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"

	"os/exec"
	"regexp"
	"strings"

	"github.com/kakkky/gonsole/decls"
	"github.com/kakkky/gonsole/errs"
	"github.com/kakkky/gonsole/types"
)

type Executor struct {
	modPath     string
	tmpCleaner  func()
	tmpFilePath string
	declEntry   *decls.DeclEntry
	astCache    *astCache
}

type astCache struct {
	// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
	nodes map[types.PkgName][]*ast.Package
	fset  *token.FileSet
}

// nolint:staticcheck // 定義されている変数名、関数名など名前だけに関心があるため、*ast.Packageだけで十分
func NewExecutor(declEntry *decls.DeclEntry, nodes map[types.PkgName][]*ast.Package, fset *token.FileSet) (*Executor, error) {
	tmpFilePath, cleaner, err := makeTmpMainFile()
	if err != nil {
		return nil, err
	}
	modPath, err := getGoModPath("go.mod")
	if err != nil {
		return nil, err
	}
	return &Executor{
		modPath:     modPath,
		tmpCleaner:  cleaner,
		tmpFilePath: tmpFilePath,
		declEntry:   declEntry,
		astCache: &astCache{
			nodes: nodes,
			fset:  fset,
		},
	}, nil
}

func (e *Executor) Execute(input string) {
	if input == "" {
		return
	}

	if err := e.addInputToTmpSrc(input); err != nil {
		errs.HandleError(err)
	}

	cmd := exec.Command("go", "run", e.tmpFilePath)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	// 評価を実行
	defer func() {
		// panicが発生した場合に備えてrecover
		// エラーメッセージはerrs.HandleErrorで表示されるため、ここでは特に何もせず、復帰するようにしておくのみ
		_ = recover()
	}()
	if err := cmd.Run(); err != nil {
		// エラーが発生した場合は、エラーメッセージを整形して表示
		errResult := stderrBuf.String()
		errs.HandleError(errs.NewBadInputError(formatErr(errResult)))
		// 最後にエラー箇所は削除
		defer func() {
			if err := e.deleteErrLine(errResult); err != nil {
				errs.HandleError(err)
			}
		}()
	}

	// 評価結果を出力
	result := stdoutBuf.String()
	if result != "" {
		outputResult(result)
	}
	// 関数呼び出しだった場合はそれをtmpファイルから削除する
	if err := e.deleteCallExpr(); err != nil {
		errs.HandleError(err)
	}

	// 変数エントリに登録する
	if err := e.declEntry.Register(input); err != nil {
		errs.HandleError(err)
	}
}

func (e *Executor) Close() {
	if e.tmpCleaner != nil {
		e.tmpCleaner()
	}
}

func formatErr(input string) string {
	lines := strings.Split(input, "\n")
	var result []string

	cliPattern := regexp.MustCompile(`^# command-line-arguments$`)
	pathPrefixPattern := regexp.MustCompile(`tmp/gonsole[0-9]+/main\.go:\d+:\d+:\s*`)
	var errCount int
	for _, line := range lines {
		if cliPattern.MatchString(line) || line == "" {
			continue
		}
		line = pathPrefixPattern.ReplaceAllString(line, "")
		if !strings.HasPrefix(line, "\t") {
			errCount++
		}
		result = append(result, line)
	}

	return fmt.Sprintf("\n%d errors found\n\n%s\n\n", errCount, strings.Join(result, "\n"))
}

func outputResult(output string) {
	fmt.Printf("\n\033[32m%s\033[0m\n", output)
}
