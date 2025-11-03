package rocket

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

// testResolver implements ModuleResolvers for testing
type testResolver struct {
	queries map[string]FieldResolveFn
}

func (r *testResolver) QueryResolvers() map[string]FieldResolveFn {
	return r.queries
}

func (r *testResolver) MutationResolvers() map[string]FieldResolveFn {
	return map[string]FieldResolveFn{}
}

func (r *testResolver) TypeResolvers() map[string]map[string]FieldResolveFn {
	return map[string]map[string]FieldResolveFn{}
}

// TestSimpleQuery tests a basic query with the new graphql-go-tools implementation
func TestSimpleQuery(t *testing.T) {
	// Create a temporary schema file
	schemaContent := `
type Query {
  hello: String!
  users: [User!]!
  user: User!
}

type User {
  id: ID!
  name: String!
  email: String!
}
`

	tmpfile, err := os.CreateTemp("", "test-schema-*.graphql")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(schemaContent); err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Define test User type
	type User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	// Create resolver implementations
	queryResolvers := map[string]FieldResolveFn{
		"hello": func(p ResolveParams) (interface{}, error) {
			return "Hello, Rocket! 🚀", nil
		},
		"users": func(p ResolveParams) (interface{}, error) {
			return []*User{
				{ID: "1", Name: "Alice", Email: "alice@example.com"},
				{ID: "2", Name: "Bob", Email: "bob@example.com"},
			}, nil
		},
		"user": func(p ResolveParams) (interface{}, error) {
			return &User{ID: "1", Name: "Alice", Email: "alice@example.com"}, nil
		},
	}

	// Create a resolver that implements ModuleResolvers
	resolver := &testResolver{queries: queryResolvers}

	// Build schema
	schema, err := BuildSchema(Config{
		SchemaPath: tmpfile.Name(),
	}, resolver)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	// Test 1: Simple hello query
	t.Run("hello query", func(t *testing.T) {
		query := `query { hello }`
		result := schema.Execute(context.Background(), query, nil, "")

		if result.Errors != nil && len(result.Errors) > 0 {
			t.Errorf("Query returned errors: %v", result.Errors)
		}

		if result.Data == nil {
			t.Fatal("Result data is nil")
		}

		dataBytes, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		hello, ok := data["hello"]
		if !ok {
			t.Fatalf("Missing 'hello' field in result: %v", data)
		}

		if hello != "Hello, Rocket! 🚀" {
			t.Errorf("Expected 'Hello, Rocket! 🚀', got '%v'", hello)
		}

		t.Logf("✓ hello query succeeded: %s", string(dataBytes))
	})

	// Test 2: Users query (tests auto-resolution for list elements)
	t.Run("users query", func(t *testing.T) {
		query := `query { users { id name email } }`
		result := schema.Execute(context.Background(), query, nil, "")

		if result.Errors != nil && len(result.Errors) > 0 {
			t.Errorf("Query returned errors: %v", result.Errors)
			for _, err := range result.Errors {
				t.Logf("  Error: %s (path: %v)", err.Message, err.Path)
			}
		}

		if result.Data == nil {
			t.Fatal("Result data is nil")
		}

		dataBytes, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}

		t.Logf("✓ users query result: %s", string(dataBytes))
	})

	// Test 3: Single user query (tests auto-resolution for single object)
	t.Run("user query", func(t *testing.T) {
		query := `query { user { id name email } }`
		result := schema.Execute(context.Background(), query, nil, "")

		if result.Errors != nil && len(result.Errors) > 0 {
			t.Errorf("Query returned errors: %v", result.Errors)
			for _, err := range result.Errors {
				t.Logf("  Error: %s (path: %v)", err.Message, err.Path)
			}
		}

		if result.Data == nil {
			t.Fatal("Result data is nil")
		}

		dataBytes, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}

		t.Logf("✓ user query result: %s", string(dataBytes))
	})
}

