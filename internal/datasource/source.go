package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/jest-cloud/rocket/internal/registry"
	"github.com/jest-cloud/rocket/internal/resolver"
	"github.com/jest-cloud/rocket/internal/types"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/datasource/httpclient"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/introspection"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

// Helper function to get Query resolver names for debugging
func getQueryResolverNames(resolvers *registry.ResolverRegistry) []string {
	if resolvers == nil {
		return nil
	}
	names := make([]string, 0, len(resolvers.Query))
	for name := range resolvers.Query {
		names = append(names, name)
	}
	return names
}

// Helper function to get map keys for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper function to get fieldMap keys for debugging
func getFieldMapKeys(m map[string]fieldInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper to get Mutation resolver names for debugging
func getMutationResolverNames(resolvers *registry.ResolverRegistry) []string {
	keys := []string{}
	for k := range resolvers.Mutation {
		keys = append(keys, k)
	}
	return keys
}

// RocketSource implements resolve.DataSource for Rocket resolvers
// This is where we actually execute Rocket's resolver functions
// Each RocketSource instance is responsible for resolving ONE specific field
type RocketSource struct {
	resolvers  *registry.ResolverRegistry
	fieldMap   map[string]fieldInfo
	schema     *ast.Document // Store schema for introspection support
	targetField *fieldCoordinate
	parentPath string // For debugging
}

// fieldCoordinate identifies a specific GraphQL field
type fieldCoordinate struct {
	TypeName  string // e.g., "Query", "Mutation", "User"
	FieldName string // e.g., "hello", "users", "id"
}

// Load executes Rocket resolvers and returns the result
func (s *RocketSource) Load(ctx context.Context, input []byte, out *bytes.Buffer) error {
	// Parse input JSON
	var inputData map[string]interface{}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &inputData); err != nil {
			inputData = make(map[string]interface{})
		}
	}
	
	// Get arguments and source from input
	arguments := make(map[string]interface{})
	if args, ok := inputData["arguments"].(map[string]interface{}); ok {
		arguments = args
	}
	
	source := inputData["source"]
	
	// Determine which field we're resolving
	var fieldName string
	var parentType string
	
	// Determine which field to resolve
	if s.targetField != nil {
		// TargetField was set during planning
		fieldName = s.targetField.FieldName
		parentType = s.targetField.TypeName
	} else if sourceMap, ok := source.(map[string]interface{}); ok && sourceMap != nil {
		// This is a nested field - get type from source.__typename
		if typename, ok := sourceMap["__typename"].(string); ok && typename != "" {
			parentType = typename
			
			// Check if there's a field name in the input (might be present for fields with arguments)
			if fieldNameFromInput, ok := inputData["fieldName"].(string); ok && fieldNameFromInput != "" {
				fieldName = fieldNameFromInput
				fmt.Printf("  -> Using fieldName from input: %s\n", fieldName)
			} else {
				// For nested fields, try each field resolver for this type
				for key, info := range s.fieldMap {
					fmt.Printf("  fieldMap[%s] = {FieldName: %s, ParentType: %s}\n", key, info.FieldName, info.ParentType)
					if info.ParentType == parentType {
						fieldName = info.FieldName
						fmt.Printf("  -> Using fieldName from fieldMap: %s\n", fieldName)
						break
					}
				}
			}
		}
	} else {
		// Root field without targetField set - try each Query resolver until one succeeds
		// This is a fallback when we can't determine the field from planning
		// Fallback: try all Query resolvers (used when field wasn't extracted during planning)
		return s.tryAllQueryResolvers(ctx, arguments, out)
	}
	
	// Check for introspection queries
	if parentType == "Query" && (fieldName == "__schema" || fieldName == "__type") {
		return s.handleIntrospection(ctx, fieldName, arguments, out)
	}
	
	if fieldName == "" || parentType == "" {
		return fmt.Errorf("could not determine field to resolve (input keys: %v, source type: %T)", getMapKeys(inputData), source)
	}
	
	var resolveFn types.FieldResolveFn
	var found bool
	
	if parentType == "Query" {
		resolveFn, found = s.resolvers.GetQueryResolver(fieldName)
	} else if parentType == "Mutation" {
		resolveFn, found = s.resolvers.GetMutationResolver(fieldName)
	} else {
		// For type fields (nested fields), first try explicit type resolver
		resolveFn, found = s.resolvers.GetTypeResolver(parentType, fieldName)
		// If no explicit resolver, use auto-resolution via DefaultFieldResolver
		if !found {
			resolveFn = resolver.DefaultFieldResolver
			found = true
		}
	}

	if !found {
		return fmt.Errorf("no resolver found for %s.%s", parentType, fieldName)
	}

	// Create ResolveParams
	params := types.ResolveParams{
		Source:  source,
		Args:    arguments,
		Context: ctx,
		Info: types.ResolveInfo{
			FieldName:  fieldName,
			ParentType: parentType,
		},
	}

	// Call the resolver
	result, err := resolveFn(params)
	if err != nil {
		return fmt.Errorf("resolver error for %s.%s: %w", parentType, fieldName, err)
	}

	// For root Query/Mutation fields that return objects, we need to resolve nested fields
	// that have explicit resolvers BEFORE marshaling to JSON
	if parentType == "Query" || parentType == "Mutation" {
		result = s.resolveNestedFieldsWithResolvers(ctx, result, parentType)
	}

	// Serialize result to JSON
	// For root Query/Mutation fields that return objects, we return the object directly.
	// graphql-go-tools will extract nested fields from this object.
	var resultBytes []byte
	
	// For root Query/Mutation fields, we need to wrap the result in an object with the field name
	// This matches what graphql-go-tools expects for PostProcessingConfiguration
	if parentType == "Query" || parentType == "Mutation" {
		// Wrap the result in an object: {"viewer": <User object>}
		wrappedResult := map[string]interface{}{
			fieldName: result,
		}
		resultBytes, err = json.Marshal(wrappedResult)
		if err != nil {
			return fmt.Errorf("failed to marshal wrapped result: %w", err)
		}
	} else {
		// For nested fields, return the result directly
		resultBytes, err = json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
	}
	
	
	// Write the JSON bytes directly to the buffer
	_, err = out.Write(resultBytes)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// handleIntrospection generates introspection data for __schema and __type queries
