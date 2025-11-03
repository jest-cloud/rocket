# Migration Plan: graphql-go → graphql-go-tools

## ✅ Proof of Concept Results

1. **Schema Parsing**: ✅ Works perfectly
   - Can parse `.graphql` files using `astparser`
   - Can merge base schema with `asttransform.MergeDefinitionWithBaseSchema`

2. **Execution Planning**: ✅ Planner works
   - Can create planner with empty DataSources
   - Planner correctly identifies missing DataSources
   - Error: "could not select the datasource to resolve Query.hello" (expected)

3. **Key Insight**: graphql-go-tools uses a DataSource pattern instead of direct resolvers

## Migration Strategy

### Phase 1: Create Custom DataSource Bridge ✅ (In Progress)

We need to create a custom DataSource that bridges Rocket's resolver pattern to graphql-go-tools:

1. **Create `ResolveDataSource`**:
   - Implements `plan.DataSource` interface
   - Wraps Rocket's `ModuleResolvers` interface
   - Maps GraphQL field paths to Rocket resolver functions

2. **Create `ResolvePlanner`**:
   - Implements `plan.DataSourcePlanner` interface  
   - Configures fetch operations to call Rocket resolvers
   - Returns `resolve.FetchConfiguration` with our custom Source

3. **Create `ResolveSource`**:
   - Implements `resolve.DataSource` interface
   - Calls Rocket resolver functions
   - Converts results to graphql-go-tools format

### Phase 2: Update Rocket Core Files

1. **Replace `schema.go`**:
   - Use `astparser` instead of `gqlparser`
   - Use `asttransform.MergeDefinitionWithBaseSchema` instead of gqlparser.LoadSchema

2. **Replace `schema_builder.go`**:
   - Build execution plan instead of `graphql.Schema`
   - Use `plan.NewPlanner` with custom DataSource
   - Return execution plan instead of schema

3. **Replace `execution.go`**:
   - Use `Resolver.ResolveGraphQLResponse` instead of `graphql.Do`
   - Adapt result format to Rocket's Result type

4. **Update `default_resolver.go`**:
   - Work within DataSource pattern
   - Auto-resolution handled by our custom Source

### Phase 3: Update Dependencies

1. **Update `go.mod`**:
   - Remove `github.com/graphql-go/graphql`
   - Add `github.com/wundergraph/graphql-go-tools/v2`
   - Keep `github.com/vektah/gqlparser/v2` (if needed for compatibility)

## Files to Create

1. `datasource.go` - Custom DataSource bridge
2. `datasource_planner.go` - Planner for our DataSource
3. `datasource_source.go` - Source that executes Rocket resolvers

## Files to Modify

1. `schema.go` - Parse with astparser
2. `schema_builder.go` - Build execution plan
3. `execution.go` - Execute with graphql-go-tools resolver
4. `go.mod` - Update dependencies

## Next Steps

1. ✅ Understand DataSource interface fully
2. ⏳ Create custom DataSource bridge
3. ⏳ Test with simple query
4. ⏳ Migrate all Rocket code
5. ⏳ Test auto-resolution works
6. ⏳ Remove graphql-go dependency

## 🎁 Bonus: Federation Support

**graphql-go-tools is designed for federation!** This is a major advantage:

✅ **GraphQL Federation v1 & v2**: Rocket will natively support federation
✅ **Subgraph Support**: Rocket schemas can be exposed as federation subgraphs  
✅ **Federated Composition**: Multiple Rocket instances can be composed into a federated graph
✅ **Entity Resolution**: Built-in entity resolution across services
✅ **Service Composition**: Automatic service composition with graphql-go-tools

This is a **huge advantage** over graphql-go, which doesn't have federation support!
The migration gives us federation capabilities "for free" - no additional work needed!

### Future Federation Features
- Expose Rocket schemas as federated subgraphs
- Compose multiple Rocket services into a federated supergraph
- Entity resolution across multiple Rocket instances
- Cross-service field resolution

