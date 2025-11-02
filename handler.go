package rocket

import (
	"encoding/json"
	"net/http"
)

// Request represents a GraphQL HTTP request
type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// Handler creates an HTTP handler for GraphQL requests
// Works with both net/http and Gin (via gin.WrapH)
func Handler(schema *Schema) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST and GET
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req Request

		// Handle POST requests
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		// Handle GET requests (for playground)
		if r.Method == http.MethodGet {
			req.Query = r.URL.Query().Get("query")
			req.OperationName = r.URL.Query().Get("operationName")
			// Variables would be in query param as JSON string
		}

		// Execute the query
		result := schema.Execute(r.Context(), req.Query, req.Variables, req.OperationName)

		// Send response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

