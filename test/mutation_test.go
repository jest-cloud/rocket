package rocket_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/jest-cloud/rocket"
)

// testResolverWithMutations implements rocket.ModuleResolvers for mutation testing
type testResolverWithMutations struct {
	queries   map[string]rocket.FieldResolveFn
	mutations map[string]rocket.FieldResolveFn
}

func (r *testResolverWithMutations) QueryResolvers() map[string]rocket.FieldResolveFn {
	if r.queries == nil {
		return map[string]rocket.FieldResolveFn{}
	}
	return r.queries
}

func (r *testResolverWithMutations) MutationResolvers() map[string]rocket.FieldResolveFn {
	return r.mutations
}

func (r *testResolverWithMutations) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
	return map[string]map[string]rocket.FieldResolveFn{}
}

// TestMutation tests a simple mutation
func TestMutation(t *testing.T) {
	// Create a temporary schema file
	schemaContent := `
type Mutation {
  createUser(input: CreateUserInput!): User!
}

input CreateUserInput {
  name: String!
  email: String!
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

	// Define test type
	type User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	// Create mutation resolver
	mutationResolvers := map[string]rocket.FieldResolveFn{
		"createUser": func(p rocket.ResolveParams) (interface{}, error) {
			// Extract input from args
			input, ok := p.Args["input"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid input")
			}

			name, _ := input["name"].(string)
			email, _ := input["email"].(string)

			return &User{
				ID:    "user-123",
				Name:  name,
				Email: email,
			}, nil
		},
	}

	// Create a resolver that implements rocket.ModuleResolvers
	resolver := &testResolverWithMutations{mutations: mutationResolvers}

	// Build schema
	schema, err := rocket.BuildSchema(rocket.Config{
		SchemaPath: tmpfile.Name(),
	}, resolver)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	// Test mutation
	t.Run("createUser mutation", func(t *testing.T) {
		query := `mutation($input: CreateUserInput!) { 
			createUser(input: $input) { 
				id
				name
				email
			} 
		}`
		
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"name":  "Alice",
				"email": "alice@example.com",
			},
		}

		result := schema.Execute(context.Background(), query, variables, "")

		if result.Errors != nil && len(result.Errors) > 0 {
			t.Errorf("Mutation returned errors: %v", result.Errors)
			for _, err := range result.Errors {
				t.Logf("  Error: %s (path: %v)", err.Message, err.Path)
			}
			t.FailNow()
		}

		if result.Data == nil {
			t.Fatal("Result data is nil")
		}

		dataBytes, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}

		t.Logf("✓ Mutation result: %s", string(dataBytes))

		// Verify result structure
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected result.Data to be a map, got %T", result.Data)
		}

		createUser, ok := data["createUser"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected createUser to be a map, got %T", data["createUser"])
		}

		if createUser["id"] != "user-123" {
			t.Errorf("Expected id to be 'user-123', got %v", createUser["id"])
		}

		if createUser["name"] != "Alice" {
			t.Errorf("Expected name to be 'Alice', got %v", createUser["name"])
		}

		if createUser["email"] != "alice@example.com" {
			t.Errorf("Expected email to be 'alice@example.com', got %v", createUser["email"])
		}
	})
}

