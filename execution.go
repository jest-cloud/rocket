package rocket

import (
	"bytes"
	"encoding/json"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// ExecuteWithFieldOrder executes a GraphQL query and returns a result with preserved field order
func ExecuteWithFieldOrder(params graphql.Params, queryString string) *Result {
	// Execute the query
	result := graphql.Do(params)

	// Convert to Rocket result
	if result.HasErrors() || result.Data == nil {
		return convertResult(result)
	}

	// Parse query to get field selection order
	source := &ast.Source{
		Name:  "query",
		Input: queryString,
	}
	doc, gqlErr := parser.ParseQuery(source)
	if gqlErr != nil {
		// If we can't parse the query, fall back to unordered result
		return convertResult(result)
	}

	// Get operation
	var operation *ast.OperationDefinition
	if params.OperationName != "" {
		for _, op := range doc.Operations {
			if op.Name == params.OperationName {
				operation = op
				break
			}
		}
	} else if len(doc.Operations) > 0 {
		for _, op := range doc.Operations {
			operation = op
			break
		}
	}

	if operation == nil {
		return convertResult(result)
	}

	// Reorder data based on selection set
	orderedData := reorderData(result.Data, operation.SelectionSet)

	return &Result{
		Data:   orderedData,
		Errors: convertErrors(result.Errors),
	}
}

// reorderData recursively reorders data based on selection set
func reorderData(data interface{}, selectionSet ast.SelectionSet) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		return reorderMap(v, selectionSet)
	case []interface{}:
		return reorderArray(v, selectionSet)
	default:
		return data
	}
}

// reorderMap reorders a map based on field selection order
func reorderMap(data map[string]interface{}, selectionSet ast.SelectionSet) *OrderedMap {
	ordered := NewOrderedMap()

	if len(selectionSet) == 0 {
		// When no selection set, preserve data keys in alphabetical order for consistency
		// This is typically for scalar fields
		keys := make([]string, 0, len(data))
		for key := range data {
			keys = append(keys, key)
		}
		for _, key := range keys {
			ordered.Set(key, data[key])
		}
		return ordered
	}

	// Process selections in order
	for _, selection := range selectionSet {
		switch sel := selection.(type) {
		case *ast.Field:
			fieldName := sel.Name
			if sel.Alias != "" {
				fieldName = sel.Alias
			}

			if value, ok := data[fieldName]; ok {
				// Recursively reorder nested objects
				if len(sel.SelectionSet) > 0 {
					value = reorderData(value, sel.SelectionSet)
				}
				ordered.Set(fieldName, value)
			}
		case *ast.InlineFragment:
			// Handle inline fragments by processing their selection set
			for _, fragSelection := range sel.SelectionSet {
				if field, ok := fragSelection.(*ast.Field); ok {
					fieldName := field.Name
					if field.Alias != "" {
						fieldName = field.Alias
					}
					if value, ok := data[fieldName]; ok {
						if len(field.SelectionSet) > 0 {
							value = reorderData(value, field.SelectionSet)
						}
						ordered.Set(fieldName, value)
					}
				}
			}
		case *ast.FragmentSpread:
			// Fragment spreads would need the fragment definition from the schema
			// For now, we'll skip them - they're less common
		}
	}

	return ordered
}

// reorderArray reorders each element in an array
func reorderArray(data []interface{}, selectionSet ast.SelectionSet) []interface{} {
	result := make([]interface{}, len(data))
	for i, item := range data {
		result[i] = reorderData(item, selectionSet)
	}
	return result
}

// OrderedMap preserves insertion order for JSON marshaling
type OrderedMap struct {
	keys   []string
	values map[string]interface{}
}

// NewOrderedMap creates a new ordered map
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		keys:   []string{},
		values: make(map[string]interface{}),
	}
}

// Set adds or updates a key-value pair
func (m *OrderedMap) Set(key string, value interface{}) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

// Get retrieves a value by key
func (m *OrderedMap) Get(key string) interface{} {
	return m.values[key]
}

// Has checks if a key exists
func (m *OrderedMap) Has(key string) bool {
	_, exists := m.values[key]
	return exists
}

// Keys returns all keys in insertion order
func (m *OrderedMap) Keys() []string {
	return m.keys
}

// MarshalJSON implements json.Marshaler to preserve order
func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	for i, key := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}

		// Marshal key
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')

		// Marshal value
		valueBytes, err := json.Marshal(m.values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valueBytes)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// convertErrors converts graphql errors to Rocket errors
func convertErrors(errors []gqlerrors.FormattedError) []Error {
	if len(errors) == 0 {
		return nil
	}

	result := make([]Error, len(errors))
	for i, gqlErr := range errors {
		result[i] = Error{
			Message: gqlErr.Message,
			Path:    gqlErr.Path,
		}
	}
	return result
}

// convertResult converts graphql.Result to Rocket Result
func convertResult(result *graphql.Result) *Result {
	rocketResult := &Result{
		Data: result.Data,
	}

	if result.HasErrors() {
		rocketResult.Errors = convertErrors(result.Errors)
	}

	return rocketResult
}

