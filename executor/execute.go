package executor

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/kakkky/gonsole/decls"
	"github.com/kakkky/gonsole/errs"
)

type Executor struct {
	modPath     string
	tmpCleaner  func()
	tmpFilePath string
	declEntry   *decls.DeclEntry
}

func NewExecutor(declEntry *decls.DeclEntry) (*Executor, error) {
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
	if err := cmd.Run(); err != nil {
		// エラーが発生した場合は、エラーメッセージを整形して表示
		errResult := stderrBuf.String()
		errs.HandleError(errs.NewInvalidSyntaxError(formatErr(errResult)))
		// エラー箇所は削除
		if err := e.deleteErrLine(errResult); err != nil {
			errs.HandleError(err)
		}
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
	fmt.Printf("\033[32m%s\033[0m", output)
}
