package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
	"github.com/hyperterse/hyperterse/core/infrastructure/transport/http/dto"
	"github.com/hyperterse/hyperterse/core/shared/errors"
)

// BaseHandler provides common functionality for all handlers
type BaseHandler struct {
	logger logging.Logger
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(tag string) *BaseHandler {
	return &BaseHandler{
		logger: logging.New(tag),
	}
}

// WriteJSON writes a JSON response
func (h *BaseHandler) WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Errorf("Failed to encode JSON response: %v", err)
	}
}

// WriteError writes an error response
func (h *BaseHandler) WriteError(w http.ResponseWriter, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.NewAppError(errors.ErrCodeInternalError, err.Error(), err)
	}

	h.WriteJSON(w, appErr.Status, dto.ErrorResponse{
		Success: false,
		Error:   appErr.Message,
		Results: []interface{}{},
	})
}

// WriteValidationError writes a validation error response
func (h *BaseHandler) WriteValidationError(w http.ResponseWriter, validationErrors map[string]string) {
	details := make([]dto.ErrorDetail, 0, len(validationErrors))
	for field, tag := range validationErrors {
		details = append(details, dto.ErrorDetail{
			Field:   field,
			Tag:     tag,
			Message: "Validation failed",
		})
	}

	h.WriteJSON(w, http.StatusBadRequest, dto.ValidationErrorResponse{
		Success: false,
		Error:   "Validation failed",
		Details: details,
	})
}

// WriteSuccess writes a success response
func (h *BaseHandler) WriteSuccess(w http.ResponseWriter, data interface{}) {
	h.WriteJSON(w, http.StatusOK, data)
}
