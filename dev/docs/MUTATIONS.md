# Mutations

## Overview

Mutations are write operations in GraphQL. Rocket provides a clean, type-safe way to define mutation resolvers using the `MutationResolvers()` method.

## Status: FULLY IMPLEMENTED ✅

Mutations have been fully supported since v0.2.0 with:
- ✅ Mutation execution via HTTP POST
- ✅ Input types and variables
- ✅ Direct execution (hybrid strategy)
- ✅ Nested field resolution in results
- ✅ Transaction support via context
- ✅ Error handling

## Basic Mutation

### Schema

```graphql
type Mutation {
  createUser(input: CreateUserInput!): User!
  updateUser(id: ID!, input: UpdateUserInput!): User!
  deleteUser(id: ID!): Boolean!
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

type User {
  id: ID!
  name: String!
  email: String!
  createdAt: String!
}
```

### Resolver

```go
package main

import "github.com/jest-cloud/rocket"

type Resolvers struct {
    service *UserService
}

func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "createUser": func(p rocket.ResolveParams) (interface{}, error) {
            input := p.Args["input"].(map[string]interface{})
            
            user := &User{
                Name:  input["name"].(string),
                Email: input["email"].(string),
            }
            
            return r.service.CreateUser(p.Context, user)
        },
        
        "updateUser": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            input := p.Args["input"].(map[string]interface{})
            
            updates := make(map[string]interface{})
            if name, ok := input["name"]; ok {
                updates["name"] = name
            }
            if email, ok := input["email"]; ok {
                updates["email"] = email
            }
            
            return r.service.UpdateUser(p.Context, id, updates)
        },
        
        "deleteUser": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            err := r.service.DeleteUser(p.Context, id)
            return err == nil, err
        },
    }
}
```

## Mutation Execution

### HTTP Request

```bash
# Simple mutation
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { createUser(input: { name: \"Alice\", email: \"alice@example.com\", password: \"secret\" }) { id name email } }"
  }'

# With variables (recommended)
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation CreateUser($input: CreateUserInput!) { createUser(input: $input) { id name email } }",
    "variables": {
      "input": {
        "name": "Alice",
        "email": "alice@example.com",
        "password": "secret"
      }
    }
  }'
```

### GraphQL Playground

```graphql
# Create user
mutation CreateUser($input: CreateUserInput!) {
  createUser(input: $input) {
    id
    name
    email
    createdAt
  }
}

# Update user
mutation UpdateUser($id: ID!, $input: UpdateUserInput!) {
  updateUser(id: $id, input: $input) {
    id
    name
    email
  }
}

# Delete user
mutation DeleteUser($id: ID!) {
  deleteUser(id: $id)
}
```

Variables:
```json
{
  "input": {
    "name": "Alice",
    "email": "alice@example.com",
    "password": "secret123"
  }
}
```

## Working with Input Types

### Parsing Input

```go
"createUser": func(p rocket.ResolveParams) (interface{}, error) {
    // Input comes as map[string]interface{}
    input := p.Args["input"].(map[string]interface{})
    
    // Extract and type assert fields
    name, ok := input["name"].(string)
    if !ok {
        return nil, fmt.Errorf("name is required")
    }
    
    email, ok := input["email"].(string)
    if !ok {
        return nil, fmt.Errorf("email is required")
    }
    
    password, ok := input["password"].(string)
    if !ok {
        return nil, fmt.Errorf("password is required")
    }
    
    // Create user
    return r.service.CreateUser(p.Context, name, email, password)
}
```

### Helper Function for Input Parsing

```go
func parseCreateUserInput(input map[string]interface{}) (*CreateUserInput, error) {
    name, ok := input["name"].(string)
    if !ok || name == "" {
        return nil, fmt.Errorf("name is required")
    }
    
    email, ok := input["email"].(string)
    if !ok || email == "" {
        return nil, fmt.Errorf("email is required")
    }
    
    password, ok := input["password"].(string)
    if !ok || password == "" {
        return nil, fmt.Errorf("password is required")
    }
    
    return &CreateUserInput{
        Name:     name,
        Email:    email,
        Password: password,
    }, nil
}

// Use in resolver
"createUser": func(p rocket.ResolveParams) (interface{}, error) {
    input, err := parseCreateUserInput(p.Args["input"].(map[string]interface{}))
    if err != nil {
        return nil, err
    }
    
    return r.service.CreateUser(p.Context, input)
}
```

