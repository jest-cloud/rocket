package rocket_test

import (
	"github.com/jest-cloud/rocket"
)
import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

// TestViewerQuery tests a viewer query with nested fields similar to fire-print
func TestViewerQuery(t *testing.T) {
	// Create a temporary schema file
	schemaContent := `
type Query {
  viewer: User
}

type User {
  id: ID!
  email: String!
  firstName: String!
  lastName: String!
  organizations: [Organization!]!
}

type Organization {
  id: ID!
  name: String!
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

	// Define test types
	type Organization struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	type User struct {
		ID            string         `json:"id"`
		Email         string         `json:"email"`
		FirstName     string         `json:"firstName"`
		LastName      string         `json:"lastName"`
		Organizations []Organization `json:"organizations"`
	}

	// Create resolver implementations
	queryResolvers := map[string]rocket.FieldResolveFn{
		"viewer": func(p rocket.ResolveParams) (interface{}, error) {
			return &User{
				ID:        "user-1",
				Email:     "alice@example.com",
				FirstName: "Alice",
				LastName:  "Smith",
				Organizations: []Organization{
					{ID: "org-1", Name: "Acme Corp"},
					{ID: "org-2", Name: "Tech Solutions"},
				},
			}, nil
		},
	}

	// Create a resolver that implements rocket.ModuleResolvers
	resolver := &testResolver{queries: queryResolvers}

	// Build schema
	schema, err := rocket.BuildSchema(rocket.Config{
		SchemaPath: tmpfile.Name(),
	}, resolver)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	// Test: Simple viewer query with id only
	t.Run("viewer with id", func(t *testing.T) {
		query := `query { viewer { id } }`
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

		dataMap, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		viewer, ok := dataMap["viewer"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected viewer to be a map")
		}

		if viewer["id"] != "user-1" {
			t.Errorf("Expected viewer.id to be 'user-1', got %v", viewer["id"])
		}

		// Verify __typename is auto-injected from struct name
		if typename, ok := viewer["__typename"].(string); !ok || typename != "User" {
			t.Errorf("Expected viewer.__typename to be 'User', got %v", viewer["__typename"])
		}

		t.Logf("✓ viewer query result: id=%s, __typename=%s", viewer["id"], viewer["__typename"])
	})

	// Test: Viewer with all user fields
	t.Run("viewer with all fields", func(t *testing.T) {
		query := `query { 
			viewer { 
				id 
				email 
				firstName 
				lastName 
			} 
		}`
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

		t.Logf("✓ viewer query with all fields result: %s", string(dataBytes))
	})

	// Test: Viewer with nested organizations
	t.Run("viewer with organizations", func(t *testing.T) {
		query := `query { 
			viewer { 
				id 
				email 
				firstName 
				lastName 
				organizations {
					id
					name
				}
			} 
		}`
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

		t.Logf("✓ viewer query with organizations result: %s", string(dataBytes))
	})
}

