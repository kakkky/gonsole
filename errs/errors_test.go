package errs

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestHandleError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedErrType ErrType
	}{
		{
			name:            "InternalError",
			err:             NewInternalError("internal error occurred"),
			expectedErrType: INTERNAL_ERROR,
		},
		{
			name:            "BadInputError",
			err:             NewBadInputError("bad input"),
			expectedErrType: BAD_INPUT_ERROR,
		},
		{
			name:            "UnknownError",
			err:             errors.New("unknown error"),
			expectedErrType: UNKNOWN_ERROR,
		},
		{
			name:            "WrappedInternalError",
			err:             NewInternalError("wrapped error").Wrap(errors.New("original error")),
			expectedErrType: INTERNAL_ERROR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 標準出力を一時的に差し替え
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// テスト終了時に元に戻す
			defer func() {
				os.Stdout = oldStdout
			}()

			// エラー処理関数を実行
			HandleError(tt.err)

			// パイプを閉じて出力を読み取る
			w.Close()
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(r); err != nil {
				t.Fatalf("failed to read from pipe: %v", err)
			}
			output := buf.String()

			// エラーメッセージが出力に含まれているか確認
			if !strings.Contains(output, string(tt.expectedErrType)) {
				t.Errorf("expected error type %s in output, got %s", tt.expectedErrType, output)
			}
		})
	}
}
