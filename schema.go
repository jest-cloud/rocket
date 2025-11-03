package rocket

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jensneuse/abstractlogger"
	"github.com/jest-cloud/rocket/internal/datasource"
	"github.com/jest-cloud/rocket/internal/execution"
	"github.com/jest-cloud/rocket/internal/registry"
	"github.com/jest-cloud/rocket/internal/resolver"
	"github.com/jest-cloud/rocket/internal/types"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astnormalization"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/asttransform"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/introspection"
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

	// First merge with base schema to add built-in types (ID, String, Int, etc.)
	// This must be done BEFORE NormalizeDefinition
	// This is required by graphql-go-tools planner
	err = asttransform.MergeDefinitionWithBaseSchema(&schemaDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to merge base schema: %w", err)
	}

	// Then normalize the schema to merge extensions
	// This merges extend type Query/Mutation into base Query/Mutation
	// NormalizeDefinition handles merging extensions properly
	astnormalization.NormalizeDefinition(&schemaDoc, &report)
	if report.HasErrors() {
		return nil, fmt.Errorf("failed to normalize schema: %v", report.Error())
	}


	// Create resolver registry from modules
	resolvers := registry.NewResolverRegistry(modules...)

	// Note: We don't pre-build the planner here anymore.
	// Instead, we'll create a fresh planner for each query execution.
	// This ensures that visitor.Operation is properly available during planning.

	return &Schema{
		config:    config,
		resolvers: resolvers,
		schemaDoc: &schemaDoc,
	}, nil
}

// executeMutationDirectly executes a mutation without using the DataSource/Planner
// This is a workaround for mutations with variables where the Input template doesn't evaluate
func (s *Schema) executeMutationDirectly(ctx context.Context, queryDoc *ast.Document, variables map[string]interface{}, requestedFields []string) *Result {
	if len(requestedFields) == 0 {
		return &Result{
			Errors: []Error{{Message: "No mutation fields found"}},
		}
	}
	
	// For now, handle single mutation field (most common case)
	mutationField := requestedFields[0]
	
	// Get the mutation resolver
	resolver, found := s.resolvers.GetMutationResolver(mutationField)
	if !found {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("No resolver found for mutation: %s", mutationField),
			}},
		}
	}
	
	// Extract selection set for this mutation field
	selectionSet := extractSelectionSetForField(queryDoc, mutationField)
	
	// Call the resolver with variables as arguments
	params := ResolveParams{
		Source:  nil,
		Args:    variables,
		Context: ctx,
		Info: ResolveInfo{
			FieldName:  mutationField,
			ParentType: "Mutation",
		},
	}
	
	result, err := resolver(params)
	if err != nil {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("Mutation error: %v", err),
			}},
		}
	}
	
	// Resolve nested fields in the result
	resolvedResult := s.resolveNestedFields(ctx, result, selectionSet)
	
	// Wrap in mutation field name
	data := map[string]interface{}{
		mutationField: resolvedResult,
	}
	
	return &Result{
		Data:   data,
		Errors: []Error{},
	}
}

// extractSelectionSetForField extracts the selection set for a specific field from the query
func extractSelectionSetForField(queryDoc *ast.Document, fieldName string) []string {
	if len(queryDoc.OperationDefinitions) == 0 {
		return []string{}
	}
	
	opDef := queryDoc.OperationDefinitions[0]
	if opDef.SelectionSet < 0 || opDef.SelectionSet >= len(queryDoc.SelectionSets) {
		return []string{}
	}
	
	selectionSet := queryDoc.SelectionSets[opDef.SelectionSet]
	for _, selRef := range selectionSet.SelectionRefs {
		if selRef < 0 || selRef >= len(queryDoc.Selections) {
			continue
		}
		selection := queryDoc.Selections[selRef]
		if selection.Kind == ast.SelectionKindField {
			fieldRef := selection.Ref
			if fieldRef >= 0 && fieldRef < len(queryDoc.Fields) {
				if queryDoc.FieldNameString(fieldRef) == fieldName {
					// Found the field, now extract its selection set
					field := queryDoc.Fields[fieldRef]
					if field.SelectionSet < 0 || field.SelectionSet >= len(queryDoc.SelectionSets) {
						return []string{}
					}
					fieldSelectionSet := queryDoc.SelectionSets[field.SelectionSet]
					var fields []string
					for _, selRef := range fieldSelectionSet.SelectionRefs {
						if selRef >= 0 && selRef < len(queryDoc.Selections) {
							sel := queryDoc.Selections[selRef]
							if sel.Kind == ast.SelectionKindField {
								fRef := sel.Ref
								if fRef >= 0 && fRef < len(queryDoc.Fields) {
									fields = append(fields, queryDoc.FieldNameString(fRef))
								}
							}
						}
					}
					return fields
				}
			}
		}
	}
	
	return []string{}
}

