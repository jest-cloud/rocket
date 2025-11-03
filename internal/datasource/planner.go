package datasource

import (
	"fmt"
	"strings"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/resolve"
)

// RocketPlanner implements plan.DataSourcePlanner for Rocket resolvers
// It configures how fields are resolved using Rocket's resolver pattern
type RocketPlanner struct {
	id            int
	factory       *RocketDataSourceFactory
	fieldMap      map[string]fieldInfo // Maps field key to field info
	plannerConfig plan.DataSourcePlannerConfiguration
	// Track which fields we've configured fetches for
	configuredFields []string
	// Store the operation and definition from the visitor
	operation *ast.Document
	definition *ast.Document
}

// fieldInfo stores information about a field Rocket can resolve
type fieldInfo struct {
	FieldName  string
	ParentType string
	IsQuery    bool
	IsMutation bool
}

func (p *RocketPlanner) SetID(id int) {
	p.id = id
}

func (p *RocketPlanner) ID() int {
	return p.id
}

func (p *RocketPlanner) DownstreamResponseFieldAlias(downstreamFieldRef int) (alias string, exists bool) {
	// Not needed for Rocket - we resolve fields directly
	return "", false
}

func (p *RocketPlanner) Register(visitor *plan.Visitor, configuration plan.DataSourceConfiguration[interface{}], dataSourcePlannerConfig plan.DataSourcePlannerConfiguration) error {
	// Store planner config for later use
	p.plannerConfig = dataSourcePlannerConfig
	
	// CRITICAL: Reset configuredFields for each new query planning session
	// The planner instance is reused across multiple queries, so we need to reset state
	p.configuredFields = nil
	
	// Store operation and definition from visitor for use in ConfigureFetch
	p.operation = visitor.Operation
	p.definition = visitor.Definition
	
	// Extract the requested field name from the operation if available
	var requestedField string
	if visitor.Operation != nil && len(visitor.Operation.OperationDefinitions) > 0 {
		opDef := visitor.Operation.OperationDefinitions[0]
		if opDef.SelectionSet >= 0 && opDef.SelectionSet < len(visitor.Operation.SelectionSets) {
			selectionSet := visitor.Operation.SelectionSets[opDef.SelectionSet]
			if len(selectionSet.SelectionRefs) > 0 {
				selRef := selectionSet.SelectionRefs[0]
				if selRef >= 0 && selRef < len(visitor.Operation.Selections) {
					selection := visitor.Operation.Selections[selRef]
					if selection.Kind == ast.SelectionKindField {
						fieldRef := selection.Ref
						if fieldRef >= 0 && fieldRef < len(visitor.Operation.Fields) {
							requestedField = visitor.Operation.FieldNameString(fieldRef)
							// Successfully extracted field from operation
						}
					}
				}
			}
		}
	}
	
	// Store the requested field for use in ConfigureFetch
	if requestedField != "" {
		p.configuredFields = []string{requestedField}
	}
	
	// If we couldn't extract from operation (because it's nil during planner creation),
	// use the requestedFields that were passed to the factory
	if requestedField == "" && len(p.factory.requestedFields) > 0 {
		requestedField = p.factory.requestedFields[0]
		p.configuredFields = []string{requestedField}
	}
	
	// Initialize field map
	p.fieldMap = make(map[string]fieldInfo)

	// Walk through the schema to find fields Rocket can resolve
	// NOTE: visitor.Definition might be nil at this point, so use factory.schema instead
	schema := p.factory.schema
	if schema == nil {
		return nil
	}
	

	// Helper to check if Rocket has a resolver for a field
	hasResolver := func(parentTypeName, fieldName string) bool {
		if p.factory == nil || p.factory.resolvers == nil {
			return false
		}
		if parentTypeName == "Query" {
			_, ok := p.factory.resolvers.GetQueryResolver(fieldName)
			if ok {
			}
			return ok
		}
		if parentTypeName == "Mutation" {
			_, ok := p.factory.resolvers.GetMutationResolver(fieldName)
			if ok {
			}
			return ok
		}
		_, ok := p.factory.resolvers.GetTypeResolver(parentTypeName, fieldName)
		if ok {
		}
		return ok
	}

	// Walk through all object type definitions
	if len(schema.ObjectTypeDefinitions) == 0 {
		return nil
	}
	
		for i := range schema.ObjectTypeDefinitions {
		typeName := schema.ObjectTypeDefinitionNameString(i)

		// Walk through fields of this type
		objDef := schema.ObjectTypeDefinitions[i]
		fieldsRefs := objDef.FieldsDefinition.Refs
		
		// Skip if no fields
		if len(fieldsRefs) == 0 {
			continue
		}
		
		for _, fieldRef := range fieldsRefs {
			fieldName := schema.FieldDefinitionNameString(fieldRef)

			// Handle introspection fields (__schema, __type) - we'll implement them in RocketSource
			// __typename is handled automatically by graphql-go-tools
			isIntrospection := strings.HasPrefix(fieldName, "__")
			if isIntrospection && fieldName != "__schema" && fieldName != "__type" {
				// Skip __typename and other introspection fields - they're handled automatically
				if typeName == "Query" {
				}
				continue
			}

			// Register introspection fields for Query type
			if typeName == "Query" && (fieldName == "__schema" || fieldName == "__type") {
				fieldKey := typeName + "." + fieldName
				p.fieldMap[fieldKey] = fieldInfo{
					FieldName:  fieldName,
					ParentType: typeName,
					IsQuery:    true,
					IsMutation: false,
				}
				continue
			}

			// Check if Rocket has a resolver for this field
			// Note: We also register fields without explicit resolvers if they're in types
			// that Rocket can resolve (for auto-resolution support)
			if hasResolver(typeName, fieldName) {
				fieldKey := typeName + "." + fieldName
				p.fieldMap[fieldKey] = fieldInfo{
					FieldName:  fieldName,
					ParentType: typeName,
					IsQuery:    typeName == "Query",
					IsMutation: typeName == "Mutation",
				}
			} else if typeName != "Query" && typeName != "Mutation" {
				// For nested types (non-Query/Mutation), register all fields for auto-resolution
				// This allows graphql-go-tools to know Rocket can handle these fields
				// IMPORTANT: We need to register ALL fields, not just ones with explicit resolvers
				// This tells the planner that Rocket can handle any field on these types
				fieldKey := typeName + "." + fieldName
				if _, exists := p.fieldMap[fieldKey]; !exists {
					p.fieldMap[fieldKey] = fieldInfo{
						FieldName:  fieldName,
						ParentType: typeName,
						IsQuery:    false,
						IsMutation: false,
					}
				}
			}
		}
	}

	return nil
}

