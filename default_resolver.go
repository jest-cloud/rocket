package rocket

import (
	"reflect"
	"strings"
	"time"
)

// DefaultFieldResolver automatically resolves struct fields to GraphQL fields
// It uses reflection to map GraphQL field names to Go struct fields
// This provides a developer-friendly auto-resolution feature
func DefaultFieldResolver(p ResolveParams) (interface{}, error) {
	if p.Source == nil {
		return nil, nil
	}

	fieldName := p.Info.FieldName

	// Handle map[string]interface{} (e.g., from JSON unmarshaling or graphql-go conversions)
	if m, ok := p.Source.(map[string]interface{}); ok {
		if value, exists := m[fieldName]; exists {
			return value, nil
		}
		// Also try with capitalized key (in case graphql-go uses struct field names)
		pascalName := toPascalCase(fieldName)
		if value, exists := m[pascalName]; exists {
			return value, nil
		}
		return nil, nil
	}

	// Use reflection to get the field value
	sourceValue := reflect.ValueOf(p.Source)

	// Handle interface{} types that might wrap the actual struct
	// This is common when values come from slices or are passed through interface{}
	// We need to unwrap recursively until we get to the actual type
	for sourceValue.Kind() == reflect.Interface {
		if sourceValue.IsNil() {
			return nil, nil
		}
		// Get the underlying value
		underlying := sourceValue.Elem()
		
		// Check if underlying is a map
		if mapValue, ok := underlying.Interface().(map[string]interface{}); ok {
			if value, exists := mapValue[fieldName]; exists {
				return value, nil
			}
			// Also try with capitalized key
			pascalName := toPascalCase(fieldName)
			if value, exists := mapValue[pascalName]; exists {
				return value, nil
			}
			return nil, nil
		}
		
		// Continue unwrapping if it's still an interface
		sourceValue = underlying
		if sourceValue.Kind() != reflect.Interface {
			break
		}
	}

	// If it's a pointer, dereference it
	if sourceValue.Kind() == reflect.Ptr {
		if sourceValue.IsNil() {
			return nil, nil
		}
		sourceValue = sourceValue.Elem()
	}

	if sourceValue.Kind() != reflect.Struct {
		return nil, nil
	}

	// Try to find the field
	field := findField(sourceValue, fieldName)
	if !field.IsValid() {
		return nil, nil
	}

	value := field.Interface()

	// Handle special types
	switch v := value.(type) {
	case time.Time:
		return v.Format(time.RFC3339), nil
	case *time.Time:
		if v == nil {
			return nil, nil
		}
		return v.Format(time.RFC3339), nil
	}

	return value, nil
}

// findField finds a struct field by GraphQL field name
// Handles camelCase -> PascalCase conversion and json tag matching
func findField(structValue reflect.Value, fieldName string) reflect.Value {
	structType := structValue.Type()

	// Try exact match with PascalCase (GraphQL: firstName -> Go: FirstName)
	pascalName := toPascalCase(fieldName)
	field := structValue.FieldByName(pascalName)
	if field.IsValid() && field.CanInterface() {
		return field
	}

	// Try matching by json tag
	for i := 0; i < structType.NumField(); i++ {
		fieldType := structType.Field(i)
		
		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}
		
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag != "" {
			tagName := strings.Split(jsonTag, ",")[0]
			if tagName == fieldName {
				return structValue.Field(i)
			}
		}
		
		// Try bson tag as well
		bsonTag := fieldType.Tag.Get("bson")
		if bsonTag != "" {
			tagName := strings.Split(bsonTag, ",")[0]
			if tagName == fieldName && tagName != "-" && tagName != "_id" {
				return structValue.Field(i)
			}
		}
	}

	return reflect.Value{}
}

// toPascalCase converts camelCase to PascalCase
// Examples: firstName -> FirstName, id -> Id, orgID -> OrgID
func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	
	// Handle common acronyms
	switch s {
	case "id":
		return "ID"
	case "url":
		return "URL"
	case "api":
		return "API"
	}
	
	// Check if it ends with common acronyms
	if strings.HasSuffix(s, "Id") {
		return s[:len(s)-2] + "ID"
	}
	if strings.HasSuffix(s, "Url") {
		return s[:len(s)-3] + "URL"
	}
	if strings.HasSuffix(s, "Api") {
		return s[:len(s)-3] + "API"
	}
	
	// Standard camelCase to PascalCase
	return strings.ToUpper(s[:1]) + s[1:]
}

