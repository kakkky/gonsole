package errs

import (
	"errors"
	"fmt"
)

// ErrType はエラーの種類を表す
type ErrType string

// ErrType の種類
const (
	UnknownErrorType  ErrType = "UNKNOWN ERROR"   // 不明なエラー
	InternalErrorType ErrType = "INTERNAL ERROR"  // 内部的なエラー
	BadInputErrorType ErrType = "BAD INPUT ERROR" // ユーザーからの不正な入力に起因するエラー
)

// InternalError は内部的なエラー
type InternalError struct {
	message string
	wrapped error
}

// NewInternalError は新しい内部エラーを作成する
func NewInternalError(message string) *InternalError {
	return &InternalError{
		message: message,
	}
}

// Wrap は元のエラーを内部エラーにラップする
func (e *InternalError) Wrap(err error) *InternalError {
	e.wrapped = err
	return e
}

// Error はエラーメッセージを返す
func (e *InternalError) Error() string {
	if e.wrapped == nil {
		return e.message
	}
	return e.message + ": " + e.wrapped.Error()
}

// BadInputError は不正入力エラー
type BadInputError struct {
	message string
	wrapped error
}

// NewBadInputError は新しい不正入力エラーを作成する
func NewBadInputError(message string) *BadInputError {
	return &BadInputError{
		message: message,
	}
}

// Wrap は元のエラーを不正入力エラーにラップする
func (e *BadInputError) Wrap(err error) *BadInputError {
	e.wrapped = err
	return e
}

// Error はエラーメッセージを返す
func (e *BadInputError) Error() string {
	if e.wrapped == nil {
		return e.message
	}
	return e.message + ": " + e.wrapped.Error()
}

// HandleError はエラーを受け取り、適切な形式で表示する
func HandleError(err error) {
	const (
		redColor   = "\033[31m"
		resetColor = "\033[0m"
	)

	var internalErr *InternalError
	var badInputErr *BadInputError
	var errType ErrType
	switch {
	case errors.As(err, &internalErr):
		errType = InternalErrorType
	case errors.As(err, &badInputErr):
		errType = BadInputErrorType
	default:
		errType = UnknownErrorType
	}

	fmt.Printf("\n%s[%s]\n %s%s\n\n", redColor, errType, err.Error(), resetColor)
}
