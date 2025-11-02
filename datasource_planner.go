package rocket

import (
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/resolve"
)

// RocketPlanner implements plan.DataSourcePlanner for Rocket resolvers
// It configures how fields are resolved using Rocket's resolver pattern
type RocketPlanner struct {
	id     int
	factory *RocketDataSourceFactory
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
	// Register our resolver registry with the visitor
	// This tells the planner how to resolve fields
	return nil
}

func (p *RocketPlanner) ConfigureFetch() resolve.FetchConfiguration {
	// Configure fetch to use our custom Source
	return resolve.FetchConfiguration{
		DataSource: &RocketSource{
			resolvers: p.factory.resolvers,
		},
	}
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

