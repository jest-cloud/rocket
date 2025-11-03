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

// TestFirePrintLikeSetup tests with a schema structure similar to fire-print
// This uses extend type Query like fire-print does
func TestFirePrintLikeSetup(t *testing.T) {
	// Create a schema file similar to fire-print's structure
	schemaContent := `
type Query {
  _empty: String
}

extend type Query {
  viewer: User
}

type User {
  id: ID!
  email: String!
  firstName: String!
  lastName: String!
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

	// Define test types matching fire-print's User struct
	type User struct {
		ID        string `json:"id" bson:"_id,omitempty"`
		Email     string `json:"email" bson:"email"`
		FirstName string `json:"firstName" bson:"firstName"`
		LastName  string `json:"lastName" bson:"lastName"`
	}

	// Create resolver implementation matching fire-print
	queryResolvers := map[string]rocket.FieldResolveFn{
		"viewer": func(p rocket.ResolveParams) (interface{}, error) {
			// Return a User like fire-print does
			return &User{
				ID:        "user-1",
				Email:     "alice@example.com",
				FirstName: "Alice",
				LastName:  "Smith",
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

	// Test query: viewer { id }
	query := `query { viewer { id } }`
	result := schema.Execute(context.Background(), query, nil, "")

	if len(result.Errors) > 0 {
		t.Errorf("Query returned errors: %v", result.Errors)
	}

	if result.Data == nil {
		t.Fatal("Query returned no data")
	}

	dataJSON, err := json.Marshal(result.Data)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	expectedJSON := `{"viewer":{"id":"user-1"}}`
	if string(dataJSON) != expectedJSON {
		t.Errorf("Expected %s, got %s", expectedJSON, string(dataJSON))
	} else {
		t.Logf("✓ viewer { id } query result: %s", string(dataJSON))
	}
}

