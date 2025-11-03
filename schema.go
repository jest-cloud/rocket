package rocket

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jensneuse/abstractlogger"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astnormalization"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/asttransform"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

// BuildSchema builds an executable GraphQL schema from .graphql file and module resolvers
// This is the main entry point for Rocket
func BuildSchema(config Config, modules ...ModuleResolvers) (*Schema, error) {
	// Read schema file
	schemaBytes, err := os.ReadFile(config.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse schema using graphql-go-tools astparser
	schemaDoc, report := astparser.ParseGraphqlDocumentString(string(schemaBytes))
	if report.HasErrors() {
		return nil, fmt.Errorf("failed to parse schema: %v", report.Error())
	}

	// Merge with base schema (required by planner)
	err = asttransform.MergeDefinitionWithBaseSchema(&schemaDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to merge base schema: %w", err)
	}

	// Create resolver registry from modules
	resolvers := NewResolverRegistry(modules...)

	// Build execution plan using graphql-go-tools
	planner, err := buildExecutionPlan(&schemaDoc, resolvers)
	if err != nil {
		return nil, fmt.Errorf("failed to build execution plan: %w", err)
	}

	return &Schema{
		config:        config,
		resolvers:     resolvers,
		schemaDoc:     &schemaDoc,
		planner:       planner,
	}, nil
}

// buildExecutionPlan creates an execution plan using graphql-go-tools planner
func buildExecutionPlan(schemaDoc *ast.Document, resolvers *ResolverRegistry) (*plan.Planner, error) {
	// Create Rocket DataSource factory
	dsFactory := NewRocketDataSourceFactory(resolvers, schemaDoc)

	// Convert to DataSourceConfiguration
	dsConfig, err := dsFactory.ToDataSourceConfiguration("rocket", "Rocket")
	if err != nil {
		return nil, fmt.Errorf("failed to create DataSource configuration: %w", err)
	}

	// Create planner configuration
	// DataSourceConfiguration extends DataSource, so we can use it directly
	plannerConfig := plan.Configuration{
		DataSources: []plan.DataSource{dsConfig},
		Fields:      plan.FieldConfigurations{},
		Logger:      abstractlogger.Noop{},
	}

	// Create planner
	planner, err := plan.NewPlanner(plannerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create planner: %w", err)
	}

	return planner, nil
}

// Schema represents a compiled and executable GraphQL schema
type Schema struct {
	config    Config
	resolvers *ResolverRegistry
	schemaDoc *ast.Document
	planner   *plan.Planner
}

// Execute executes a GraphQL query/mutation
// Field order is always preserved for better developer experience
func (s *Schema) Execute(ctx context.Context, query string, variables map[string]interface{}, operationName string) *Result {
	// Parse query using graphql-go-tools
	queryDoc, report := astparser.ParseGraphqlDocumentString(query)
	if report.HasErrors() {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("Failed to parse query: %v", report.Error()),
			}},
		}
	}

	// Normalize query (required by planner)
	astnormalization.NormalizeOperation(&queryDoc, s.schemaDoc, &report)
	if report.HasErrors() {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("Failed to normalize query: %v", report.Error()),
			}},
		}
	}

	// Create execution plan
	var planReport operationreport.Report
	execPlan := s.planner.Plan(&queryDoc, s.schemaDoc, operationName, &planReport)
	if planReport.HasErrors() {
		errors := make([]Error, 0)
		for _, err := range planReport.InternalErrors {
			errors = append(errors, Error{Message: err.Error()})
		}
		for _, err := range planReport.ExternalErrors {
			errors = append(errors, Error{Message: err.Message})
		}
		return &Result{Errors: errors}
	}

	// Execute using graphql-go-tools resolver
	return ExecutePlan(ctx, execPlan, variables, s.resolvers)
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