// generateInputTemplate creates an Input template that extracts field arguments
// For a field like createUser(input: CreateUserInput!), it generates:
// {"arguments": {"input": {{.arguments.input}}}, "source": {{.object}}}
func (p *RocketPlanner) generateInputTemplate(targetField *fieldCoordinate) string {
	if targetField == nil || p.factory.schema == nil {
		return `{}`
	}
	
	// Find the field definition in the schema
	typeName := targetField.TypeName
	fieldName := targetField.FieldName
	
	// Find the type definition (Query, Mutation, or custom type)
	var fieldDef *ast.FieldDefinition
	for i := range p.factory.schema.ObjectTypeDefinitions {
		typeDef := &p.factory.schema.ObjectTypeDefinitions[i]
		if p.factory.schema.ObjectTypeDefinitionNameString(i) == typeName {
			// Find the field
			for _, fieldRef := range typeDef.FieldsDefinition.Refs {
				if p.factory.schema.FieldDefinitionNameString(fieldRef) == fieldName {
					fieldDef = &p.factory.schema.FieldDefinitions[fieldRef]
					break
				}
			}
			break
		}
	}
	
	if fieldDef == nil || !fieldDef.HasArgumentsDefinitions {
		// No arguments, return empty object
		return `{}`
	}
	
	// Build template with arguments
	var argTemplates []string
	for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
		argName := p.factory.schema.InputValueDefinitionNameString(argRef)
		argTemplates = append(argTemplates, fmt.Sprintf(`"%s": {{.arguments.%s}}`, argName, argName))
	}
	
	if len(argTemplates) == 0 {
		return `{}`
	}
	
	return fmt.Sprintf(`{"arguments": {%s}, "source": {{.object}}}`, strings.Join(argTemplates, ", "))
}

func (p *RocketPlanner) ConfigureFetch() resolve.FetchConfiguration {
	// FINAL SOLUTION: Store the planner config path information in the RocketSource
	// so it can determine which field to resolve based on the path
	
	// ConfigureFetch is called during query planning to set up the fetch for this field
	
	var targetField *fieldCoordinate
	if len(p.configuredFields) > 0 {
		// Use the requested field we extracted during Register()
		fieldName := p.configuredFields[0]
		// Determine type based on operation type from factory
		typeName := "Query"
		if p.factory.operationType == ast.OperationTypeMutation {
			typeName = "Mutation"
		} else if p.factory.operationType == ast.OperationTypeSubscription {
			typeName = "Subscription"
		}
		targetField = &fieldCoordinate{
			TypeName:  typeName,
			FieldName: fieldName,
		}
		// Using field extracted from operation
	}
	
	rocketSource := &RocketSource{
		resolvers:   p.factory.resolvers,
		fieldMap:    p.fieldMap,
		schema:      p.factory.schema,
		targetField: targetField,
		// Store parent path to help with debugging
		parentPath: p.plannerConfig.ParentPath,
	}

	// Generate Input template based on field arguments in the schema
	// This extracts arguments from the query and passes them to Load()
	inputTemplate := p.generateInputTemplate(targetField)
	
	return resolve.FetchConfiguration{
		Input:      inputTemplate,
		DataSource: rocketSource,
		PostProcessing: resolve.PostProcessingConfiguration{
			SelectResponseDataPath: []string{},
			MergePath:              []string{},
		},
	}
}

// NOTE: We previously tried to implement ObjectFetchConfiguration() to extract field coordinates
// during planning, but this method is internal to graphql-go-tools' plannerConfiguration
// and is not meant to be overridden by custom DataSourcePlanners.
//
// Instead, we'll extract field information from the input template variables which are
// populated by graphql-go-tools during execution. The input template receives:
// - {{.object.arguments}} - the field arguments
// - {{.object.source}} - the parent object (for nested fields)
// - Other context that might contain field information
//
// For now, we'll use a simpler approach: encode the field information in the DataSource
// by creating separate DataSource instances for each field during ConfigureFetch().

func (p *RocketPlanner) ConfigureSubscription() plan.SubscriptionConfiguration {
	// Subscriptions not yet supported, but can be added later
	return plan.SubscriptionConfiguration{}
}

func (p *RocketPlanner) DataSourcePlanningBehavior() plan.DataSourcePlanningBehavior {
	return plan.DataSourcePlanningBehavior{
		MergeAliasedRootNodes:      false,
		OverrideFieldPathFromAlias: false,
	}
}

