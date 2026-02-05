package errors_test

import (
	stderrors "errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hyperterse/hyperterse/core/shared/errors"
)

func TestNewAppError(t *testing.T) {
	tests := []struct {
		name           string
		code           errors.ErrorCode
		message        string
		err            error
		expectedStatus int
	}{
		{
			name:           "not found error",
			code:           errors.ErrCodeNotFound,
			message:        "resource not found",
			err:            nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "validation error",
			code:           errors.ErrCodeValidationError,
			message:        "invalid input",
			err:            nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "internal error",
			code:           errors.ErrCodeInternalError,
			message:        "internal error",
			err:            stderrors.New("underlying error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		appErr := errors.NewAppError(tt.code, tt.message, tt.err)
		assert.Equal(t, tt.code, appErr.Code)
		assert.Equal(t, tt.message, appErr.Message)
		assert.Equal(t, tt.expectedStatus, appErr.Status)
		if tt.err != nil {
			assert.Equal(t, tt.err, appErr.Unwrap())
		}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "not found error",
			err:      errors.NewAppError(errors.ErrCodeNotFound, "not found", nil),
			expected: true,
		},
		{
			name:     "query not found",
			err:      errors.NewAppError(errors.ErrCodeQueryNotFound, "query not found", nil),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.NewAppError(errors.ErrCodeInternalError, "internal error", nil),
			expected: false,
		},
		{
			name:     "non-app error",
			err:      stderrors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, errors.IsNotFound(tt.err))
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "validation error",
			err:      errors.NewAppError(errors.ErrCodeValidationError, "validation failed", nil),
			expected: true,
		},
		{
			name:     "invalid input",
			err:      errors.NewAppError(errors.ErrCodeInvalidInput, "invalid input", nil),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.NewAppError(errors.ErrCodeInternalError, "internal error", nil),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, errors.IsValidationError(tt.err))
		})
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appErr   *errors.AppError
		expected string
	}{
		{
			name: "error with underlying error",
			appErr: &errors.AppError{
				Code:    errors.ErrCodeNotFound,
				Message: "resource not found",
				Err:     stderrors.New("underlying error"),
			},
			expected: "NOT_FOUND: resource not found (underlying error)",
		},
		{
			name: "error without underlying error",
			appErr: &errors.AppError{
				Code:    errors.ErrCodeValidationError,
				Message: "validation failed",
				Err:     nil,
			},
			expected: "VALIDATION_ERROR: validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.appErr.Error())
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name           string
		code           errors.ErrorCode
		message        string
		err            error
		expectedStatus int
	}{
		{
			name:           "wrap not found error",
			code:           errors.ErrCodeNotFound,
			message:        "wrapped error",
			err:            stderrors.New("original error"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "wrap validation error",
			code:           errors.ErrCodeValidationError,
			message:        "wrapped validation error",
			err:            stderrors.New("original validation error"),
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := errors.WrapError(tt.code, tt.message, tt.err)
			assert.Equal(t, tt.code, appErr.Code)
			assert.Equal(t, tt.message, appErr.Message)
			assert.Equal(t, tt.err, appErr.Unwrap())
			assert.Equal(t, tt.expectedStatus, appErr.Status)
		})
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		code           errors.ErrorCode
		expectedStatus int
	}{
		{
			name:           "not found",
			code:           errors.ErrCodeNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "query not found",
			code:           errors.ErrCodeQueryNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "adapter not found",
			code:           errors.ErrCodeAdapterNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid input",
			code:           errors.ErrCodeInvalidInput,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error",
			code:           errors.ErrCodeValidationError,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "execution failed",
			code:           errors.ErrCodeExecutionFailed,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "connection failed",
			code:           errors.ErrCodeConnectionFailed,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "internal error",
			code:           errors.ErrCodeInternalError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "unknown error code",
			code:           errors.ErrorCode("UNKNOWN"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := errors.NewAppError(tt.code, "test", nil)
			assert.Equal(t, tt.expectedStatus, appErr.Status)
		})
	}
}

func TestIsNotFound_AdapterNotFound(t *testing.T) {
	err := errors.NewAppError(errors.ErrCodeAdapterNotFound, "adapter not found", nil)
	assert.True(t, errors.IsNotFound(err))
}

func TestIsValidationError_AllCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "validation error",
			err:      errors.NewAppError(errors.ErrCodeValidationError, "validation failed", nil),
			expected: true,
		},
		{
			name:     "invalid input",
			err:      errors.NewAppError(errors.ErrCodeInvalidInput, "invalid input", nil),
			expected: true,
		},
		{
			name:     "not found error",
			err:      errors.NewAppError(errors.ErrCodeNotFound, "not found", nil),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, errors.IsValidationError(tt.err))
		})
	}
}
