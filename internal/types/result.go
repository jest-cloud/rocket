package types

// Result represents the result of a GraphQL operation
type Result struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []Error     `json:"errors,omitempty"`
}

// Error represents a GraphQL error
type Error struct {
	Message string        `json:"message"`
	Path    []interface{} `json:"path,omitempty"`
}

