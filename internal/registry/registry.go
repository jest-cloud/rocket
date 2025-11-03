package registry

import (
	"github.com/jest-cloud/rocket/internal/types"
)

// ResolverRegistry holds all resolvers stitched together
// Combines resolvers from different modules in a clean, modular way
type ResolverRegistry struct {
	Query        map[string]types.FieldResolveFn
	Mutation     map[string]types.FieldResolveFn
	Subscription map[string]types.SubscriptionResolveFn
	Types        map[string]map[string]types.FieldResolveFn
}

// NewResolverRegistry creates a new registry by merging module resolvers
// This is like:
//   export const resolvers = {
//     Query: { ...userQueries, ...orgQueries },
//     Mutation: { ...userMutations, ...orgMutations }
//   }
func NewResolverRegistry(modules ...types.ModuleResolvers) *ResolverRegistry {
	registry := &ResolverRegistry{
		Query:        make(map[string]types.FieldResolveFn),
		Mutation:     make(map[string]types.FieldResolveFn),
		Subscription: make(map[string]types.SubscriptionResolveFn),
		Types:        make(map[string]map[string]types.FieldResolveFn),
	}

	// Merge all module resolvers
	for _, module := range modules {
		// Spread query resolvers
		for key, resolver := range module.QueryResolvers() {
			registry.Query[key] = resolver
		}

		// Spread mutation resolvers
		for key, resolver := range module.MutationResolvers() {
			registry.Mutation[key] = resolver
		}

		// Spread subscription resolvers
		for key, resolver := range module.SubscriptionResolvers() {
			registry.Subscription[key] = resolver
		}

		// Spread type resolvers
		for typeName, typeResolvers := range module.TypeResolvers() {
			registry.Types[typeName] = typeResolvers
		}
	}

	return registry
}

// GetQueryResolver returns a query resolver by name
func (r *ResolverRegistry) GetQueryResolver(name string) (types.FieldResolveFn, bool) {
	resolver, ok := r.Query[name]
	return resolver, ok
}

// GetMutationResolver returns a mutation resolver by name
func (r *ResolverRegistry) GetMutationResolver(name string) (types.FieldResolveFn, bool) {
	resolver, ok := r.Mutation[name]
	return resolver, ok
}

// GetSubscriptionResolver returns a subscription resolver by name
func (r *ResolverRegistry) GetSubscriptionResolver(name string) (types.SubscriptionResolveFn, bool) {
	resolver, ok := r.Subscription[name]
	return resolver, ok
}

// GetTypeResolver returns a type's field resolver
func (r *ResolverRegistry) GetTypeResolver(typeName, fieldName string) (types.FieldResolveFn, bool) {
	if typeResolvers, ok := r.Types[typeName]; ok {
		if resolver, ok := typeResolvers[fieldName]; ok {
			return resolver, true
		}
	}
	return nil, false
}

