# Queries

## Overview

Queries are read operations in GraphQL. Rocket provides a simple, type-safe way to define query resolvers using the `QueryResolvers()` method.

## Status: FULLY IMPLEMENTED ✅

Queries have been fully supported since v0.1.0 with:
- ✅ Query execution via HTTP POST/GET
- ✅ Auto-field resolution for struct fields
- ✅ Nested queries and relationships
- ✅ Arguments and variables
- ✅ Context propagation
- ✅ Field order preservation

## Basic Query

### Schema

```graphql
type Query {
  hello: String!
  user(id: ID!): User
  users(limit: Int): [User!]!
}

type User {
  id: ID!
  name: String!
  email: String!
}
```

### Resolver

```go
package main

import "github.com/jest-cloud/rocket"

type Resolvers struct {
    service *UserService
}

func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "hello": func(p rocket.ResolveParams) (interface{}, error) {
            return "Hello, World!", nil
        },
        "user": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            return r.service.GetUser(p.Context, id)
        },
        "users": func(p rocket.ResolveParams) (interface{}, error) {
            limit := 10
            if l, ok := p.Args["limit"].(float64); ok {
                limit = int(l)
            }
            return r.service.GetUsers(p.Context, limit)
        },
    }
}
```

## Query Execution

### HTTP Request

```bash
# POST request
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ hello }"}'

# With variables
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query GetUser($id: ID!) { user(id: $id) { id name email } }",
    "variables": { "id": "123" }
  }'
```

### GraphQL Playground

```graphql
# Simple query
query {
  hello
}

# Query with arguments
query GetUser($id: ID!) {
  user(id: $id) {
    id
    name
    email
  }
}

# Multiple fields
query {
  hello
  users(limit: 5) {
    id
    name
    email
  }
}
```

## Working with Arguments

### Required Arguments

```go
"user": func(p rocket.ResolveParams) (interface{}, error) {
    id, ok := p.Args["id"].(string)
    if !ok {
        return nil, fmt.Errorf("id is required")
    }
    return r.service.GetUser(p.Context, id)
}
```

### Optional Arguments with Defaults

```go
"users": func(p rocket.ResolveParams) (interface{}, error) {
    limit := 10 // default
    if l, ok := p.Args["limit"].(float64); ok {
        limit = int(l)
    }
    
    offset := 0 // default
    if o, ok := p.Args["offset"].(float64); ok {
        offset = int(o)
    }
    
    return r.service.GetUsers(p.Context, limit, offset)
}
```

## Context Usage

Access context for auth, tracing, etc:

```go
"currentUser": func(p rocket.ResolveParams) (interface{}, error) {
    // Get user from context (set by auth middleware)
    userID, ok := p.Context.Value("userID").(string)
    if !ok {
        return nil, fmt.Errorf("unauthorized")
    }
    
    return r.service.GetUser(p.Context, userID)
}
```

## Nested Queries

### Auto-Resolution

Return structs and Rocket auto-resolves fields:

```go
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

"user": func(p rocket.ResolveParams) (interface{}, error) {
    id := p.Args["id"].(string)
    
    // Return struct - fields auto-resolve!
    return &User{
        ID:    id,
        Name:  "Alice",
        Email: "alice@example.com",
    }, nil
}
```

### Custom Field Resolvers

Override specific fields when needed:

```go
func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            "posts": func(p rocket.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return r.service.GetPostsByUser(p.Context, user.ID)
            },
        },
    }
}
```

Schema:
```graphql
type User {
  id: ID!
  name: String!
  email: String!
  posts: [Post!]!  # Custom resolver
}

type Post {
  id: ID!
  title: String!
  content: String!
}
```

## Error Handling

### Returning Errors

```go
"user": func(p rocket.ResolveParams) (interface{}, error) {
    id := p.Args["id"].(string)
    
    user, err := r.service.GetUser(p.Context, id)
    if err != nil {
        // Return error - will be included in GraphQL errors array
        return nil, fmt.Errorf("user not found: %w", err)
    }
    
    return user, nil
}
```

### Partial Errors

