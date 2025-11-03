# Rocket GraphQL - Current Status & Next Steps

## Current State: 10/11 Tests Passing (91%)

### ✅ What Works
- **All Query tests** - Simple queries, complex queries with arguments
- **All Viewer tests** - Nested object resolution with auto-resolution
- **Field coordinate extraction** - Successfully extracting requested fields from queries
- **Operation type detection** - Correctly identifying Query vs Mutation operations
- **Auto-resolution** - Struct fields automatically resolved via reflection

### ❌ What Doesn't Work
- **Mutations with variables** - Arguments not being passed to resolvers
  - Test: `TestMutation/createUser_mutation`
  - Root cause: Input template `{"arguments": {"input": {{.arguments.input}}}}` evaluates to empty
  - graphql-go-tools' template system doesn't provide the context we expected

## Root Cause Analysis

We created a **custom DataSource** to bridge Rocket's resolver pattern to graphql-go-tools. This works great for queries but struggles with mutations because:

1. **Input Template Complexity**: graphql-go-tools uses a template system for DataSource inputs, but the template variables we need (`{{.arguments.input}}`) aren't available or aren't evaluating correctly
2. **Fighting the Framework**: Custom DataSources are meant for external data sources (REST APIs, databases), not for implementing resolver logic
3. **Missing Documentation**: Limited examples of custom DataSources with argument extraction

## The Better Approach: Use Built-in Resolver Pattern

WunderGraph's graphql-go-tools is designed to work with resolvers, but we've been implementing them at the wrong layer.

### Current (Wrong) Approach
```
Query → Planner → CustomDataSource.Load() → Our Resolvers
```
- We intercept at the DataSource layer
- Have to manually extract arguments, field coordinates, etc.
- Fighting against the framework's design

### Correct Approach
```
Query → Planner → Resolver.ResolveGraphQLResponse() → Built-in field resolution
```
- Register resolvers at the execution layer, not datasource layer
- graphql-go-tools handles argument extraction automatically
- Work WITH the framework, not against it

## Refactor Plan

### Step 1: Remove Custom DataSource Layer
**Files to delete/refactor:**
- `internal/datasource/planner.go` - Custom planner (delete)
- `internal/datasource/source.go` - Custom DataSource.Load() (delete)
- `internal/datasource/datasource.go` - DataSource factory (simplify or delete)

### Step 2: Implement Proper Resolver Registration

**Option A: Local Resolver (Simpler)**
```go
// In execution.go
func ExecuteQuery(ctx context.Context, schema *ast.Document, query string, resolvers *registry.ResolverRegistry) *types.Result {
    // Parse and normalize query
    queryDoc, _ := astparser.ParseGraphqlDocumentString(query)
    
    // Create execution plan WITHOUT custom DataSource
    planner := plan.NewPlanner(plan.Configuration{
        // No custom datasource - use built-in local resolution
    })
    
    execPlan := planner.Plan(&queryDoc, schema, "", &report)
    
    // Create resolver with our field resolvers registered
    resolver := resolve.New(ctx, resolve.ResolverOptions{
        // Register Rocket resolvers here
    })
    
    // Execute - arguments automatically extracted!
    resolver.ResolveGraphQLResponse(ctx, execPlan.Response, nil, &output)
}
```

**Option B: Use graphql-go-tools' Schema Execution**
graphql-go-tools might have a higher-level API for schema execution that we're missing. Need to research `graphql` package in graphql-go-tools.

### Step 3: Map Rocket Resolvers to graphql-go-tools Format

The key challenge: graphql-go-tools expects resolvers in a specific format. We need to:
1. Understand what format it expects
2. Create an adapter from our `FieldResolveFn` to their format
3. Register all Query, Mutation, and Type resolvers

### Step 4: Test Everything
Run all 11 tests and verify:
- ✅ Queries work
- ✅ Mutations with arguments work  
- ✅ Nested object resolution works
- ✅ Auto-resolution works

## Research Needed

1. **What is the correct way to register field resolvers in graphql-go-tools?**
   - Is there a `TypeFieldResolver` interface?
   - Does the `Resolver` accept a callback/handler?
   - Is there a schema execution API we're missing?

2. **How do other projects use graphql-go-tools for standalone execution?**
   - Most examples are for federation/gateway
   - Need to find examples of standalone schema execution

3. **Can we use the `graphql` package instead of lower-level `engine` APIs?**
   - graphql-go-tools has a `pkg/graphql` package
   - Might provide higher-level execution APIs

## Files to Review

- `github.com/wundergraph/graphql-go-tools/v2/pkg/graphql` - High-level API?
- `github.com/wundergraph/graphql-go-tools/v2/pkg/engine/resolve` - Resolver interface
- `github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan` - Planning without DataSources

## Current Code Structure

```
rocket/
├── schema.go                          - Entry point, needs refactor
├── internal/
│   ├── execution/execution.go         - Already using Resolver.ResolveGraphQLResponse
│   ├── datasource/                    - TO BE DELETED/REFACTORED
│   │   ├── datasource.go
│   │   ├── planner.go
│   │   └── source.go
│   ├── registry/registry.go           - Keeper - holds our resolvers
│   ├── resolver/default_resolver.go   - Keeper - auto-resolution
│   └── types/                         - Keeper - type definitions
└── test/                              - 10/11 passing
```

## Success Criteria

- [ ] All 11 tests pass
- [ ] Code is simpler (fewer lines than current custom DataSource approach)
- [ ] No "fighting the framework" - using graphql-go-tools as intended
- [ ] Arguments automatically extracted for mutations
- [ ] Performance is good (no unnecessary overhead)

## Estimated Effort

- **Research**: 30-60 minutes (understand correct graphql-go-tools patterns)
- **Implementation**: 2-3 hours (refactor execution layer)
- **Testing & Debugging**: 1-2 hours (ensure all tests pass)
- **Total**: 4-6 hours

This is a clean refactor with a clear goal: use graphql-go-tools the way it was designed to be used.

