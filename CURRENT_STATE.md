# Rocket - Current State Summary

## What's Working ✅

### Successfully Implemented
1. **Module Organization**: Code organized into internal packages
   - `internal/execution` - GraphQL execution logic
   - `internal/datasource` - graphql-go-tools DataSource bridge
   - `internal/resolver` - Default field resolver
   - `internal/registry` - Resolver registry
   - `internal/types` - Common types  
   - `internal/http` - HTTP handlers
   - `internal/compiler` - Schema compilation (planned)

2. **Documentation**: Moved to `dev/docs/` and referenced in `README.md`

3. **Tests Passing** (7/11):
   - ✅ TestFirePrintLikeSetup
   - ✅ TestSimpleQuery/hello_query  
   - ✅ TestViewerQuery/viewer_with_id
   - ✅ TestViewerQuery/viewer_with_all_fields
   - ✅ TestViewerQuery/viewer_with_organizations

### Architecture
- Successfully bridged Rocket's resolver pattern to graphql-go-tools' DataSource pattern
- Registered each Query/Mutation field as separate RootNodes
- Implemented field extraction from GraphQL operation when available
- Created fallback mechanism when field can't be determined

## What's Not Working ❌

### Failing Tests (4/11):
- ❌ TestSimpleQuery/users_query
- ❌ TestSimpleQuery/user_query  
- ❌ TestMutation/createUser_mutation

### Root Cause
**Field Detection Issue**: When `visitor.Operation` is `nil` during planning (first query), we fall back to trying all resolvers alphabetically. This returns the wrong field (e.g., `hello` when `users` was requested), causing graphql-go-tools to return null.

## Technical Details

### How Field Detection Works
1. **During Register()**: Extract requested field from `visitor.Operation` if available
2. **During ConfigureFetch()**: Create RocketSource with `targetField` set to extracted field
3. **During Load()**: Use `targetField` if set, otherwise fall back to trying all resolvers

### The Problem
- `visitor.Operation` is `nil` on first planning (when planner is created)
- `visitor.Operation` is populated on subsequent queries (planner reuse)
- Fallback tries resolvers alphabetically, returns first success
- graphql-go-tools expects specific field but receives different one
- Result: null for requested field

### Why Some Tests Pass
- **hello_query**: First alphabetically, so fallback works by luck
- **viewer tests**: More complex queries where operation IS populated during planning
- **users/user tests**: Fail because fallback returns wrong field

## Next Steps to Fix

### Option 1: Fix Planner Lifecycle (Recommended)
Create a new planner instance for each query execution instead of reusing:
```go
// In schema.go Execute()
planner, err := buildExecutionPlan(s.schemaDoc, s.resolvers)
// Use fresh planner for this query
```

### Option 2: Improve Fallback Logic
Instead of trying resolvers blindly, parse the actual query string to extract field names when operation is nil.

### Option 3: Use graphql-go-tools Templates Properly
Research how to use Input templates with variables like `{{.field}}` or `{{.path}}` that might contain field information.

## Files Modified

### Core Files
- `rocket/schema.go` - Schema building and execution
- `rocket/rocket.go` - Public API re-exports
- `rocket/http.go` - HTTP handler re-exports

### Internal Modules
- `internal/datasource/datasource.go` - DataSource factory
- `internal/datasource/planner.go` - Planning logic
- `internal/datasource/source.go` - Execution logic
- `internal/execution/execution.go` - Plan execution
- `internal/registry/registry.go` - Resolver registry
- `internal/types/types.go` - Common types
- `internal/types/result.go` - Result types
- `internal/http/handler.go` - HTTP handler
- `internal/http/playground.go` - Playground handlers
- `internal/resolver/default_resolver.go` - Auto-resolution

### Tests  
- `test/migration_test.go` - Basic query tests
- `test/mutation_test.go` - Mutation tests
- `test/viewer_test.go` - Complex query tests
- `test/fireprint_test.go` - Integration tests

## Current Test Results
```
7 passing, 4 failing
Success rate: 64%
```

The foundation is solid. The remaining issue is a known, isolated problem with field detection that has clear solution paths.

