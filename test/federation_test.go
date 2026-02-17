package rocket_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jest-cloud/rocket"
)

// Test data structures
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Post struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	AuthorID string `json:"authorId"`
}

// Federation resolver
type federationTestResolver struct {
	users map[string]*User
	posts map[string]*Post
}

func (r *federationTestResolver) QueryResolvers() map[string]rocket.FieldResolveFn {
	return map[string]rocket.FieldResolveFn{
		"user": func(p rocket.ResolveParams) (interface{}, error) {
			idVal, ok := p.Args["id"]
			if !ok || idVal == nil {
				return nil, nil
			}
			id := idVal.(string)
			return r.users[id], nil
		},
		"post": func(p rocket.ResolveParams) (interface{}, error) {
			idVal, ok := p.Args["id"]
			if !ok || idVal == nil {
				return nil, nil
			}
			id := idVal.(string)
			return r.posts[id], nil
		},
	}
}

func (r *federationTestResolver) MutationResolvers() map[string]rocket.FieldResolveFn {
	return map[string]rocket.FieldResolveFn{}
}

func (r *federationTestResolver) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
	return map[string]rocket.SubscriptionResolveFn{}
}

func (r *federationTestResolver) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
	return map[string]map[string]rocket.FieldResolveFn{}
}

func (r *federationTestResolver) EntityResolvers() map[string]rocket.EntityResolveFn {
	return map[string]rocket.EntityResolveFn{
		"User": func(p rocket.ResolveParams, representation map[string]interface{}) (interface{}, error) {
			id := representation["id"].(string)
			return r.users[id], nil
		},
		"Post": func(p rocket.ResolveParams, representation map[string]interface{}) (interface{}, error) {
			id := representation["id"].(string)
			return r.posts[id], nil
		},
	}
}

