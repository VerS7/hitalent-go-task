package apperror

import "errors"

type Code string

const (
	CodeValidation Code = "validation"
	CodeNotFound   Code = "not_found"
	CodeConflict   Code = "conflict"
	CodeInternal   Code = "internal"
)

type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func GetCode(err error) Code {
	if err == nil {
		return ""
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}

	return CodeInternal
}