## Validation

### Input Validation

```go
"createUser": func(p rocket.ResolveParams) (interface{}, error) {
    input := p.Args["input"].(map[string]interface{})
    email := input["email"].(string)
    
    // Validate email format
    if !isValidEmail(email) {
        return nil, fmt.Errorf("invalid email format")
    }
    
    // Validate password strength
    password := input["password"].(string)
    if len(password) < 8 {
        return nil, fmt.Errorf("password must be at least 8 characters")
    }
    
    return r.service.CreateUser(p.Context, input)
}
```

### Business Logic Validation

```go
"createUser": func(p rocket.ResolveParams) (interface{}, error) {
    input := p.Args["input"].(map[string]interface{})
    email := input["email"].(string)
    
    // Check if user already exists
    existing, _ := r.service.GetUserByEmail(p.Context, email)
    if existing != nil {
        return nil, fmt.Errorf("user with email %s already exists", email)
    }
    
    return r.service.CreateUser(p.Context, input)
}
```

## Authentication & Authorization

### Require Authentication

```go
"createPost": func(p rocket.ResolveParams) (interface{}, error) {
    // Get user from context (set by auth middleware)
    userID, ok := p.Context.Value("userID").(string)
    if !ok {
        return nil, fmt.Errorf("unauthorized: must be logged in")
    }
    
    input := p.Args["input"].(map[string]interface{})
    return r.service.CreatePost(p.Context, userID, input)
}
```

### Check Permissions

```go
"deletePost": func(p rocket.ResolveParams) (interface{}, error) {
    userID := p.Context.Value("userID").(string)
    postID := p.Args["id"].(string)
    
    // Check if user owns the post
    post, err := r.service.GetPost(p.Context, postID)
    if err != nil {
        return nil, err
    }
    
    if post.AuthorID != userID {
        return nil, fmt.Errorf("forbidden: you can only delete your own posts")
    }
    
    err = r.service.DeletePost(p.Context, postID)
    return err == nil, err
}
```

## Transactions

### Using Database Transactions

```go
"createOrder": func(p rocket.ResolveParams) (interface{}, error) {
    input := p.Args["input"].(map[string]interface{})
    
    // Start transaction
    tx, err := r.db.BeginTx(p.Context, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    // Create order
    order, err := r.service.CreateOrder(p.Context, tx, input)
    if err != nil {
        return nil, err
    }
    
    // Update inventory
    err = r.service.UpdateInventory(p.Context, tx, order.Items)
    if err != nil {
        return nil, err
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    
    return order, nil
}
```

## Error Handling

### Structured Errors

```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

"createUser": func(p rocket.ResolveParams) (interface{}, error) {
    input := p.Args["input"].(map[string]interface{})
    
    email := input["email"].(string)
    if !isValidEmail(email) {
        return nil, &ValidationError{
            Field:   "email",
            Message: "invalid email format",
        }
    }
    
    return r.service.CreateUser(p.Context, input)
}
```

### Multiple Errors

```go
"createUser": func(p rocket.ResolveParams) (interface{}, error) {
    input := p.Args["input"].(map[string]interface{})
    
    var errors []string
    
    if name := input["name"].(string); name == "" {
        errors = append(errors, "name is required")
    }
    
    if email := input["email"].(string); !isValidEmail(email) {
        errors = append(errors, "invalid email format")
    }
    
    if len(errors) > 0 {
        return nil, fmt.Errorf("validation errors: %s", strings.Join(errors, ", "))
    }
    
    return r.service.CreateUser(p.Context, input)
}
```

## Optimistic Updates

Return the object for immediate UI updates:

```go
"updateUser": func(p rocket.ResolveParams) (interface{}, error) {
    id := p.Args["id"].(string)
    input := p.Args["input"].(map[string]interface{})
    
    // Update in database
    user, err := r.service.UpdateUser(p.Context, id, input)
    if err != nil {
        return nil, err
    }
    
    // Return updated user for optimistic UI updates
    return user, nil
}
```

