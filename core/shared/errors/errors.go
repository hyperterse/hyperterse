package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents a standardized error code
type ErrorCode string

const (
	// Domain errors
	ErrCodeNotFound        ErrorCode = "NOT_FOUND"
	ErrCodeInvalidInput    ErrorCode = "INVALID_INPUT"
	ErrCodeValidationError ErrorCode = "VALIDATION_ERROR"

	// Application errors
	ErrCodeExecutionFailed ErrorCode = "EXECUTION_FAILED"
	ErrCodeQueryNotFound   ErrorCode = "QUERY_NOT_FOUND"
	ErrCodeAdapterNotFound ErrorCode = "ADAPTER_NOT_FOUND"

	// Infrastructure errors
	ErrCodeConnectionFailed ErrorCode = "CONNECTION_FAILED"
	ErrCodeInternalError    ErrorCode = "INTERNAL_ERROR"
)

// AppError represents an application error with code and context
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
	Status  int // HTTP status code
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Status:  getHTTPStatus(code),
	}
}

// WrapError wraps an existing error with an error code and message
func WrapError(code ErrorCode, message string, err error) *AppError {
	return NewAppError(code, message, err)
}

// getHTTPStatus maps error codes to HTTP status codes
func getHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrCodeNotFound, ErrCodeQueryNotFound, ErrCodeAdapterNotFound:
		return http.StatusNotFound
	case ErrCodeInvalidInput, ErrCodeValidationError:
		return http.StatusBadRequest
	case ErrCodeExecutionFailed, ErrCodeConnectionFailed:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrCodeNotFound || appErr.Code == ErrCodeQueryNotFound || appErr.Code == ErrCodeAdapterNotFound
	}
	return false
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrCodeValidationError || appErr.Code == ErrCodeInvalidInput
	}
	return false
}