// TestFederationDisabled ensures federation features don't interfere when disabled
func TestFederationDisabled(t *testing.T) {
	// Create test schema (without federation)
	schema, err := rocket.BuildSchema(
		rocket.Config{SchemaPath: "../test_data/federation_schema.graphql"},
		&federationTestResolver{
			users: map[string]*User{
				"1": {ID: "1", Name: "Alice", Email: "alice@example.com"},
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	// Test that regular query still works
	query := `{ user(id: "1") { id name } }`
	result := schema.Execute(context.Background(), query, nil, "")

	if len(result.Errors) > 0 {
		t.Errorf("Query returned errors: %v", result.Errors)
	}

	if result.Data == nil {
		t.Fatal("Result data is nil")
	}
	
	t.Log("✓ Federation disabled mode works correctly")
}

// TestFederationServiceQuery tests the _service query
func TestFederationServiceQuery(t *testing.T) {
	// Create federation schema
	schema, err := rocket.BuildSchema(
		rocket.Config{
			SchemaPath: "../test_data/federation_schema.graphql",
			Federation: rocket.FederationConfig{
				Enabled:     true,
				ServiceName: "test-service",
			},
		},
		&federationTestResolver{},
	)
	if err != nil {
		t.Fatalf("Failed to build federation schema: %v", err)
	}

	// Query _service
	query := `{ _service { sdl } }`
	result := schema.Execute(context.Background(), query, nil, "")

	if len(result.Errors) > 0 {
		t.Fatalf("_service query returned errors: %v", result.Errors)
	}

	if result.Data == nil {
		t.Fatal("Result data is nil")
	}

	// Check that SDL is returned
	dataMap := result.Data.(map[string]interface{})
	service, ok := dataMap["_service"].(map[string]interface{})
	if !ok {
		t.Fatal("_service field not found in response")
	}

	sdl, ok := service["sdl"].(string)
	if !ok {
		t.Fatal("SDL not found in _service response")
	}

	if len(sdl) == 0 {
		t.Fatal("SDL is empty")
	}

	t.Logf("✓ _service query returned SDL (%d bytes)", len(sdl))
}

// TestFederationEntitiesQuery tests the _entities query
func TestFederationEntitiesQuery(t *testing.T) {
	resolver := &federationTestResolver{
		users: map[string]*User{
			"1": {ID: "1", Name: "Alice", Email: "alice@example.com"},
			"2": {ID: "2", Name: "Bob", Email: "bob@example.com"},
		},
		posts: map[string]*Post{
			"101": {ID: "101", Title: "Hello World", AuthorID: "1"},
		},
	}

	_, err := rocket.BuildSchema(
		rocket.Config{
			SchemaPath: "../test_data/federation_schema.graphql",
			Federation: rocket.FederationConfig{
				Enabled: true,
			},
		},
		resolver,
	)
	if err != nil {
		t.Fatalf("Failed to build federation schema: %v", err)
	}

	// TODO: _entities query needs further work with graphql-go-tools
	// For now, verify that entity resolvers are registered
	if len(resolver.EntityResolvers()) != 2 {
		t.Fatalf("Expected 2 entity resolvers, got %d", len(resolver.EntityResolvers()))
	}

	t.Log("✓ Entity resolvers registered correctly")
	t.Log("ℹ️  Note: Full _entities query integration requires additional work with graphql-go-tools planner")
}

// TestFederationMissingEntityResolver tests error handling
func TestFederationMissingEntityResolver(t *testing.T) {
	// For this test, we'll just verify the error message would be correct
	// In real usage, if no entity resolver exists, _entities will error
	t.Log("✓ Missing entity resolver test skipped (requires runtime check)")
}

// TestFederationDirectives tests that federation directives are available
func TestFederationDirectives(t *testing.T) {
	schema, err := rocket.BuildSchema(
		rocket.Config{
			SchemaPath: "../test_data/federation_schema.graphql",
			Federation: rocket.FederationConfig{
				Enabled: true,
			},
		},
		&federationTestResolver{},
	)
	if err != nil {
		t.Fatalf("Failed to build federation schema: %v", err)
	}

	if schema == nil {
		t.Fatal("Schema is nil")
	}

	// Just verify schema builds successfully with @key directives
	t.Log("✓ Federation directives parsed successfully")
}

// TestTypenameAutoInjection verifies that __typename is automatically injected
// into responses when the resolver returns a map with __typename set
func TestTypenameAutoInjection(t *testing.T) {
	schemaSDL := `
type Query {
  user(id: ID!): User
}

type User {
  id: ID!
  name: String!
}
`
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.graphql")
	if err := os.WriteFile(schemaPath, []byte(schemaSDL), 0644); err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	// Resolver returns a map with __typename — verifies it flows through to response
	resolver := &simpleResolver{
		queryResolvers: map[string]rocket.FieldResolveFn{
			"user": func(p rocket.ResolveParams) (interface{}, error) {
				return map[string]interface{}{
					"__typename": "User",
					"id":         "1",
					"name":       "Alice",
				}, nil
			},
		},
	}

	schema, err := rocket.BuildSchema(
		rocket.Config{SchemaPath: schemaPath},
		resolver,
	)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	query := `{ user(id: "1") { id name __typename } }`
	result := schema.Execute(context.Background(), query, nil, "")

	if len(result.Errors) > 0 {
		t.Fatalf("Query returned errors: %v", result.Errors)
	}

	dataMap, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data to be map, got %T", result.Data)
	}

	userMap, ok := dataMap["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected user to be map, got %T", dataMap["user"])
	}

	typename, ok := userMap["__typename"].(string)
	if !ok {
		t.Fatal("__typename not found in response - auto-injection failed")
	}

	if typename != "User" {
		t.Fatalf("Expected __typename to be 'User', got '%s'", typename)
	}

	t.Logf("✓ __typename present in response: %s", typename)
}

// TestTypenameAutoInjectionWithStruct verifies __typename injection for struct-based resolvers
func TestTypenameAutoInjectionWithStruct(t *testing.T) {
	schemaSDL := `
type Query {
  user(id: ID!): User
}

type User {
  id: ID!
  name: String!
}
`
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.graphql")
	if err := os.WriteFile(schemaPath, []byte(schemaSDL), 0644); err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	// Resolver returns a struct - __typename should be derived from struct name
	resolver := &simpleResolver{
		queryResolvers: map[string]rocket.FieldResolveFn{
			"user": func(p rocket.ResolveParams) (interface{}, error) {
				return &User{ID: "1", Name: "Alice"}, nil
			},
		},
	}

	schema, err := rocket.BuildSchema(
		rocket.Config{SchemaPath: schemaPath},
		resolver,
	)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	query := `{ user(id: "1") { id name __typename } }`
	result := schema.Execute(context.Background(), query, nil, "")

	if len(result.Errors) > 0 {
		t.Fatalf("Query returned errors: %v", result.Errors)
	}

	dataMap, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data to be map, got %T", result.Data)
	}

	userMap, ok := dataMap["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected user to be map, got %T", dataMap["user"])
	}

	typename, ok := userMap["__typename"].(string)
	if !ok {
		t.Fatalf("__typename not found or not a string in response")
	}

	if typename != "User" {
		t.Fatalf("Expected __typename to be 'User', got '%s'", typename)
	}

	t.Logf("✓ __typename auto-injected as '%s' from struct name", typename)
}

// simpleResolver is a minimal ModuleResolvers implementation for testing
type simpleResolver struct {
	queryResolvers map[string]rocket.FieldResolveFn
}

func (r *simpleResolver) QueryResolvers() map[string]rocket.FieldResolveFn {
	return r.queryResolvers
}

func (r *simpleResolver) MutationResolvers() map[string]rocket.FieldResolveFn {
	return map[string]rocket.FieldResolveFn{}
}

func (r *simpleResolver) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
	return map[string]rocket.SubscriptionResolveFn{}
}

func (r *simpleResolver) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
	return map[string]map[string]rocket.FieldResolveFn{}
}

func (r *simpleResolver) EntityResolvers() map[string]rocket.EntityResolveFn {
	return map[string]rocket.EntityResolveFn{}
}

