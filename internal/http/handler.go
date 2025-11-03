package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jest-cloud/rocket/internal/types"
)

// Schema is the interface that the handler needs
// This avoids importing rocket package to prevent cycles
type Schema interface {
	Execute(ctx context.Context, query string, variables map[string]interface{}, operationName string) *types.Result
}

// Request represents a GraphQL HTTP request
type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// Handler creates an HTTP handler for GraphQL requests
// Works with both net/http and Gin (via gin.WrapH)
func Handler(schema Schema) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Recover from panics to prevent empty responses
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": fmt.Sprintf("Internal server error: %v", err),
						},
					},
				})
			}
		}()

		// Only allow POST and GET
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req Request

		// Handle POST requests
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": fmt.Sprintf("Failed to parse request: %v", err),
						},
					},
				})
				return
			}
		}

		// Handle GET requests (for playground)
		if r.Method == http.MethodGet {
			req.Query = r.URL.Query().Get("query")
			req.OperationName = r.URL.Query().Get("operationName")
			// Variables would be in query param as JSON string
		}

		// Execute the query/mutation
		result := schema.Execute(r.Context(), req.Query, req.Variables, req.OperationName)

		// Send response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			// If encoding fails, try to send an error response
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"message": fmt.Sprintf("Failed to encode response: %v", err),
					},
				},
			})
			return
		}
	}
}
