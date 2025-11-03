# Resolvers

## Overview

Resolvers are functions that tell GraphQL how to fetch data for each field in your schema. Rocket uses a **TypeScript-like map-based resolver pattern** that's clean, modular, and type-safe.

## The ModuleResolvers Interface

Every Rocket module must implement the `ModuleResolvers` interface:

```go
type ModuleResolvers interface {
    QueryResolvers() map[string]FieldResolveFn
    MutationResolvers() map[string]FieldResolveFn
    SubscriptionResolvers() map[string]SubscriptionResolveFn
    TypeResolvers() map[string]map[string]FieldResolveFn
}
```

This interface ensures your module provides resolvers for all operation types.

## Resolver Types

### 1. FieldResolveFn (Queries & Mutations)

Used for queries and mutations:

```go
type FieldResolveFn func(p ResolveParams) (interface{}, error)

type ResolveParams struct {
    Source  interface{}              // Parent object (for nested resolvers)
    Args    map[string]interface{}   // Field arguments
    Context context.Context          // Request context
    Info    ResolveInfo              // Field metadata
}
```

### 2. SubscriptionResolveFn (Subscriptions)

Used for subscriptions - returns a channel:

```go
type SubscriptionResolveFn func(p ResolveParams) (<-chan interface{}, error)
```

### 3. ResolveInfo

Metadata about the field being resolved:

```go
type ResolveInfo struct {
    FieldName    string      // Name of the field
    ParentType   string      // Parent type name
    ReturnType   string      // Return type name
    SelectionSet interface{} // Requested subfields
}
```

### Accessing Context in Resolvers

The `Context` field in `ResolveParams` contains request-scoped data like authenticated users:

```go
"currentUser": func(p rocket.ResolveParams) (interface{}, error) {
    // Access context value
    userID := p.Context.Value("userID").(string)
    
    // Use it
    return r.userService.GetUser(p.Context, userID)
}
```

See the **[Context Guide](./CONTEXT.md)** for complete authentication patterns.

## Basic Resolver Structure

### Minimal Example

```go
package main

import "github.com/jest-cloud/rocket"

type Resolvers struct {
    // Dependencies (services, database, etc.)
    userService *UserService
}

func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "hello": func(p rocket.ResolveParams) (interface{}, error) {
            return "Hello, Rocket!", nil
        },
    }
}

func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{}
}

func (r *Resolvers) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
    return map[string]rocket.SubscriptionResolveFn{}
}

func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{}
}
```

## Query Resolvers

Query resolvers fetch data without side effects:

```go
func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        // Simple query - no arguments
        "hello": func(p rocket.ResolveParams) (interface{}, error) {
            return "Hello, World!", nil
        },
        
        // Query with argument
        "user": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            return r.userService.GetUser(p.Context, id)
        },
        
        // Query with optional arguments
        "users": func(p rocket.ResolveParams) (interface{}, error) {
            limit := 10 // default
            if l, ok := p.Args["limit"].(float64); ok {
                limit = int(l)
            }
            return r.userService.GetUsers(p.Context, limit)
        },
        
        // Query using context
        "currentUser": func(p rocket.ResolveParams) (interface{}, error) {
            userID := p.Context.Value("userID").(string)
            return r.userService.GetUser(p.Context, userID)
        },
    }
}
```

### Schema
```graphql
type Query {
  hello: String!
  user(id: ID!): User
  users(limit: Int): [User!]!
  currentUser: User
}
```

## Mutation Resolvers

Mutation resolvers modify data:

```go
func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        // Create
        "createUser": func(p rocket.ResolveParams) (interface{}, error) {
            input := p.Args["input"].(map[string]interface{})
            
            // Extract fields
            name := input["name"].(string)
            email := input["email"].(string)
            password := input["password"].(string)
            
            // Validate
            if !isValidEmail(email) {
                return nil, fmt.Errorf("invalid email format")
            }
            
            // Create user
            return r.userService.CreateUser(p.Context, name, email, password)
        },
        
        // Update
        "updateUser": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            input := p.Args["input"].(map[string]interface{})
            
            // Check permissions
            userID := p.Context.Value("userID").(string)
            if id != userID {
                return nil, fmt.Errorf("unauthorized")
            }
            
            // Update
            return r.userService.UpdateUser(p.Context, id, input)
        },
        
        // Delete
        "deleteUser": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            
            // Check permissions
            userID := p.Context.Value("userID").(string)
            if id != userID {
                return nil, fmt.Errorf("unauthorized")
            }
            
            // Delete
            err := r.userService.DeleteUser(p.Context, id)
            return err == nil, err
        },
        
        // Transaction example
        "createOrder": func(p rocket.ResolveParams) (interface{}, error) {
            input := p.Args["input"].(map[string]interface{})
            
            // Start transaction
            tx, err := r.db.BeginTx(p.Context, nil)
            if err != nil {
                return nil, err
            }
            defer tx.Rollback()
            
            // Create order
            order, err := r.orderService.CreateOrder(p.Context, tx, input)
            if err != nil {
                return nil, err
            }
            
            // Update inventory
            err = r.inventoryService.Deduct(p.Context, tx, order.Items)
            if err != nil {
                return nil, err
            }
            
            // Commit
            if err := tx.Commit(); err != nil {
                return nil, err
            }
            
            return order, nil
        },
    }
}
```