// resolveNestedFields resolves nested fields in a result object
func (s *Schema) resolveNestedFields(ctx context.Context, source interface{}, fields []string) interface{} {
	if source == nil || len(fields) == 0 {
		return source
	}
	
	// If source is a map, only return requested fields
	if sourceMap, ok := source.(map[string]interface{}); ok {
		result := make(map[string]interface{})
		for _, field := range fields {
			if value, exists := sourceMap[field]; exists {
				result[field] = value
			}
		}
		return result
	}
	
	// If source is a struct, use reflection to extract fields
	result := make(map[string]interface{})
	for _, field := range fields {
		params := ResolveParams{
			Source:  source,
			Args:    nil,
			Context: ctx,
			Info: ResolveInfo{
				FieldName:  field,
				ParentType: "", // We don't know the type here
			}	,
		}
		value, _ := resolver.DefaultFieldResolver(params)
		result[field] = value
	}
	
	return result
}

// extractFieldsAndType extracts top-level field names and operation type from a GraphQL operation
func extractFieldsAndType(queryDoc *ast.Document) ([]string, ast.OperationType) {
	fields := []string{}
	opType := ast.OperationTypeQuery // Default to query
	
	if len(queryDoc.OperationDefinitions) == 0 {
		return fields, opType
	}
	
	opDef := queryDoc.OperationDefinitions[0]
	opType = opDef.OperationType
	
	if opDef.SelectionSet < 0 || opDef.SelectionSet >= len(queryDoc.SelectionSets) {
		return fields, opType
	}
	
	selectionSet := queryDoc.SelectionSets[opDef.SelectionSet]
	for _, selRef := range selectionSet.SelectionRefs {
		if selRef < 0 || selRef >= len(queryDoc.Selections) {
			continue
		}
		selection := queryDoc.Selections[selRef]
		if selection.Kind == ast.SelectionKindField {
			fieldRef := selection.Ref
			if fieldRef >= 0 && fieldRef < len(queryDoc.Fields) {
				fieldName := queryDoc.FieldNameString(fieldRef)
				fields = append(fields, fieldName)
			}
		}
	}
	
	return fields, opType
}

// buildExecutionPlanWithFields creates an execution plan with knowledge of requested fields and operation type
func buildExecutionPlanWithFields(schemaDoc *ast.Document, resolvers *registry.ResolverRegistry, requestedFields []string, operationType ast.OperationType) (*plan.Planner, error) {
	// Create Rocket DataSource factory with requested fields and operation type
	dsFactory := datasource.NewRocketDataSourceFactoryWithFields(resolvers, schemaDoc, requestedFields, operationType)

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
	resolvers *registry.ResolverRegistry
	schemaDoc *ast.Document
	// Note: We don't cache the planner anymore - we create a fresh one for each query
	// to ensure visitor.Operation is properly populated during planning
}

// Execute executes a GraphQL query/mutation
// Field order is always preserved for better developer experience
func (s *Schema) Execute(ctx context.Context, query string, variables map[string]interface{}, operationName string) *Result {
	// Check if this is an introspection query - handle it directly
	// This bypasses the planner which has issues routing introspection to DataSources
	if isIntrospectionQuery(query) {
		return s.handleIntrospectionDirectly(ctx, query, variables)
	}
	
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
	
	// Extract the requested field names and operation type from the query BEFORE planning
	// This is necessary because Register() is called during planner creation,
	// before the operation is available to visitor.Operation
	requestedFields, operationType := extractFieldsAndType(&queryDoc)
	
	// SPECIAL HANDLING FOR MUTATIONS:
	// Mutations with variables don't work well with our DataSource approach because
	// the Input template doesn't evaluate properly. So we execute mutations directly.
	if operationType == ast.OperationTypeMutation {
		return s.executeMutationDirectly(ctx, &queryDoc, variables, requestedFields)
	}
	
	// Create a fresh planner for this query with knowledge of requested fields and operation type
	planner, err := buildExecutionPlanWithFields(s.schemaDoc, s.resolvers, requestedFields, operationType)
	if err != nil {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("Failed to build execution plan: %v", err),
			}},
		}
	}
	
	// Create execution plan for this specific query
	var planReport operationreport.Report
	execPlan := planner.Plan(&queryDoc, s.schemaDoc, operationName, &planReport)
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
	result := execution.ExecutePlan(ctx, execPlan, variables, s.resolvers)
	// Convert internal result to public Result type
	return &Result{
		Data:   result.Data,
		Errors: convertErrors(result.Errors),
	}
}

