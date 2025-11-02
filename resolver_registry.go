package rocket

// ResolverRegistry holds all resolvers stitched together
// Similar to how Apollo combines resolvers from different modules
type ResolverRegistry struct {
	Query    map[string]FieldResolveFn
	Mutation map[string]FieldResolveFn
	Types    map[string]map[string]FieldResolveFn
}

// NewResolverRegistry creates a new registry by merging module resolvers
// This is like:
//   export const resolvers = {
//     Query: { ...userQueries, ...orgQueries },
//     Mutation: { ...userMutations, ...orgMutations }
//   }
func NewResolverRegistry(modules ...ModuleResolvers) *ResolverRegistry {
	registry := &ResolverRegistry{
		Query:    make(map[string]FieldResolveFn),
		Mutation: make(map[string]FieldResolveFn),
		Types:    make(map[string]map[string]FieldResolveFn),
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

		// Spread type resolvers
		for typeName, typeResolvers := range module.TypeResolvers() {
			registry.Types[typeName] = typeResolvers
		}
	}

	return registry
}

// GetQueryResolver returns a query resolver by name
func (r *ResolverRegistry) GetQueryResolver(name string) (FieldResolveFn, bool) {
	resolver, ok := r.Query[name]
	return resolver, ok
}

// GetMutationResolver returns a mutation resolver by name
func (r *ResolverRegistry) GetMutationResolver(name string) (FieldResolveFn, bool) {
	resolver, ok := r.Mutation[name]
	return resolver, ok
}

// GetTypeResolver returns a type's field resolver
func (r *ResolverRegistry) GetTypeResolver(typeName, fieldName string) (FieldResolveFn, bool) {
	if typeResolvers, ok := r.Types[typeName]; ok {
		if resolver, ok := typeResolvers[fieldName]; ok {
			return resolver, true
		}
	}
	return nil, false
}

