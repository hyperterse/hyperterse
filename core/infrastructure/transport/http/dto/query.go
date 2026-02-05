package dto

// ExecuteQueryRequest represents a query execution request
type ExecuteQueryRequest struct {
	// Inputs map will be validated per query definition
	// This is a generic structure - specific validation happens in handlers
	Inputs map[string]interface{} `json:"inputs" validate:"required"`
}

// ExecuteQueryResponse represents a query execution response
type ExecuteQueryResponse struct {
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
	Results []map[string]interface{} `json:"results"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Results []interface{} `json:"results"`
}
