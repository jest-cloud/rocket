package datasource

import (
	"context"
	"sort"
	"strings"

	"github.com/jensneuse/abstractlogger"
	"github.com/jest-cloud/rocket/internal/registry"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
)

// RocketDataSource bridges Rocket's resolver pattern to graphql-go-tools DataSource
// This allows Rocket to work with graphql-go-tools while maintaining its developer-friendly API
type RocketDataSource struct {
	id        string
	name      string
	resolvers *registry.ResolverRegistry
	schema    *ast.Document
	factory   *RocketDataSourceFactory
}

// RocketDataSourceFactory creates RocketDataSource instances
type RocketDataSourceFactory struct {
	resolvers        *registry.ResolverRegistry
	schema           *ast.Document
	requestedFields  []string           // Fields requested in the current query
	operationType    ast.OperationType  // Type of operation (Query, Mutation, Subscription)
	objectFetchConfig interface{}        // Store reference to objectFetchConfiguration to pass field coord
}

// NewRocketDataSourceFactory creates a new factory for Rocket DataSources
func NewRocketDataSourceFactory(resolvers *registry.ResolverRegistry, schema *ast.Document) *RocketDataSourceFactory {
	return &RocketDataSourceFactory{
		resolvers:     resolvers,
		schema:        schema,
		operationType: ast.OperationTypeQuery, // Default to Query
	}
}

// NewRocketDataSourceFactoryWithFields creates a factory with knowledge of requested fields and operation type
func NewRocketDataSourceFactoryWithFields(resolvers *registry.ResolverRegistry, schema *ast.Document, requestedFields []string, operationType ast.OperationType) *RocketDataSourceFactory {
	return &RocketDataSourceFactory{
		resolvers:       resolvers,
		schema:          schema,
		requestedFields: requestedFields,
		operationType:   operationType,
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
	return plan.DataSourcePlanningBehavior{
		// Allow Rocket to handle nested fields via auto-resolution
		// This tells the planner that Rocket can handle any field in ChildNodes
	}
}

// ToDataSourceConfiguration converts the factory to a DataSourceConfiguration
// This allows it to be used in plan.Configuration.DataSources
func (f *RocketDataSourceFactory) ToDataSourceConfiguration(id, name string) (plan.DataSourceConfiguration[interface{}], error) {
	// Create metadata and populate RootNodes with fields Rocket can resolve
	metadata := &plan.DataSourceMetadata{}
	
	// Populate RootNodes with Query and Mutation fields that Rocket can resolve
	// This tells the planner which fields this DataSource handles
	// Include introspection fields (__schema, __type) - we'll handle them in RocketSource
	
	// CRITICAL: Register each Query and Mutation field as a SEPARATE RootNode
	// This ensures graphql-go-tools creates separate fetch configurations for each field
	// and calls ConfigureFetch() once per field, allowing us to determine which field is being fetched
	
	// Add introspection fields
	metadata.RootNodes = append(metadata.RootNodes, plan.TypeField{
		TypeName:   "Query",
		FieldNames: []string{"__schema"},
	})
	metadata.RootNodes = append(metadata.RootNodes, plan.TypeField{
		TypeName:   "Query",
		FieldNames: []string{"__type"},
	})
	
	// Add each Query field as a separate RootNode
	// IMPORTANT: We must iterate in a consistent order to match ConfigureFetch() call order
	// Collect all query names, sort them, then register
	queryNames := make([]string, 0, len(f.resolvers.Query))
	for queryName := range f.resolvers.Query {
		// Skip introspection fields (already added above)
		if !strings.HasPrefix(queryName, "__") {
			queryNames = append(queryNames, queryName)
		}
	}
	// Sort to ensure deterministic order
	sort.Strings(queryNames)
	for _, queryName := range queryNames {
		metadata.RootNodes = append(metadata.RootNodes, plan.TypeField{
			TypeName:   "Query",
			FieldNames: []string{queryName},
		})
	}
	
	// Add each Mutation field as a separate RootNode
	// Also sort mutations for deterministic order
	mutationNames := make([]string, 0, len(f.resolvers.Mutation))
	for mutationName := range f.resolvers.Mutation {
		mutationNames = append(mutationNames, mutationName)
	}
	sort.Strings(mutationNames)
	for _, mutationName := range mutationNames {
		metadata.RootNodes = append(metadata.RootNodes, plan.TypeField{
			TypeName:   "Mutation",
			FieldNames: []string{mutationName},
		})
	}
	
	// For type resolvers, we need to add them as ChildNodes
	// ChildNodes are fields that can be resolved when we have the parent object
	// We'll track which types we've seen so we can merge fields later
	typeResolverFields := make(map[string]map[string]bool)
	for typeName, typeResolvers := range f.resolvers.Types {
		fieldMap := make(map[string]bool)
		var fieldNames []string
		for fieldName := range typeResolvers {
			fieldNames = append(fieldNames, fieldName)
			fieldMap[fieldName] = true
		}
		if len(fieldNames) > 0 {
			typeResolverFields[typeName] = fieldMap
			metadata.ChildNodes = append(metadata.ChildNodes, plan.TypeField{
				TypeName:   typeName,
				FieldNames: fieldNames,
			})
		}
	}
	
	// For auto-resolution: Add all fields of types returned by Query fields to ChildNodes
	// This allows Rocket to auto-resolve nested fields even without explicit resolvers
	// We'll scan the schema to find types that Query fields return
	// If a type already has a ChildNode (from type resolvers), merge in all remaining fields
	if f.schema != nil {
		for i := range f.schema.ObjectTypeDefinitions {
			typeName := f.schema.ObjectTypeDefinitionNameString(i)
			// Skip Query and Mutation (they're root nodes)
			if typeName == "Query" || typeName == "Mutation" {
				continue
			}
			
			objDef := f.schema.ObjectTypeDefinitions[i]
			fieldsRefs := objDef.FieldsDefinition.Refs
			if len(fieldsRefs) == 0 {
				continue
			}
			
			// Get all fields from schema
			var allFieldNames []string
			for _, fieldRef := range fieldsRefs {
				fieldName := f.schema.FieldDefinitionNameString(fieldRef)
				allFieldNames = append(allFieldNames, fieldName)
			}
			
			// Check if this type already has a ChildNode
			existingChildNodeIndex := -1
			for idx, childNode := range metadata.ChildNodes {
				if childNode.TypeName == typeName {
					existingChildNodeIndex = idx
					break
				}
			}
			
			if existingChildNodeIndex >= 0 {
				// Merge: Add all schema fields that aren't already in the ChildNode
				existingChildNode := metadata.ChildNodes[existingChildNodeIndex]
				existingFieldMap := make(map[string]bool)
				for _, fieldName := range existingChildNode.FieldNames {
					existingFieldMap[fieldName] = true
				}
				
				// Add any missing fields
				var newFields []string
				for _, fieldName := range allFieldNames {
					if !existingFieldMap[fieldName] {
						newFields = append(newFields, fieldName)
					}
				}
				
				if len(newFields) > 0 {
					// Update the existing ChildNode with all fields
					metadata.ChildNodes[existingChildNodeIndex].FieldNames = append(existingChildNode.FieldNames, newFields...)
				}
			} else {
				// Add new ChildNode with all fields
				if len(allFieldNames) > 0 {
					metadata.ChildNodes = append(metadata.ChildNodes, plan.TypeField{
						TypeName:   typeName,
						FieldNames: allFieldNames,
					})
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