```go
"users": func(p rocket.ResolveParams) (interface{}, error) {
    users, err := r.service.GetUsers(p.Context)
    if err != nil {
        // Return partial data with error
        return users, fmt.Errorf("some users failed to load: %w", err)
    }
    
    return users, nil
}
```

## Batching & DataLoader

For N+1 query prevention, implement DataLoader pattern:

```go
type Resolvers struct {
    service *UserService
    loader  *dataloader.Loader
}

func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "Post": {
            "author": func(p rocket.ResolveParams) (interface{}, error) {
                post := p.Source.(*Post)
                
                // Use DataLoader to batch user fetches
                thunk := r.loader.Load(p.Context, post.AuthorID)
                return thunk()
            },
        },
    }
}
```

## Performance Tips

### 1. Use Pointers for Large Structs

```go
// Good - uses pointer
return &User{...}, nil

// Less efficient for large structs
return User{...}, nil
```

### 2. Lazy Load Relations

```go
"user": func(p rocket.ResolveParams) (interface{}, error) {
    id := p.Args["id"].(string)
    
    // Only fetch user, not relations
    user, err := r.service.GetUser(p.Context, id)
    // Relations are fetched only if requested in query
    return user, err
}
```

### 3. Use Selection Sets

Check what fields are requested:

```go
"user": func(p rocket.ResolveParams) (interface{}, error) {
    id := p.Args["id"].(string)
    
    // Check if expensive fields are requested
    selectionSet := p.Info.SelectionSet
    // Optimize query based on selection
    
    return r.service.GetUser(p.Context, id)
}
```

## Testing

### Unit Test

```go
func TestUserQuery(t *testing.T) {
    resolver := &Resolvers{service: mockService}
    
    params := rocket.ResolveParams{
        Args: map[string]interface{}{"id": "123"},
        Context: context.Background(),
    }
    
    result, err := resolver.QueryResolvers()["user"](params)
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Integration Test

```go
func TestQueryExecution(t *testing.T) {
    schema, _ := rocket.BuildSchema(
        rocket.Config{SchemaPath: "schema.graphql"},
        &Resolvers{service: testService},
    )
    
    query := `{ user(id: "123") { id name } }`
    result := schema.Execute(context.Background(), query, nil, "")
    
    assert.Empty(t, result.Errors)
    assert.NotNil(t, result.Data)
}
```

## Examples

See complete examples in:
- [Migration Test](../../test/migration_test.go) - Basic queries
- [Viewer Test](../../test/viewer_test.go) - Nested queries
- [FirePrint API](../../../fire-print/apps/api/) - Production example

## Best Practices

1. **Keep resolvers thin** - Move business logic to service layer
2. **Use context** - Pass user auth, tracing, etc via context
3. **Return errors** - Let GraphQL handle error formatting
4. **Type assertion** - Safely cast arguments and source
5. **Leverage auto-resolution** - Return structs when possible
6. **Document complex queries** - Add comments for maintainability

## Common Patterns

### Pagination

```graphql
type Query {
  users(limit: Int, offset: Int): UserConnection!
}

type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
}

type UserEdge {
  node: User!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
}
```

### Filtering

```graphql
type Query {
  users(filter: UserFilter): [User!]!
}

input UserFilter {
  name: String
  email: String
  role: String
}
```

### Sorting

```graphql
type Query {
  users(sortBy: UserSort, sortOrder: SortOrder): [User!]!
}

enum UserSort {
  NAME
  EMAIL
  CREATED_AT
}

enum SortOrder {
  ASC
  DESC
}
```

## Status Summary

| Feature | Status | Since |
|---------|--------|-------|
| Basic Queries | ✅ Complete | v0.1.0 |
| Arguments | ✅ Complete | v0.1.0 |
| Variables | ✅ Complete | v0.1.0 |
| Nested Queries | ✅ Complete | v0.1.0 |
| Context Propagation | ✅ Complete | v0.1.0 |
| Auto-field Resolution | ✅ Complete | v0.1.0 |
| Field Order Preservation | ✅ Complete | v0.1.0 |
| Error Handling | ✅ Complete | v0.1.0 |

**Queries: 100% Complete** ✅