// Result represents the result of a GraphQL operation
// Re-exported from internal/types for public API
type Result = types.Result

// Error represents a GraphQL error
// Re-exported from internal/types for public API
type Error = types.Error

// convertErrors converts internal errors to public errors
func convertErrors(errors []types.Error) []Error {
	if errors == nil {
		return nil
	}
	result := make([]Error, len(errors))
	for i, e := range errors {
		result[i] = Error(e)
	}
	return result
}

// handleIntrospectionDirectly handles introspection queries directly
// without going through the planner, bypassing the DataSource routing issue
// This is a workaround for the planner not routing introspection to our DataSource
func (s *Schema) handleIntrospectionDirectly(ctx context.Context, query string, variables map[string]interface{}) *Result {
	
	// Parse query using graphql-go-tools
	queryDoc, report := astparser.ParseGraphqlDocumentString(query)
	if report.HasErrors() {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("Failed to parse introspection query: %v", report.Error()),
			}},
		}
	}
	
	// Normalize query
	astnormalization.NormalizeOperation(&queryDoc, s.schemaDoc, &report)
	if report.HasErrors() {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("Failed to normalize introspection query: %v", report.Error()),
			}},
		}
	}
	
	// Use graphql-go-tools introspection generator
	generator := introspection.NewGenerator()
	introspectionData := &introspection.Data{}
	
	// Generate introspection data from schema
	generator.Generate(s.schemaDoc, &report, introspectionData)
	if report.HasErrors() {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("Failed to generate introspection data: %v", report.Error()),
			}},
		}
	}
	
	// Extract the type name from the query and return the appropriate data
	// This is a simplified approach - for full introspection support, we'd need to
	// properly execute the query selection set using the resolver
	var result map[string]interface{}
	
	// Check if query is asking for __type(name: "...")
	if strings.Contains(query, "__type") {
		// Extract type name from variables or query
		typeName := ""
		
		// First check variables
		if variables != nil {
			if name, ok := variables["name"].(string); ok {
				typeName = name
			}
		}
		
		// If no variable, try to extract from query string
		if typeName == "" {
			// Look for __type(name: "TypeName") pattern
			if idx := strings.Index(query, "__type"); idx >= 0 {
				// Extract type name from query (simplified parsing)
				// Look for name: "TypeName" pattern
				nameIdx := strings.Index(query[idx:], "name:")
				if nameIdx > 0 {
					namePart := query[idx+nameIdx:]
					// Find the quoted string
					startQuote := strings.Index(namePart, "\"")
					if startQuote > 0 {
						endQuote := strings.Index(namePart[startQuote+1:], "\"")
						if endQuote > 0 {
							typeName = namePart[startQuote+1 : startQuote+1+endQuote]
						}
					}
				}
			}
		}
		
		if typeName != "" {
			typeData := introspectionData.Schema.TypeByName(typeName)
			if typeData == nil {
				return &Result{
					Errors: []Error{{
						Message: fmt.Sprintf("Type '%s' not found", typeName),
					}},
				}
			}
			result = map[string]interface{}{
				"__type": typeData,
			}
		} else {
			// __type query without name argument - should return null
			result = map[string]interface{}{
				"__type": nil,
			}
		}
	} else if strings.Contains(query, "__schema") {
		// __schema query
		result = map[string]interface{}{
			"__schema": introspectionData.Schema,
		}
	} else {
		// Unknown introspection query
		return &Result{
			Errors: []Error{{
				Message: "Unknown introspection query",
			}},
		}
	}
	
	return &Result{
		Data: result,
	}
}

// isIntrospectionQuery checks if a query is an introspection query
// Introspection queries query __schema or __type which are meta fields
func isIntrospectionQuery(query string) bool {
	// Simple heuristic: check if the query contains __schema or __type
	// This covers the standard GraphQL introspection queries
	return strings.Contains(query, "__schema") || strings.Contains(query, "__type")
}
