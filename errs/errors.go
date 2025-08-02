package errs

import (
	"errors"
	"fmt"
)

// 内部的なエラー
type InternalError struct {
	message string
	wrapped error
}

func NewInternalError(message string) *InternalError {
	return &InternalError{
		message: message,
	}
}
func (e *InternalError) Wrap(err error) error {
	e.wrapped = err
	return e
}

func (e *InternalError) Error() string {
	if e.wrapped == nil {
		return e.message
	}
	return e.message + ": " + e.wrapped.Error()
}

// ユーザー起因の無効な構文エラー
type BadInputError struct {
	message string
	wrapped error
}

func NewBadInputError(message string) *BadInputError {
	return &BadInputError{
		message: message,
	}
}

func (e *BadInputError) Wrap(err error) error {
	e.wrapped = err
	return e
}

func (e *BadInputError) Error() string {
	if e.wrapped == nil {
		return e.message
	}
	return e.message + ": " + e.wrapped.Error()
}

// エラーを処理する関数
func HandleError(err error) {
	var internalErr *InternalError
	var badInputErr *BadInputError
	switch {
	case errors.As(err, &internalErr):
		fmt.Printf("\033[31m[INTERNAL ERROR]\n %s\033[0m\n", err.Error())
	case errors.As(err, &badInputErr):
		fmt.Printf("\033[31m[BAD INPUT ERROR]\n %s\033[0m\n", err.Error())
	default:
		fmt.Printf("\033[31m[UNKNOWN ERROR]\n %s\033[0m\n", err.Error())
	}
}
