# Migration Decision: graphql-go vs graphql-go-tools

## Current Status

Rocket currently uses `github.com/graphql-go/graphql` for GraphQL execution.

## Issue Encountered

Auto-resolution doesn't work for list/slice elements (see issue #1).

## Investigation: graphql-go-tools

### ✅ What Works
- Schema parsing: Can parse `.graphql` files using `astparser`
- Library is actively maintained and high-performance

### ❌ Challenges
1. **Designed for Federation/Routing**: Primary use case is GraphQL gateway/router
2. **Different Execution Model**: 
   - Uses execution plans (`GraphQLResponse`) instead of direct execution
   - Requires DataSource pattern instead of resolver functions
   - Would need significant wrapper code
3. **Complexity**: Much more complex than needed for standalone execution

### Migration Effort Estimate
- **High** (~2-3 weeks)
- Would need to:
  - Build wrapper for execution engine
  - Create custom DataSource to bridge our resolver pattern
  - Implement execution plan builder
  - Test thoroughly to ensure compatibility

## Recommendation

**Stay with graphql-go and fix the auto-resolution bug**

### Reasons:
1. **Faster**: Fixing the bug is much faster than full migration
2. **Simpler**: graphql-go's execution model matches our needs
3. **Compatible**: Our current resolver pattern works well with graphql-go
4. **Working**: Everything else works - just need to fix one bug

### Next Steps
1. Continue investigating the auto-resolution bug in graphql-go
2. Fix the issue by ensuring resolvers are called correctly for list elements
3. If graphql-go proves problematic long-term, revisit migration

