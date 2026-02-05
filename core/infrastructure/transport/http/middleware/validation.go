package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidateRequest validates the request body against a struct
func ValidateRequest(schema interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new instance of the schema type
			// This is a simplified version - in production, use reflection or code generation
			if err := json.NewDecoder(r.Body).Decode(schema); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid JSON",
				})
				return
			}

			// Validate the struct
			if err := validate.Struct(schema); err != nil {
				validationErrors := make(map[string]string)
				if validationErrs, ok := err.(validator.ValidationErrors); ok {
					for _, validationErr := range validationErrs {
						validationErrors[validationErr.Field()] = validationErr.Tag()
					}
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "Validation failed",
					"details": validationErrors,
				})
				return
			}

			// Store validated schema in context for handler to use
			ctx := r.Context()
			// In a real implementation, you'd store the schema in context
			// For now, we'll pass it through a custom context key
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ValidateQueryParams validates query parameters
func ValidateQueryParams(validatorFunc func(*http.Request) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := validatorFunc(r); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": err.Error(),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
