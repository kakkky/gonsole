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
type InvalidSyntaxError struct {
	message string
	wrapped error
}

func NewInvalidSyntaxError(message string) *InvalidSyntaxError {
	return &InvalidSyntaxError{
		message: message,
	}
}

func (e *InvalidSyntaxError) Wrap(err error) error {
	e.wrapped = err
	return e
}

func (e *InvalidSyntaxError) Error() string {
	if e.wrapped == nil {
		return e.message
	}
	return e.message + ": " + e.wrapped.Error()
}

// エラーを処理する関数
func HandleError(err error) {
	var internalErr *InternalError
	var syntaxErr *InvalidSyntaxError
	switch {
	case errors.As(err, &internalErr):
		fmt.Printf("\033[31m[INTERNAL ERR]\n %s\033[0m\n", err.Error())
	case errors.As(err, &syntaxErr):
		fmt.Printf("\033[31m[INVALID SYNTAX ERR]\n %s\033[0m\n", err.Error())
	default:
		fmt.Printf("\033[31m[UNKNOWN ERR]\n %s\033[0m\n", err.Error())
	}
}
