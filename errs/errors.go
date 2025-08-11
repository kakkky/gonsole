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
	var errType ErrType
	switch {
	case errors.As(err, &internalErr):
		errType = INTERNAL_ERROR
	case errors.As(err, &badInputErr):
		errType = BAD_INPUT_ERROR
	default:
		errType = UNKNOWN_ERROR
	}
	fmt.Printf("\n\033[31m[%s]\n %s\033[0m\n\n", errType, err.Error())
}
