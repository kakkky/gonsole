package errs

import (
	"errors"
	"fmt"
)

type ErrType string

const (
	INTERNAL_ERROR  ErrType = "INTERNAL ERROR"
	BAD_INPUT_ERROR ErrType = "BAD INPUT ERROR"
	UNKNOWN_ERROR   ErrType = "UNKNOWN ERROR"
)

// 内部的なエラー
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
func (e *InternalError) Wrap(err error) error {
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

// ユーザーからの不正な入力に起因するエラー
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
func (e *BadInputError) Wrap(err error) error {
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
		errType = INTERNAL_ERROR
	case errors.As(err, &badInputErr):
		errType = BAD_INPUT_ERROR
	default:
		errType = UNKNOWN_ERROR
	}

	fmt.Printf("\n%s[%s]\n %s%s\n\n", redColor, errType, err.Error(), resetColor)
}
