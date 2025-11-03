package rocket

import (
	"fmt"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/resolve"
)

// RocketPlanner implements plan.DataSourcePlanner for Rocket resolvers
// It configures how fields are resolved using Rocket's resolver pattern
type RocketPlanner struct {
	id                int
	factory           *RocketDataSourceFactory
	fieldMap          map[string]fieldInfo // Maps field key to field info
	objectFetchConfig *objectFetchConfiguration // Store reference to share with ConfigureFetch
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

func (p *RocketPlanner) Register(visitor *plan.Visitor, configuration plan.DataSourceConfiguration[interface{}], _ plan.DataSourcePlannerConfiguration) error {
	// Initialize field map
	p.fieldMap = make(map[string]fieldInfo)

	// Walk through the schema to find fields Rocket can resolve
	schema := visitor.Definition
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
			return ok
		}
		if parentTypeName == "Mutation" {
			_, ok := p.factory.resolvers.GetMutationResolver(fieldName)
			return ok
		}
		_, ok := p.factory.resolvers.GetTypeResolver(parentTypeName, fieldName)
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

			// Check if Rocket has a resolver for this field
			// Note: We also register fields without explicit resolvers if they're in types
			// that Rocket can resolve (for auto-resolution support)
			if hasResolver(typeName, fieldName) {
				fieldKey := fmt.Sprintf("%s.%s", typeName, fieldName)
				p.fieldMap[fieldKey] = fieldInfo{
					FieldName:  fieldName,
					ParentType: typeName,
					IsQuery:    typeName == "Query",
					IsMutation: typeName == "Mutation",
				}
			}
		}
	}

	return nil
}

func (p *RocketPlanner) ConfigureFetch() resolve.FetchConfiguration {
	// Configure fetch to use our custom Source
	// The Input template uses {{.object.field}} syntax for graphql-go-tools
	// Available variables: {{.object.arguments}}, {{.object.source}}
	// We'll pass a minimal template with arguments and source
	// Field identification will be done in Load based on the fetch context
	// Input template uses {{.object.field}} syntax which gets converted to ObjectVariable
	// For root Query fields, source will be null, arguments might be empty object
	// Use a simple template that includes both arguments and source
	inputTemplate := `{"arguments":{{.object.arguments}},"source":{{.object.source}}}`

	// Try to get field coordinate from objectFetchConfiguration if available
	// Note: This may be nil if graphql-go-tools uses its own internal type
	var fieldCoord *fieldCoordinate
	if p.objectFetchConfig != nil && len(p.objectFetchConfig.rootFields) > 0 {
		// Use the first rootField - for Query fields, there should be one coordinate
		coord := p.objectFetchConfig.rootFields[0]
		fieldCoord = &fieldCoordinate{
			TypeName:  coord.TypeName,
			FieldName: coord.FieldName,
		}
		// Field coordinate extracted from rootFields
	} else {
		// RootFields not available in our custom type - will extract from FetchInfo during execution
	}

	return resolve.FetchConfiguration{
		Input:      inputTemplate,
		DataSource: &RocketSource{
			resolvers:  p.factory.resolvers,
			fieldMap:   p.fieldMap,
			fieldCoord: fieldCoord,
		},
		PostProcessing: resolve.PostProcessingConfiguration{
			// SelectResponseDataPath: path to extract data from Load response
			// For root Query fields, we return the field value directly
			// The path is empty [] because we return the value directly, not wrapped in an object
			SelectResponseDataPath: []string{},
			// MergePath: path where to place the result in the response
			// For root Query fields, we place it at the field name
			// This is handled automatically by graphql-go-tools based on the field name
			MergePath: []string{},
		},
	}
}

// ObjectFetchConfiguration is called during planning to get field-specific configuration
// This method returns a pointer to the same object, so graphql-go-tools can populate rootFields
func (p *RocketPlanner) ObjectFetchConfiguration() *objectFetchConfiguration {
	if p.objectFetchConfig == nil {
		p.objectFetchConfig = &objectFetchConfiguration{
			fieldMap: p.fieldMap,
		}
	}
	return p.objectFetchConfig
}

type objectFetchConfiguration struct {
	fieldMap          map[string]fieldInfo
	rootFields        []resolve.GraphCoordinate // Populated by graphql-go-tools with field coordinates
	currentFieldCoord *fieldCoordinate
}

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