func (s *RocketSource) handleIntrospection(ctx context.Context, fieldName string, arguments map[string]interface{}, out *bytes.Buffer) error {
	if s.schema == nil {
		return fmt.Errorf("schema not available for introspection")
	}
	
	
	// Use graphql-go-tools introspection generator
	generator := introspection.NewGenerator()
	introspectionData := &introspection.Data{}
	var report operationreport.Report
	
	// Generate introspection data from schema
	generator.Generate(s.schema, &report, introspectionData)
	if report.HasErrors() {
		return fmt.Errorf("failed to generate introspection data: %v", report.Error())
	}
	
	var result interface{}
	
	if fieldName == "__schema" {
		// Use the Schema from introspection data
		result = introspectionData.Schema
	} else if fieldName == "__type" {
		// Get type name from arguments
		typeName, ok := arguments["name"].(string)
		if !ok {
			return fmt.Errorf("__type query requires 'name' argument")
		}
		
		// Find type in introspection data using Schema.TypeByName
		typeData := introspectionData.Schema.TypeByName(typeName)
		if typeData == nil {
			return fmt.Errorf("type '%s' not found", typeName)
		}
		result = typeData
	} else {
		return fmt.Errorf("unknown introspection field: %s", fieldName)
	}
	
	// Wrap result in object with field name (for root Query fields)
	wrappedResult := map[string]interface{}{
		fieldName: result,
	}
	
	resultBytes, err := json.Marshal(wrappedResult)
	if err != nil {
		return fmt.Errorf("failed to marshal introspection result: %w", err)
	}
	
	_, err = out.Write(resultBytes)
	if err != nil {
		return fmt.Errorf("failed to write introspection response: %w", err)
	}
	
	return nil
}

