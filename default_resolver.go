package rocket

import (
	"reflect"
	"strings"
	"time"
)

// DefaultFieldResolver automatically resolves struct fields to GraphQL fields
// This is similar to Apollo's default field resolver in TypeScript
// It uses reflection to map GraphQL field names to Go struct fields
func DefaultFieldResolver(p ResolveParams) (interface{}, error) {
	if p.Source == nil {
		return nil, nil
	}

	fieldName := p.Info.FieldName

	// Use reflection to get the field value
	sourceValue := reflect.ValueOf(p.Source)

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

