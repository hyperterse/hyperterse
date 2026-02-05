package http

import (
	"encoding/json"
	"net/http"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
)

// handleQuery handles query endpoint requests
func handleQuery(queryService interfaces.QueryService, query *hyperterse.Query) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handlerLog := logging.New("handler")
		handlerLog.Infof("Request: %s %s", r.Method, r.URL.Path)
		handlerLog.Debugf("Query: %s", query.Name)

		// Helper function to return error in documented format
		writeErrorResponse := func(w http.ResponseWriter, statusCode int, errorMsg string) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			responseJSON := map[string]any{
				"success": false,
				"error":   errorMsg,
				"results": []any{},
			}
			json.NewEncoder(w).Encode(responseJSON)
		}

		if r.Method != http.MethodPost {
			handlerLog.Warnf("Method not allowed: %s", r.Method)
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		// Parse JSON body
		var requestBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			handlerLog.Errorf("Failed to parse JSON body: %v", err)
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
			return
		}
		handlerLog.Debugf("Request body parsed, %d input(s)", len(requestBody))

		// Convert inputs to map[string]string (JSON-encoded)
		inputs := make(map[string]string)
		for k, v := range requestBody {
			jsonBytes, _ := json.Marshal(v)
			inputs[k] = string(jsonBytes)
		}

		// Execute query
		reqProto := &runtime.ExecuteQueryRequest{
			QueryName: query.Name,
			Inputs:    inputs,
		}
		resp, err := queryService.ExecuteQuery(r.Context(), reqProto)
		if err != nil {
			handlerLog.Errorf("Query execution failed: %v", err)
			writeErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		statusCode := http.StatusOK
		if !resp.Success {
			statusCode = http.StatusBadRequest
			handlerLog.Warnf("Query returned error: %s", resp.Error)
		} else {
			handlerLog.Debugf("Query executed successfully, %d result(s)", len(resp.Results))
		}
		handlerLog.Infof("Response: %d", statusCode)
		w.WriteHeader(statusCode)

		// Manually construct response to ensure 'results' is always included
		responseJSON := map[string]any{
			"success": resp.Success,
			"error":   resp.Error,
			"results": make([]any, 0),
		}

		// Convert results from proto format to regular JSON
		if len(resp.Results) > 0 {
			results := make([]map[string]any, len(resp.Results))
			for i, row := range resp.Results {
				rowMap := make(map[string]any)
				for key, valueJSON := range row.Fields {
					var value any
					if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
						value = valueJSON
					}
					rowMap[key] = value
				}
				results[i] = rowMap
			}
			responseJSON["results"] = results
		}

		json.NewEncoder(w).Encode(responseJSON)
	}
}
