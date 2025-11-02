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
	resolvers *ResolverRegistry
	schema    *ast.Document
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
	return plan.NewDataSourceConfigurationWithName[interface{}](
		id,
		name,
		f,           // factory implements PlannerFactory
		nil,         // metadata - optional
		interface{}(nil), // custom config - we don't need it
	)
}