## Batch Mutations

### Multiple Items

```graphql
type Mutation {
  createUsers(inputs: [CreateUserInput!]!): [User!]!
}
```

```go
"createUsers": func(p rocket.ResolveParams) (interface{}, error) {
    inputs := p.Args["inputs"].([]interface{})
    
    users := make([]*User, 0, len(inputs))
    
    for _, input := range inputs {
        inputMap := input.(map[string]interface{})
        user, err := r.service.CreateUser(p.Context, inputMap)
        if err != nil {
            return nil, fmt.Errorf("failed to create user: %w", err)
        }
        users = append(users, user)
    }
    
    return users, nil
}
```

## Testing

### Unit Test

```go
func TestCreateUserMutation(t *testing.T) {
    mockService := &MockUserService{}
    resolver := &Resolvers{service: mockService}
    
    params := rocket.ResolveParams{
        Args: map[string]interface{}{
            "input": map[string]interface{}{
                "name":     "Alice",
                "email":    "alice@example.com",
                "password": "secret123",
            },
        },
        Context: context.Background(),
    }
    
    result, err := resolver.MutationResolvers()["createUser"](params)
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
    
    user := result.(*User)
    assert.Equal(t, "Alice", user.Name)
}
```

### Integration Test

```go
func TestMutationExecution(t *testing.T) {
    schema, _ := rocket.BuildSchema(
        rocket.Config{SchemaPath: "schema.graphql"},
        &Resolvers{service: testService},
    )
    
    query := `
        mutation {
            createUser(input: {
                name: "Alice"
                email: "alice@example.com"
                password: "secret123"
            }) {
                id
                name
                email
            }
        }
    `
    
    result := schema.Execute(context.Background(), query, nil, "")
    
    assert.Empty(t, result.Errors)
    assert.NotNil(t, result.Data)
}
```

## Examples

See complete examples in:
- [Mutation Test](../../test/mutation_test.go) - Basic mutations
- [FirePrint API](../../../fire-print/apps/api/) - Production mutations with auth

## Best Practices

1. **Use input types** - Group related fields into input types
2. **Validate early** - Check inputs before calling services
3. **Use transactions** - For operations that modify multiple entities
4. **Return the result** - For optimistic UI updates
5. **Handle errors properly** - Return clear, actionable error messages
6. **Check permissions** - Verify user has access to perform mutation
7. **Idempotency** - Make mutations safe to retry when possible
8. **Audit logging** - Log important mutations for compliance

## Common Patterns

### Soft Delete

```go
"deleteUser": func(p rocket.ResolveParams) (interface{}, error) {
    id := p.Args["id"].(string)
    
    // Soft delete - set deletedAt timestamp
    err := r.service.SoftDeleteUser(p.Context, id)
    return err == nil, err
}
```

### Upsert

```go
"upsertUser": func(p rocket.ResolveParams) (interface{}, error) {
    input := p.Args["input"].(map[string]interface{})
    email := input["email"].(string)
    
    // Check if exists
    user, _ := r.service.GetUserByEmail(p.Context, email)
    
    if user != nil {
        // Update existing
        return r.service.UpdateUser(p.Context, user.ID, input)
    }
    
    // Create new
    return r.service.CreateUser(p.Context, input)
}
```

### Returning Relationships

```graphql
type Mutation {
  createPost(input: CreatePostInput!): Post!
}

type Post {
  id: ID!
  title: String!
  content: String!
  author: User!  # Return relationship
}
```

## Status Summary

| Feature | Status | Since |
|---------|--------|-------|
| Basic Mutations | ✅ Complete | v0.2.0 |
| Input Types | ✅ Complete | v0.2.0 |
| Variables | ✅ Complete | v0.2.0 |
| Direct Execution | ✅ Complete | v0.2.0 |
| Nested Results | ✅ Complete | v0.2.0 |
| Error Handling | ✅ Complete | v0.2.0 |
| Transaction Support | ✅ Complete | v0.2.0 |

**Mutations: 100% Complete** ✅

