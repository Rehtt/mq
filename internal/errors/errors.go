package errors

import (
	"fmt"
)

// 定义业务错误类型
type ErrorType string

const (
	ErrorTypeNotFound      ErrorType = "NOT_FOUND"
	ErrorTypeAlreadyExists ErrorType = "ALREADY_EXISTS"
	ErrorTypeInvalidInput  ErrorType = "INVALID_INPUT"
	ErrorTypeInternal      ErrorType = "INTERNAL"
	ErrorTypeAuth          ErrorType = "AUTH"
)

// BusinessError 业务错误
type BusinessError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *BusinessError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *BusinessError) Unwrap() error {
	return e.Cause
}

// 便捷的错误创建函数
func NotFound(message string) *BusinessError {
	return &BusinessError{
		Type:    ErrorTypeNotFound,
		Message: message,
	}
}

func AlreadyExists(message string) *BusinessError {
	return &BusinessError{
		Type:    ErrorTypeAlreadyExists,
		Message: message,
	}
}

func InvalidInput(message string) *BusinessError {
	return &BusinessError{
		Type:    ErrorTypeInvalidInput,
		Message: message,
	}
}

func Internal(message string, cause error) *BusinessError {
	return &BusinessError{
		Type:    ErrorTypeInternal,
		Message: message,
		Cause:   cause,
	}
}

func Auth(message string) *BusinessError {
	return &BusinessError{
		Type:    ErrorTypeAuth,
		Message: message,
	}
}

// Wrap 包装已有错误
func Wrap(err error, message string) *BusinessError {
	if err == nil {
		return nil
	}

	// 如果已经是BusinessError，直接返回
	if be, ok := err.(*BusinessError); ok {
		return be
	}

	return &BusinessError{
		Type:    ErrorTypeInternal,
		Message: message,
		Cause:   err,
	}
}
