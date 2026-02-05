package dto

// HealthResponse represents a health check response
type HealthResponse struct {
	Success bool `json:"success"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag"`
}

// ValidationErrorResponse represents a validation error response
type ValidationErrorResponse struct {
	Success bool          `json:"success"`
	Error   string        `json:"error"`
	Details []ErrorDetail `json:"details,omitempty"`
}
