# Rocket Usage Guide

## Installation

```bash
go get github.com/jest-cloud/rocket
```

## Basic Usage

### 1. Define Your GraphQL Schema

Create `.graphql` files in your modules:

```graphql
# src/user/schema.graphql
type User {
  id: ID!
  email: String!
  firstName: String!
  lastName: String!
}

extend type Query {
  user(id: ID!): User
  users(limit: Int): [User!]!
}
```

### 2. Create Module Resolvers

```go
// src/user/resolvers.go
package user

import "github.com/jest-cloud/rocket"

type Resolvers struct {
    service *Service
}

func NewResolvers(service *Service) *Resolvers {
    return &Resolvers{service: service}
}

// QueryResolvers returns query resolvers
func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "user": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            return r.service.GetUserByID(p.Context, id)
        },
        "users": func(p rocket.ResolveParams) (interface{}, error) {
            limit := 10
            if l, ok := p.Args["limit"].(int); ok {
                limit = l
            }
            return r.service.GetAllUsers(p.Context, limit)
        },
    }
}

// MutationResolvers returns mutation resolvers
func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        // Add your mutations here
    }
}

// TypeResolvers returns type field resolvers
func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            // Fields auto-resolve from User struct!
            // Only define custom resolvers when needed:
            "lastName": func(p rocket.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return strings.ToUpper(user.LastName), nil
            },
        },
    }
}
```

### 3. Compile Schema

Add a Makefile target:

```makefile
schema:
	@go run tools/compile-schema/main.go
```

Or use Rocket's compiler directly:

```go
// tools/compile-schema/main.go
package main

import "github.com/jest-cloud/rocket"

func main() {
    compiler := rocket.NewSchemaCompiler("src", "schema/schema.graphql")
    if err := compiler.Compile(); err != nil {
        panic(err)
    }
}
```

### 4. Build Schema and Start Server

```go
// main.go
package main

import (
    "log"
    "net/http"
    
    "github.com/jest-cloud/rocket"
    "myapp/src/user"
    "myapp/src/org"
)

func main() {
    // Initialize modules
    userModule := user.Initialize(db)
    orgModule := org.Initialize(db)
    
    // Build GraphQL schema
    schema, err := rocket.BuildSchema(
        rocket.Config{
            SchemaPath: "schema/schema.graphql",
        },
        userModule.Resolvers,
        orgModule.Resolvers,
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Create HTTP handler
    http.Handle("/graphql", rocket.Handler(schema))
    
    log.Println("GraphQL server running on :8080")
    http.ListenAndServe(":8080", nil)
}
```

## Module Structure

Each module should follow this pattern:

```
src/user/
├── model.go       # Domain models
├── service.go     # Business logic
├── resolvers.go   # GraphQL resolvers (implements rocket.ModuleResolvers)
├── init.go        # Module initialization
└── schema.graphql # GraphQL schema definitions
```

### Module Initialization

```go
// src/user/init.go
package user

type Module struct {
    Service   *Service
    Resolvers *Resolvers
}

func Initialize(db *database.Service) *Module {
    service := NewService(db.GetCollection("users"))
    resolvers := NewResolvers(service)
    
    return &Module{
        Service:   service,
        Resolvers: resolvers,
    }
}
```

## Auto-Recompilation with Air

Add Rocket's compiler to your `.air.toml`:

```toml
[build]
  pre_cmd = ["mkdir -p ./tmp", "go run tools/compile-if-needed/main.go"]
  include_ext = ["go", "graphql"]
```

Create the conditional compiler:

```go
// tools/compile-if-needed/main.go
package main

import (
    "fmt"
    "github.com/jest-cloud/rocket"
)

func main() {
    compiler := rocket.NewSchemaCompiler("src", "schema/schema.graphql")
    recompiled, err := compiler.CompileIfNeeded()
    if err != nil {
        panic(err)
    }
    
    if recompiled {
        fmt.Println("GraphQL schema recompiled")
    } else {
        fmt.Println("Schema up to date")
    }
}
```

## Features

### Auto Field Resolution

Fields automatically resolve from struct fields. No boilerplate needed!

```go
type User struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    FirstName string `json:"firstName"`
    LastName  string `json:"lastName"`
}

// All fields auto-resolve! No code needed.
```

### Custom Field Resolvers

Override only when you need custom logic:

```go
func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            // Only define when you need custom behavior
            "fullName": func(p rocket.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return user.FirstName + " " + user.LastName, nil
            },
        },
    }
}
```

### Field Order Preservation

Responses maintain the exact field order from your query:

```graphql
query {
  user(id: "123") {
    id
    email
    firstName
    lastName
  }
}
```

Response preserves order: `id`, `email`, `firstName`, `lastName` ✅

## Adding a New Module

1. Create module structure in `src/inventory/`
2. Implement `rocket.ModuleResolvers` interface
3. Add ONE line to your schema builder:

```go
schema, err := rocket.BuildSchema(
    rocket.Config{SchemaPath: "schema/schema.graphql"},
    userModule.Resolvers,
    orgModule.Resolvers,
    inventoryModule.Resolvers,  // ← Just add it!
)
```

That's it! Rocket handles the rest automatically.

## Advanced Features

### With Gin

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/jest-cloud/rocket"
)

router := gin.Default()
router.POST("/graphql", gin.WrapH(rocket.Handler(schema)))
router.GET("/graphql", gin.WrapH(rocket.Handler(schema)))
```

### Cross-Module Resolvers

When a resolver needs data from another module:

```go
type OrgResolvers struct {
    service     *Service
    userService *user.Service  // Inject other services
}

func (r *OrgResolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "OrgRegisterResponse": {
            "adminUser": func(p rocket.ResolveParams) (interface{}, error) {
                result := p.Source.(*OrgRegisterResult)
                return r.userService.GetUserByID(p.Context, result.UserID)
            },
        },
    }
}
```

## Best Practices

1. **Keep resolvers thin** - Business logic belongs in services
2. **One module, one concern** - Don't cross module boundaries except when necessary
3. **Use struct tags** - `json:"fieldName"` for proper field mapping
4. **Compile before commit** - Run `make schema` before committing changes
5. **Return pointers for objects** - Makes nil handling cleaner

## Troubleshooting

### Field not resolving

Check:
- Struct field is exported (PascalCase)
- Field name matches GraphQL field (use `json` tags)
- No typos in field name

### Resolver not called

Check:
- Resolver is registered in the correct map (Query/Mutation/Types)
- Field name matches schema exactly (case-sensitive)
- Module is passed to `BuildSchema()`

## Philosophy

Rocket believes GraphQL in Go should be developer-friendly:

✅ **Declarative** - Maps, not switch statements  
✅ **Convention over configuration** - Sensible defaults  
✅ **DRY** - Don't repeat your struct in resolvers  
✅ **Modular** - Each domain owns its schema and resolvers  
✅ **Type-safe** - Leverage Go's type system  

---

**Rocket: A developer-friendly approach to GraphQL in Go** 🚀