### Schema
```graphql
type Mutation {
  createUser(input: CreateUserInput!): User!
  updateUser(id: ID!, input: UpdateUserInput!): User!
  deleteUser(id: ID!): Boolean!
  createOrder(input: CreateOrderInput!): Order!
}

input CreateUserInput {
  name: String!
  email: String!
  password: String!
}

input UpdateUserInput {
  name: String
  email: String
}
```

### Mutation Best Practices

1. **Validate Input**
```go
"createUser": func(p rocket.ResolveParams) (interface{}, error) {
    input := p.Args["input"].(map[string]interface{})
    
    // Validate before processing
    if email := input["email"].(string); !isValidEmail(email) {
        return nil, fmt.Errorf("invalid email")
    }
    
    return r.userService.CreateUser(p.Context, input)
}
```

2. **Check Permissions**
```go
"updatePost": func(p rocket.ResolveParams) (interface{}, error) {
    postID := p.Args["id"].(string)
    userID := p.Context.Value("userID").(string)
    
    // Check ownership
    post, _ := r.postService.GetPost(p.Context, postID)
    if post.AuthorID != userID {
        return nil, fmt.Errorf("forbidden: not the author")
    }
    
    return r.postService.UpdatePost(p.Context, postID, input)
}
```

3. **Use Transactions**
```go
"transferMoney": func(p rocket.ResolveParams) (interface{}, error) {
    // Start transaction
    tx, _ := r.db.BeginTx(p.Context, nil)
    defer tx.Rollback()
    
    // Debit from account
    err := r.accountService.Debit(p.Context, tx, fromID, amount)
    if err != nil {
        return nil, err
    }
    
    // Credit to account
    err = r.accountService.Credit(p.Context, tx, toID, amount)
    if err != nil {
        return nil, err
    }
    
    // Commit
    tx.Commit()
    return true, nil
}
```

4. **Return Useful Data**
```go
"createPost": func(p rocket.ResolveParams) (interface{}, error) {
    input := p.Args["input"].(map[string]interface{})
    
    // Create and return the full object
    // This allows client to get data without refetching
    post, err := r.postService.CreatePost(p.Context, input)
    return post, err
}
```

## Subscription Resolvers

Subscription resolvers return a channel that emits values:

```go
func (r *Resolvers) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
    return map[string]rocket.SubscriptionResolveFn{
        // Simple countdown
        "countdown": func(p rocket.ResolveParams) (<-chan interface{}, error) {
            from := int(p.Args["from"].(float64))
            
            ch := make(chan interface{})
            go func() {
                defer close(ch)
                for i := from; i >= 0; i-- {
                    select {
                    case <-p.Context.Done():
                        return
                    case ch <- i:
                        time.Sleep(1 * time.Second)
                    }
                }
            }()
            return ch, nil
        },
        
        // Event stream
        "messageAdded": func(p rocket.ResolveParams) (<-chan interface{}, error) {
            roomID := p.Args["roomID"].(string)
            
            // Subscribe to message events
            ch := make(chan interface{})
            go func() {
                defer close(ch)
                
                // Subscribe to event bus
                events := r.eventBus.Subscribe(p.Context, "message."+roomID)
                
                for {
                    select {
                    case <-p.Context.Done():
                        return
                    case msg := <-events:
                        ch <- msg
                    }
                }
            }()
            return ch, nil
        },
    }
}
```

### Schema
```graphql
type Subscription {
  countdown(from: Int!): Int!
  messageAdded(roomID: ID!): Message!
}

type Message {
  id: ID!
  text: String!
  userID: String!
}
```

## Type Resolvers

Type resolvers customize how specific fields on types are resolved:

```go
func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            // Custom field resolver
            "fullName": func(p rocket.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return user.FirstName + " " + user.LastName, nil
            },
            
            // Relationship resolver
            "posts": func(p rocket.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return r.postService.GetPostsByUser(p.Context, user.ID)
            },
            
            // Computed field
            "isAdmin": func(p rocket.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return user.Role == "admin", nil
            },
        },
        "Post": {
            // Lazy load author
            "author": func(p rocket.ResolveParams) (interface{}, error) {
                post := p.Source.(*Post)
                return r.userService.GetUser(p.Context, post.AuthorID)
            },
        },
    }
}
```

