package rocket

import (
	"context"

	"github.com/jensneuse/abstractlogger"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
)

// RocketDataSource bridges Rocket's resolver pattern to graphql-go-tools DataSource
// This allows Rocket to work with graphql-go-tools while maintaining its developer-friendly API
type RocketDataSource struct {
	id        string
	name      string
	resolvers *ResolverRegistry
	schema    *ast.Document
	factory   *RocketDataSourceFactory
}

// RocketDataSourceFactory creates RocketDataSource instances
type RocketDataSourceFactory struct {
	resolvers        *ResolverRegistry
	schema           *ast.Document
	objectFetchConfig interface{} // Store reference to objectFetchConfiguration to pass field coord
}

// NewRocketDataSourceFactory creates a new factory for Rocket DataSources
func NewRocketDataSourceFactory(resolvers *ResolverRegistry, schema *ast.Document) *RocketDataSourceFactory {
	return &RocketDataSourceFactory{
		resolvers: resolvers,
		schema:    schema,
	}
}

// DataSourceFactory methods required by graphql-go-tools

func (f *RocketDataSourceFactory) Planner(_ abstractlogger.Logger) plan.DataSourcePlanner[interface{}] {
	return &RocketPlanner{
		factory: f,
	}
}

func (f *RocketDataSourceFactory) Context() context.Context {
	return context.Background()
}

func (f *RocketDataSourceFactory) UpstreamSchema(config plan.DataSourceConfiguration[interface{}]) (*ast.Document, bool) {
	return f.schema, true
}

func (f *RocketDataSourceFactory) PlanningBehavior() plan.DataSourcePlanningBehavior {
	return plan.DataSourcePlanningBehavior{}
}

// ToDataSourceConfiguration converts the factory to a DataSourceConfiguration
// This allows it to be used in plan.Configuration.DataSources
func (f *RocketDataSourceFactory) ToDataSourceConfiguration(id, name string) (plan.DataSourceConfiguration[interface{}], error) {
	// Create metadata and populate RootNodes with fields Rocket can resolve
	metadata := &plan.DataSourceMetadata{}
	
	// Populate RootNodes with Query and Mutation fields that Rocket can resolve
	// This tells the planner which fields this DataSource handles
	for queryName := range f.resolvers.Query {
		// Add Query field to RootNodes
		metadata.RootNodes = append(metadata.RootNodes, plan.TypeField{
			TypeName:   "Query",
			FieldNames: []string{queryName},
		})
	}
	
	for mutationName := range f.resolvers.Mutation {
		// Add Mutation field to RootNodes
		metadata.RootNodes = append(metadata.RootNodes, plan.TypeField{
			TypeName:   "Mutation",
			FieldNames: []string{mutationName},
		})
	}
	
	// For type resolvers, we need to add them as ChildNodes
	// ChildNodes are fields that can be resolved when we have the parent object
	for typeName, typeResolvers := range f.resolvers.Types {
		var fieldNames []string
		for fieldName := range typeResolvers {
			fieldNames = append(fieldNames, fieldName)
		}
		if len(fieldNames) > 0 {
			metadata.ChildNodes = append(metadata.ChildNodes, plan.TypeField{
				TypeName:   typeName,
				FieldNames: fieldNames,
			})
		}
	}
	
	// For auto-resolution: Add all fields of types returned by Query fields to ChildNodes
	// This allows Rocket to auto-resolve nested fields even without explicit resolvers
	// We'll scan the schema to find types that Query fields return
	// For now, we'll add all object types to ChildNodes if they're not already there
	// This is a simplified approach - in practice, we'd want to track which types
	// are actually returned by Query fields
	if f.schema != nil {
		for i := range f.schema.ObjectTypeDefinitions {
			typeName := f.schema.ObjectTypeDefinitionNameString(i)
			// Skip Query and Mutation (they're root nodes)
			if typeName == "Query" || typeName == "Mutation" {
				continue
			}
			
			// Check if this type is already in ChildNodes
			alreadyExists := false
			for _, childNode := range metadata.ChildNodes {
				if childNode.TypeName == typeName {
					alreadyExists = true
					break
				}
			}
			
			// If not already there, add all fields for auto-resolution
			if !alreadyExists {
				objDef := f.schema.ObjectTypeDefinitions[i]
				fieldsRefs := objDef.FieldsDefinition.Refs
				if len(fieldsRefs) > 0 {
					var fieldNames []string
					for _, fieldRef := range fieldsRefs {
						fieldName := f.schema.FieldDefinitionNameString(fieldRef)
						fieldNames = append(fieldNames, fieldName)
					}
					if len(fieldNames) > 0 {
						metadata.ChildNodes = append(metadata.ChildNodes, plan.TypeField{
							TypeName:   typeName,
							FieldNames: fieldNames,
						})
					}
				}
			}
		}
	}
	
	// Initialize metadata to avoid nil pointer dereference
	if err := metadata.Init(); err != nil {
		return nil, err
	}

	return plan.NewDataSourceConfigurationWithName[interface{}](
		id,
		name,
		f,                // factory implements PlannerFactory
		metadata,         // metadata with RootNodes populated
		interface{}(nil), // custom config - we don't need it
	)
}

