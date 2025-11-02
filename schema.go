package rocket

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// BuildSchema builds an executable GraphQL schema from .graphql file and module resolvers
// This is the main entry point for Rocket
func BuildSchema(config Config, modules ...ModuleResolvers) (*Schema, error) {
	// Read schema file
	schemaBytes, err := os.ReadFile(config.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse schema using gqlparser
	parsedSchema, err := gqlparser.LoadSchema(&ast.Source{
		Name:  config.SchemaPath,
		Input: string(schemaBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Create resolver registry from modules
	resolvers := NewResolverRegistry(modules...)

	// Build graphql-go schema
	builder := newSchemaBuilder(resolvers, parsedSchema)
	executableSchema, err := builder.build()
	if err != nil {
		return nil, fmt.Errorf("failed to build schema: %w", err)
	}

	return &Schema{
		config:           config,
		resolvers:        resolvers,
		executableSchema: executableSchema,
		parsedSchema:     parsedSchema,
	}, nil
}

// Schema represents a compiled and executable GraphQL schema
type Schema struct {
	config           Config
	resolvers        *ResolverRegistry
	executableSchema graphql.Schema
	parsedSchema     *ast.Schema
}

// Execute executes a GraphQL query/mutation
// Field order is always preserved for better developer experience
func (s *Schema) Execute(ctx context.Context, query string, variables map[string]interface{}, operationName string) *Result {
	params := graphql.Params{
		Schema:         s.executableSchema,
		RequestString:  query,
		VariableValues: variables,
		OperationName:  operationName,
		Context:        ctx,
	}

	// Skip field ordering for introspection queries (they use fragment spreads heavily)
	if isIntrospectionQuery(query) {
		result := graphql.Do(params)
		return convertResult(result)
	}

	// Execute with field order preservation for regular queries
	return ExecuteWithFieldOrder(params, query)
}


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

// isIntrospectionQuery checks if a query is an introspection query
// Introspection queries query __schema or __type which are meta fields
func isIntrospectionQuery(query string) bool {
	// Simple heuristic: check if the query contains __schema or __type
	// This covers the standard GraphQL introspection queries
	return strings.Contains(query, "__schema") || strings.Contains(query, "__type")
}