### Schema
```graphql
type User {
  id: ID!
  firstName: String!
  lastName: String!
  fullName: String!      # Custom resolver
  posts: [Post!]!        # Relationship resolver
  isAdmin: Boolean!      # Computed field
}

type Post {
  id: ID!
  title: String!
  authorID: ID!
  author: User!          # Relationship resolver
}
```

## Auto-Resolution

Rocket automatically resolves struct fields that match GraphQL fields:

```go
type User struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Email     string `json:"email"`
    CreatedAt string `json:"createdAt"`
}

// These fields auto-resolve - no resolver needed!
// id -> ID
// name -> Name
// email -> Email
// createdAt -> CreatedAt
```

**Only define custom resolvers when you need special logic:**
- Computed fields
- Relationships
- Data transformations
- Authorization checks

## Working with Arguments

### Type Assertions

Arguments come as `map[string]interface{}`:

```go
"user": func(p rocket.ResolveParams) (interface{}, error) {
    // String argument
    id, ok := p.Args["id"].(string)
    if !ok {
        return nil, fmt.Errorf("id must be a string")
    }
    
    // Float64 (numbers in JSON)
    age, ok := p.Args["age"].(float64)
    if ok {
        ageInt := int(age)
    }
    
    // Boolean
    active, ok := p.Args["active"].(bool)
    
    // Array
    tags, ok := p.Args["tags"].([]interface{})
    if ok {
        for _, tag := range tags {
            str := tag.(string)
        }
    }
    
    // Object (input type)
    filter, ok := p.Args["filter"].(map[string]interface{})
    
    return r.service.GetUser(p.Context, id)
}
```

### Helper Functions

```go
// Safe string extraction
func getString(args map[string]interface{}, key string) (string, bool) {
    val, ok := args[key]
    if !ok {
        return "", false
    }
    str, ok := val.(string)
    return str, ok
}

// Safe int extraction (from float64)
func getInt(args map[string]interface{}, key string) (int, bool) {
    val, ok := args[key]
    if !ok {
        return 0, false
    }
    f, ok := val.(float64)
    return int(f), ok
}

// Use in resolver
"users": func(p rocket.ResolveParams) (interface{}, error) {
    limit, ok := getInt(p.Args, "limit")
    if !ok {
        limit = 10 // default
    }
    return r.service.GetUsers(p.Context, limit)
}
```

## Error Handling

### Return Errors Directly

```go
"user": func(p rocket.ResolveParams) (interface{}, error) {
    id := p.Args["id"].(string)
    
    user, err := r.service.GetUser(p.Context, id)
    if err != nil {
        // Error is automatically added to GraphQL errors array
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    
    return user, nil
}
```

### Custom Errors

```go
type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

"user": func(p rocket.ResolveParams) (interface{}, error) {
    id := p.Args["id"].(string)
    
    user, err := r.service.GetUser(p.Context, id)
    if err != nil {
        return nil, &NotFoundError{Resource: "User", ID: id}
    }
    
    return user, nil
}
```

## Context Usage

### Passing Data via Context

```go
// In middleware
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := extractUserID(r)
        ctx := context.WithValue(r.Context(), "userID", userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// In resolver
"currentUser": func(p rocket.ResolveParams) (interface{}, error) {
    userID, ok := p.Context.Value("userID").(string)
    if !ok {
        return nil, fmt.Errorf("unauthorized")
    }
    return r.service.GetUser(p.Context, userID)
}
```

### Context Best Practices

1. **Authentication**
```go
userID := p.Context.Value("userID").(string)
```

2. **Request Tracing**
```go
traceID := p.Context.Value("traceID").(string)
```

3. **Database Transactions**
```go
tx, ok := p.Context.Value("tx").(*sql.Tx)
```

## Modular Resolvers

### Splitting by Domain

```go
// user/resolvers.go
package user

type Resolvers struct {
    service *Service
}

func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "user":  r.getUser,
        "users": r.getUsers,
    }
}

func (r *Resolvers) getUser(p rocket.ResolveParams) (interface{}, error) {
    id := p.Args["id"].(string)
    return r.service.GetUser(p.Context, id)
}
```

### Combining Modules

```go
// main.go
import (
    "github.com/jest-cloud/rocket"
    "yourapp/user"
    "yourapp/post"
)

func main() {
    userModule := user.Initialize(db)
    postModule := post.Initialize(db)
    
    // Rocket automatically merges all resolvers
    schema, _ := rocket.BuildSchema(
        rocket.Config{SchemaPath: "schema.graphql"},
        userModule.Resolvers,
        postModule.Resolvers,
    )
}
```

## Testing Resolvers

### Unit Test

