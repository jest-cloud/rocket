# Architecture Decision: Execution Engine

## Current Situation

We're using `graphql-go-tools` to build Rocket as a **federation subgraph**. This is the CORRECT approach:

1. **Rocket is a federation subgraph** - A standalone GraphQL server that can run independently OR be composed into a federated supergraph via a router
2. **DataSource pattern is correct** - This is how federation subgraphs work in graphql-go-tools
3. **Can run standalone OR behind a router** - Rocket can be:
   - Run standalone as a regular GraphQL server
   - Composed with other Rocket instances via a federation router (like Cosmo Router) to create a federated supergraph

## Implementation Details

We're building Rocket as a federation subgraph, which requires:

1. **Custom DataSource bridge** (`datasource.go`, `datasource_planner.go`, `datasource_source.go`)
   - This is the standard way to build a subgraph
   - Our DataSource implements `resolve.DataSource` interface
   
2. **Execution wrapper** (`execution.go`)
   - Wraps graphql-go-tools resolver execution
   - Handles plan execution and result formatting

3. **Field coordinate extraction** (in `execution.go`)
   - Extracts field info from planner to pass to our DataSource
   - This is necessary because we need to know which field is being resolved

The approach is **correct** - we're building a proper federation subgraph. Any "brittleness" comes from implementation details that can be improved, not from fighting the library.

## Decision: Continue with Current Approach ✅

**We ARE using graphql-go-tools correctly** - Rocket is a federation subgraph.

### Why This Works

1. **Federation Architecture**:
   - Individual subgraphs (like Rocket) are standalone GraphQL servers
   - A router/gateway (like Cosmo Router) composes multiple subgraphs into a federated supergraph
   - Rocket can run standalone OR be composed with others

2. **DataSource Pattern**:
   - This is the standard way to build a subgraph in graphql-go-tools
   - Each subgraph implements a DataSource that handles its fields
   - The router uses these DataSources to compose the federated graph

3. **Implementation Status**:
   - ✅ Schema parsing works
   - ✅ Query execution works
   - ✅ Nested field resolution works
   - ⚠️ Mutations need fixing (implementation detail, not architectural issue)
   - ⚠️ Some code cleanup needed (field coordinate extraction could be cleaner)

### Next Steps: Refinement, Not Replacement

Instead of changing the architecture, we should:

1. **Clean up implementation**:
   - Improve field coordinate extraction
   - Fix mutation execution
   - Simplify where possible

2. **Document the architecture**:
   - Explain that Rocket is a federation subgraph
   - Show how it can run standalone or be composed
   - Provide examples of both use cases

3. **Add federation features** (when needed):
   - Entity resolution (`_entities` query)
   - Federation directives (`@key`, `@requires`, `@provides`, etc.)
   - Service composition

### Benefits

- ✅ Built-in federation support (ready for composition)
- ✅ Production-grade execution engine
- ✅ Can run standalone OR as part of federated graph
- ✅ Using graphql-go-tools as intended