// resolveNestedFieldsWithResolvers resolves nested fields that have explicit resolvers
// before marshaling to JSON. This ensures computed/lookup fields are included in the response.
func (s *RocketSource) resolveNestedFieldsWithResolvers(ctx context.Context, source interface{}, parentQueryField string) interface{} {
	if source == nil {
		return source
	}
	
	// Determine the type of the source object
	var typeName string
	
	// Try to get the type name from the source
	// For structs, we can use reflection to get the type name
	sourceValue := reflect.ValueOf(source)
	if sourceValue.Kind() == reflect.Ptr {
		if sourceValue.IsNil() {
			return source
		}
		sourceValue = sourceValue.Elem()
	}
	
	if sourceValue.Kind() == reflect.Struct {
		typeName = sourceValue.Type().Name()
	} else if sourceMap, ok := source.(map[string]interface{}); ok {
		// For maps, try to get __typename
		if tn, ok := sourceMap["__typename"].(string); ok {
			typeName = tn
		}
	}
	
	if typeName == "" {
		return source
	}
	
	// Check if this type has any explicit resolvers
	typeResolvers, hasResolvers := s.resolvers.Types[typeName]
	if !hasResolvers || len(typeResolvers) == 0 {
		return source
	}
	
	// Convert source to a map that we can augment
	var resultMap map[string]interface{}
	
	// Marshal and unmarshal to get a map representation
	sourceBytes, err := json.Marshal(source)
	if err != nil {
		return source
	}
	
	if err := json.Unmarshal(sourceBytes, &resultMap); err != nil {
		return source
	}
	
	// Auto-inject __typename for federation compatibility
	resultMap["__typename"] = typeName

	// Now resolve each field that has an explicit resolver
	for fieldName, resolveFn := range typeResolvers {
		
		// Use an empty map instead of nil to prevent panics when resolvers check for arguments
		// Resolvers that require arguments should check if the args map is empty and return nil
		params := types.ResolveParams{
			Source:  source,  // Pass the original source to the resolver
			Args:    make(map[string]interface{}), // Empty map to prevent nil panic
			Context: ctx,
			Info: types.ResolveInfo{
				FieldName:  fieldName,
				ParentType: typeName,
			},
		}
		
		// Use panic recovery to handle any panics gracefully
		func() {
			defer func() {
				if r := recover(); r != nil {
					// If resolver panics (e.g., trying to access args), skip it
					// Fields with required arguments will be resolved separately by graphql-go-tools
					fmt.Printf("WARNING: Resolver for %s.%s panicked: %v. Skipping.\n", typeName, fieldName, r)
				}
			}()
			
			result, err := resolveFn(params)
			if err == nil && result != nil {
				// Only add the field if resolution succeeded and result is not nil
				// This allows resolvers to return nil to indicate "skip this field"
				resultMap[fieldName] = result
			}
		}()
	}
	
	return resultMap
}

// tryAllQueryResolvers attempts to call each Query resolver and returns the FIRST successful one
// This is a fallback for when we can't determine the field from planning
// NOTE: This only works correctly for single-field queries. Multi-field queries need proper field detection.
func (s *RocketSource) tryAllQueryResolvers(ctx context.Context, arguments map[string]interface{}, out *bytes.Buffer) error {
	// Collect all query field names and sort for deterministic order
	queryFields := make([]string, 0, len(s.resolvers.Query))
	for queryField := range s.resolvers.Query {
		queryFields = append(queryFields, queryField)
	}
	sort.Strings(queryFields)
	
	// Try each Query resolver in sorted order and return the FIRST success
	for _, queryField := range queryFields {
		resolveFn := s.resolvers.Query[queryField]
		params := types.ResolveParams{
			Source:  nil,
			Args:    arguments,
			Context: ctx,
			Info: types.ResolveInfo{
				FieldName:  queryField,
				ParentType: "Query",
			},
		}
		
		result, err := resolveFn(params)
		if err != nil {
			// This resolver failed, try next
			// This resolver failed, try next
			continue
		}
		
		// Success! Return this result
		// This resolver succeeded, return its result
		wrappedResult := map[string]interface{}{
			queryField: result,
		}
		resultBytes, marshalErr := json.Marshal(wrappedResult)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal result: %w", marshalErr)
		}
		_, writeErr := out.Write(resultBytes)
		return writeErr
	}
	
	return fmt.Errorf("no Query resolver succeeded")
}

func (s *RocketSource) LoadWithFiles(ctx context.Context, input []byte, files []*httpclient.FileUpload, out *bytes.Buffer) error {
	// File uploads not yet supported
	return s.Load(ctx, input, out)
}

