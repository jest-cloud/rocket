# 🚀 Rocket

**Developer-friendly GraphQL for Go** - Bringing a modern, DX-focused approach to GraphQL development in Golang.

## Philosophy

Rocket aims to bring a more developer-friendly approach to GraphQL in Go:

- **Schema-First**: Define your API in `.graphql` files
- **Modular Resolvers**: Each module owns its resolvers
- **Declarative**: Map-based resolvers, no switch statements
- **Auto-Resolution**: Struct fields auto-resolve, override only when needed
- **Field Order**: Preserves query field selection order in responses
- **Type-Safe**: Leverage Go's type system with minimal boilerplate

## Features

- ✨ **TypeScript-like resolver pattern** - Map-based resolvers you can spread/merge
- 🎯 **Auto-field resolution** - Define custom resolvers only when you need them
- 📦 **Modular architecture** - Each module is self-contained
- 🔄 **Schema compilation** - Concat `.graphql` files with smart ordering
- ⚡ **Hot reload support** - Auto-recompile schemas on change
- 🎨 **Field order preservation** - Responses match query field order
- 🏗️ **Built on Wundergraph** - Production-grade GraphQL tools
- 🎮 **Apollo Sandbox** - Modern playground with best-in-class DX

## Quick Start

### 1. Define Your Schema

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

func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "user": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            return r.service.GetUserByID(p.Context, id)
        },
        "users": func(p rocket.ResolveParams) (interface{}, error) {
            limit := p.Args["limit"].(int)
            return r.service.GetAllUsers(p.Context, limit)
        },
    }
}

func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{}
}

func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            // Fields auto-resolve from User struct!
            // Only override when you need custom logic:
            "lastName": func(p rocket.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return strings.ToUpper(user.LastName), nil
            },
        },
    }
}
```

### 3. Build Schema

```go
// main.go
import "github.com/jest-cloud/rocket"

func main() {
    // Initialize modules
    userModule := user.Initialize(db)
    orgModule := org.Initialize(db)
    
    // Build schema with all resolvers
    schema, err := rocket.BuildSchema(
        rocket.Config{
            SchemaPath: "schema/schema.graphql",
        },
        userModule.Resolvers,
        orgModule.Resolvers,
    )
    
    // Create HTTP handlers
    http.Handle("/graphql", rocket.Handler(schema))
    
    // Optional: Add playground
    http.HandleFunc("/playground", rocket.PlaygroundHandler("/graphql"))
    
    http.ListenAndServe(":8080", nil)
}
```

## Comparison with TypeScript GraphQL

### TypeScript GraphQL

```typescript
const resolvers = {
  Query: {
    user: (parent, args, context) => {
      return userService.getUser(args.id);
    },
  },
  User: {
    // Fields auto-resolve!
    lastName: (parent) => parent.lastName.toUpperCase(),
  },
};
```

### Rocket (Go)

```go
func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "user": func(p rocket.ResolveParams) (interface{}, error) {
            return r.service.GetUser(p.Args["id"].(string))
        },
    }
}

func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            // Fields auto-resolve!
            "lastName": func(p rocket.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return strings.ToUpper(user.LastName), nil
            },
        },
    }
}
```

**Same pattern, same philosophy!** 🎯

## Architecture

```
Rocket Package
├── Schema Compiler      - Concatenates .graphql files
├── Schema Builder       - Parses and builds executable schema
├── Resolver Registry    - Stitches module resolvers together
├── Execution Engine     - Executes queries with field order
├── HTTP Handler         - Gin/net/http integration
└── Default Resolvers    - Auto-resolution for struct fields
```

## Why Rocket?

- 🚀 **Fast to develop** - Write less code, get more done
- 🎯 **Developer-friendly** - Intuitive patterns that reduce boilerplate
- 💪 **Production-ready** - Built on Wundergraph's battle-tested tools
- 🔧 **Flexible** - Override anything when you need to
- 📦 **Modular** - Clean separation of concerns

## Introspection Support

Rocket fully supports GraphQL introspection queries out of the box:

- `__schema` - Get the schema structure
- `__type` - Get information about a specific type  
- `__typename` - Get the type name of an object

The GraphQL Playground automatically uses introspection to provide autocomplete and documentation.

## Coming Soon

- [ ] Subscriptions support
- [ ] DataLoader for N+1 prevention
- [ ] Custom scalar types
- [ ] Field-level middleware/directives
- [ ] GraphQL Federation
- [ ] Query complexity analysis

## License

MIT

---

**Rocket**: A developer-friendly approach to GraphQL in Go 🚀