```go
func TestUserResolver(t *testing.T) {
    mockService := &MockUserService{
        GetUserFunc: func(ctx context.Context, id string) (*User, error) {
            return &User{ID: id, Name: "Alice"}, nil
        },
    }
    
    resolver := &Resolvers{service: mockService}
    
    params := rocket.ResolveParams{
        Args:    map[string]interface{}{"id": "123"},
        Context: context.Background(),
    }
    
    result, err := resolver.QueryResolvers()["user"](params)
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
    
    user := result.(*User)
    assert.Equal(t, "123", user.ID)
    assert.Equal(t, "Alice", user.Name)
}
```

### Integration Test

```go
func TestResolverIntegration(t *testing.T) {
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

## Performance Tips

### 1. Avoid N+1 Queries

Use DataLoader:
```go
"posts": func(p rocket.ResolveParams) (interface{}, error) {
    user := p.Source.(*User)
    
    // Bad: Separate query for each user
    return r.service.GetPostsByUser(p.Context, user.ID)
    
    // Good: Use DataLoader to batch
    return r.postLoader.Load(p.Context, user.ID)
}
```

### 2. Use Pointers

```go
// Good - pointer
return &User{...}, nil

// Less efficient for large structs
return User{...}, nil
```

### 3. Lazy Load Relationships

Only fetch if requested in query:
```go
"posts": func(p rocket.ResolveParams) (interface{}, error) {
    // Only called if client requests posts field
    user := p.Source.(*User)
    return r.service.GetPosts(p.Context, user.ID)
}
```

## Common Patterns

### Pagination

```go
"users": func(p rocket.ResolveParams) (interface{}, error) {
    limit := 10
    offset := 0
    
    if l, ok := p.Args["limit"].(float64); ok {
        limit = int(l)
    }
    if o, ok := p.Args["offset"].(float64); ok {
        offset = int(o)
    }
    
    return r.service.GetUsers(p.Context, limit, offset)
}
```

### Filtering

```go
"users": func(p rocket.ResolveParams) (interface{}, error) {
    filter := p.Args["filter"].(map[string]interface{})
    
    role, _ := filter["role"].(string)
    active, _ := filter["active"].(bool)
    
    return r.service.FilterUsers(p.Context, role, active)
}
```

### Sorting

```go
"users": func(p rocket.ResolveParams) (interface{}, error) {
    sortBy := "createdAt"
    sortOrder := "desc"
    
    if s, ok := p.Args["sortBy"].(string); ok {
        sortBy = s
    }
    if o, ok := p.Args["sortOrder"].(string); ok {
        sortOrder = o
    }
    
    return r.service.GetUsers(p.Context, sortBy, sortOrder)
}
```

## Best Practices

1. **Keep resolvers thin** - Move business logic to services
2. **Use context for auth** - Don't pass auth in arguments
3. **Return errors** - Let GraphQL format them
4. **Type assert safely** - Always check `ok` values
5. **Use auto-resolution** - Only override when needed
6. **Test thoroughly** - Unit and integration tests
7. **Document complex logic** - Add comments
8. **Handle permissions early** - Check auth first
9. **Validate inputs** - Before calling services
10. **Use transactions** - For multi-step mutations

## Examples

See complete resolver examples in:
- [Migration Test](../../test/migration_test.go)
- [Mutation Test](../../test/mutation_test.go)
- [Subscription Test](../../test/subscription_test.go)
- [FirePrint API](../../../fire-print/apps/api/) - Production resolvers

## Related Documentation

- **[Queries](./QUERIES.md)** - Query-specific patterns
- **[Mutations](./MUTATIONS.md)** - Mutation-specific patterns
- **[Subscriptions](./SUBSCRIPTIONS.md)** - Subscription patterns
- **[Usage Guide](./USAGE.md)** - Getting started

## Quick Reference

```go
// Basic structure
type Resolvers struct {
    service *Service
}

// Queries - read operations
func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "field": func(p rocket.ResolveParams) (interface{}, error) {
            return r.service.Get(p.Context, p.Args["id"].(string))
        },
    }
}

// Mutations - write operations
func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "field": func(p rocket.ResolveParams) (interface{}, error) {
            input := p.Args["input"].(map[string]interface{})
            return r.service.Create(p.Context, input)
        },
    }
}

// Subscriptions - real-time
func (r *Resolvers) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
    return map[string]rocket.SubscriptionResolveFn{
        "field": func(p rocket.ResolveParams) (<-chan interface{}, error) {
            ch := make(chan interface{})
            go func() {
                defer close(ch)
                // Emit values to ch
            }()
            return ch, nil
        },
    }
}

// Types - custom field resolution
func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "TypeName": {
            "field": func(p rocket.ResolveParams) (interface{}, error) {
                obj := p.Source.(*Type)
                return obj.Field, nil
            },
        },
    }
}
```

