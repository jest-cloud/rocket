# Rocket v0.6.0 Release Notes

## 🎉 Major Improvements

### Simplified Query Execution Model

This release dramatically simplifies how Rocket handles GraphQL queries by moving to a **direct execution model** for all queries.

#### What Changed

Previously, Rocket used a complex DataSource/Input template system from `graphql-go-tools` which had limitations:
- Nested fields with arguments didn't work properly
- Complex AST traversal was needed to detect which queries needed special handling
- The Input template evaluation was unreliable for custom DataSources

**Now**, all queries use direct resolver execution:
- ✅ **Simpler architecture** - No complex detection logic
- ✅ **Nested fields with arguments work naturally** - `user { organization(id: $orgId) }` just works
- ✅ **More predictable behavior** - Resolvers are always called directly with arguments
- ✅ **Better error handling** - Stack traces are clearer

#### Example: Nested Fields with Arguments

```graphql
query GetUserOrg($userId: ID!, $orgId: ID!) {
  user(id: $userId) {
    id
    name
    # This now works perfectly! 🎉
    organization(id: $orgId) {
      id
      name
    }
  }
}
```

Your resolver:
```go
func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            "organization": func(p rocket.ResolveParams) (interface{}, error) {
                // Arguments are properly passed!
                orgID := p.Args["id"].(string)
                return r.orgService.GetOrganizationByID(orgID)
            },
        },
    }
}
```

### What This Means for You

- **No breaking changes to your code** - Your existing resolvers work exactly the same
- **Better performance** - Less overhead from AST traversal
- **More reliable** - Fewer edge cases and gotchas
- **Simpler to reason about** - Queries always go through the same execution path

## 🐛 Bug Fixes

- Fixed infinite recursion when checking for fields with arguments in query AST
- Removed complex field detection logic that was prone to bugs
- Cleaned up debug logging throughout codebase

## 🧹 Internal Improvements

- Simplified `schema.go` execution logic
- Removed unused AST traversal functions
- Better code organization in datasource layer
- Improved error messages

## 📝 Documentation

- Updated README with information about nested fields with arguments
- Added release notes

## Migration Guide

No migration needed! This is a **drop-in replacement** - your existing code will continue to work exactly as before, but with better support for nested fields with arguments.

---

**Full Changelog**: https://github.com/jest-cloud/rocket/compare/v0.5.0...v0.6.0

