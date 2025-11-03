package rocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/datasource/httpclient"
)

// RocketSource implements resolve.DataSource for Rocket resolvers
// This is where we actually execute Rocket's resolver functions
type RocketSource struct {
	resolvers     *ResolverRegistry
	fieldMap      map[string]fieldInfo
	fieldCoord    *fieldCoordinate // Store field coordinate for this fetch (set during ConfigureFetch)
	fallbackField string           // Fallback: store field name if coordinate not available
}

// fieldCoordinate stores the GraphQL field coordinate (TypeName + FieldName)
type fieldCoordinate struct {
	TypeName  string
	FieldName string
}

// fetchInput represents the input structure passed to Load
type fetchInput struct {
	FieldName  string                 `json:"fieldName"`
	ParentType string                 `json:"parentType"`
	Arguments  map[string]interface{} `json:"arguments"`
	Source     interface{}            `json:"source"`
}

// Load executes Rocket resolvers and returns the result
func (s *RocketSource) Load(ctx context.Context, input []byte, out *bytes.Buffer) error {
	// Load is called by graphql-go-tools to execute the resolver
	
	// Try to parse the input JSON
	var inputData map[string]interface{}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &inputData); err != nil {
			// Input might not be valid JSON, continue with empty map
			inputData = make(map[string]interface{})
		}
	}
	
	// Extract field information from context or input
	// For Query/Mutation fields, we need to determine which field is being resolved
	// We can use the fieldMap to look up based on context, or extract from the fetch info
	// For now, let's try to extract from the input data or use a fallback
	
	// Get arguments and source from input
	arguments := make(map[string]interface{})
	if args, ok := inputData["arguments"].(map[string]interface{}); ok {
		arguments = args
	}
	
	source := inputData["source"]
	
	// Try to determine field from context or use a workaround
	// Since we don't have direct access to field name in Load, we'll need to
	// extract it from the fetch path or use a different approach
	// For now, let's try to look it up from the context if available
	
	// Get field info from context - we'll need to store it during fetch configuration
	// For now, let's try a different approach: use FetchInfo from the context
	// or extract field information from the RootFields in FetchInfo
	// Since we don't have direct access, let's try to extract from the path or use fieldMap
	
	// For Query fields, source should be nil/empty and we can identify from the fieldMap
	// Since Load is called per fetch, and each fetch corresponds to one field,
	// we need to match the field somehow. Let's use a workaround:
	// For Query fields, check which resolver matches (this won't work if multiple have same args)
	
	// Better approach: We can store field coordinate in the DataSource during planning
	// and retrieve it here. For now, let's try to get it from FetchInfo if available
	
	// Since we're calling Load, and Load is called with a specific fetch,
	// we need to identify which field this fetch is for
	// The simplest approach: use the fieldMap to find matching fields
	// But we need to know which field is being resolved
	
	// Use stored field coordinate if available
	var fieldName, parentType string
	var resolver FieldResolveFn
	var found bool
	
	if s.fieldCoord != nil {
		// We have the field coordinate - use it directly
		fieldName = s.fieldCoord.FieldName
		parentType = s.fieldCoord.TypeName
	} else {
		// Fallback: Try to determine field from source
		// For Query fields, source is nil or empty
		isQueryField := false
		if source == nil {
			isQueryField = true
		} else if sourceMap, ok := source.(map[string]interface{}); ok && len(sourceMap) == 0 {
			isQueryField = true
		}
		
		if isQueryField {
			// This is a Query field - try to find matching Query resolver
			// For now, if we have only one Query resolver, use that
			if len(s.resolvers.Query) == 1 {
				// Get the only Query resolver
				for name := range s.resolvers.Query {
					fieldName = name
					parentType = "Query"
					resolver, found = s.resolvers.GetQueryResolver(name)
					break
				}
			} else {
				return fmt.Errorf("cannot determine Query field from input: have %d Query resolvers, need field coordinate", len(s.resolvers.Query))
			}
		} else {
			// This is a type field - source should contain the parent object
			// We need field info to know which field to resolve
			return fmt.Errorf("type field resolution not yet implemented - need field coordinate")
		}
		
		if !found {
			return fmt.Errorf("no resolver found for %s.%s", parentType, fieldName)
		}
	}
	
	// Now look up the resolver using the field coordinate
	if parentType == "Query" {
		resolver, found = s.resolvers.GetQueryResolver(fieldName)
	} else if parentType == "Mutation" {
		resolver, found = s.resolvers.GetMutationResolver(fieldName)
	} else {
		resolver, found = s.resolvers.GetTypeResolver(parentType, fieldName)
	}

	if !found {
		return fmt.Errorf("no resolver found for %s.%s", parentType, fieldName)
	}

	// Create ResolveParams
	params := ResolveParams{
		Source:  source,
		Args:    arguments,
		Context: ctx,
		Info: ResolveInfo{
			FieldName:  fieldName,
			ParentType: parentType,
		},
	}

	// Call the resolver
	result, err := resolver(params)
	if err != nil {
		return fmt.Errorf("resolver error for %s.%s: %w", parentType, fieldName, err)
	}

	// Serialize result to JSON
	// For root Query/Mutation fields, mergeResult expects an object when items is empty
	// We need to wrap the field value in an object with the field name
	// For nested fields (items not empty), we return the value directly
	var resultBytes []byte
	if parentType == "Query" || parentType == "Mutation" {
		// Root Query/Mutation field - wrap in object with field name
		responseObj := map[string]interface{}{
			fieldName: result,
		}
		resultBytes, err = json.Marshal(responseObj)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
	} else {
		// Nested field - return value directly
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

func (s *RocketSource) LoadWithFiles(ctx context.Context, input []byte, files []*httpclient.FileUpload, out *bytes.Buffer) error {
	// File uploads not yet supported
	return s.Load(ctx, input, out)
}

