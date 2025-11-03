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
- 🏗️ **Built on WunderGraph** - Production-grade `graphql-go-tools` for federation
- 🎮 **Apollo Sandbox** - Modern playground with best-in-class DX
- 🔄 **GraphQL Subscriptions** - Real-time updates via WebSocket with `graphql-ws` protocol
- 🌐 **Federation-ready** - Built on DataSource pattern for federated supergraphs

## Getting Started

### Prerequisites

- **Go 1.25+** - Make sure you have Go installed ([download here](https://go.dev/dl/))
- Basic understanding of GraphQL (helpful but not required)

### Installation

Add Rocket to your Go project:

```bash
go get github.com/jest-cloud/rocket@latest
```

Or specify a version:

```bash
go get github.com/jest-cloud/rocket@v0.3.0
```

### Create a Simple GraphQL API

Follow these steps to create your first Rocket-powered GraphQL API:

#### Step 1: Create Your Project

```bash
mkdir my-graphql-api
cd my-graphql-api
go mod init my-graphql-api
go get github.com/jest-cloud/rocket
```

#### Step 2: Define Your Schema

Create a `schema.graphql` file:

```graphql
type Query {
  hello: String!
  users: [User!]!
}

type User {
  id: ID!
  name: String!
  email: String!
}
```

#### Step 3: Create Your Resolvers

Create `resolvers.go`:

```go
package main

import (
    "github.com/jest-cloud/rocket"
)

type Resolvers struct{}

// User represents a user in our system
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "hello": func(p rocket.ResolveParams) (interface{}, error) {
            return "Hello, Rocket! 🚀", nil
        },
        "users": func(p rocket.ResolveParams) (interface{}, error) {
            // Return structs - Rocket will auto-resolve fields!
            return []*User{
                {ID: "1", Name: "Alice", Email: "alice@example.com"},
                {ID: "2", Name: "Bob", Email: "bob@example.com"},
            }, nil
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
    // For list types, you may need explicit type resolvers
    // Auto-resolution works best for single objects, not slices
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            "id": func(p rocket.ResolveParams) (interface{}, error) {
                switch v := p.Source.(type) {
                case *User:
                    return v.ID, nil
                case User:
                    return v.ID, nil
                default:
                    return nil, nil
                }
            },
            "name": func(p rocket.ResolveParams) (interface{}, error) {
                switch v := p.Source.(type) {
                case *User:
                    return v.Name, nil
                case User:
                    return v.Name, nil
                default:
                    return nil, nil
                }
            },
            "email": func(p rocket.ResolveParams) (interface{}, error) {
                switch v := p.Source.(type) {
                case *User:
                    return v.Email, nil
                case User:
                    return v.Email, nil
                default:
                    return nil, nil
                }
            },
        },
    }
}
```

> **Note**: Rocket's auto-field resolution works with **structs**, not maps. Use structs with `json` tags to match your GraphQL schema field names.
> 
> **Important**: When returning slices/lists, you may need to add explicit type resolvers for the fields. This is a known limitation being investigated. See the working example below with explicit resolvers.

#### Step 4: Build and Run

Create `main.go`:

```go
package main

import (
    "log"
    "net/http"
    "github.com/jest-cloud/rocket"
)

func main() {
    resolvers := &Resolvers{}
    
    // Build GraphQL schema
    schema, err := rocket.BuildSchema(
        rocket.Config{
            SchemaPath: "schema.graphql",
        },
        resolvers,
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Create HTTP handlers
    http.Handle("/graphql", rocket.Handler(schema))
    http.HandleFunc("/playground", rocket.PlaygroundHandler("/graphql"))
    
    log.Println("🚀 Server starting on http://localhost:8080")
    log.Println("🚀 GraphQL endpoint: http://localhost:8080/graphql")
    log.Println("🚀 Playground: http://localhost:8080/playground")
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

#### Step 5: Test Your API

Run your server:

```bash
go run main.go
```

Then:

1. **Visit the Playground**: Open http://localhost:8080/playground in your browser
2. **Try a Query**: 
   ```graphql
   query {
     hello
     users {
       id
       name
       email
     }
   }
   ```

That's it! You now have a working GraphQL API powered by Rocket. 🎉

### Next Steps

- 📖 Read the [Usage Guide](dev/docs/USAGE.md) for detailed examples
- 🎮 Learn about [Playground Options](dev/docs/PLAYGROUNDS.md)
- 🏗️ Explore [Module Architecture](#architecture) patterns
- 🔧 Check out the [TODO](#todo) list for upcoming features

## Documentation

- **[Usage Guide](dev/docs/USAGE.md)** - Comprehensive guide on using Rocket
- **[Playgrounds](dev/docs/PLAYGROUNDS.md)** - GraphQL playground options and configuration
- **[Subscriptions](dev/docs/SUBSCRIPTIONS.md)** - Real-time subscriptions with WebSockets
- **[CHANGELOG](dev/docs/CHANGELOG.md)** - Release history and changes
- **[Examples](examples/)** - Working examples including subscriptions

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

func (r *Resolvers) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
    return map[string]rocket.SubscriptionResolveFn{}
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

## What's Supported

### ✅ All 3 GraphQL Operation Types

| Operation | Status | Transport |
|-----------|--------|-----------|
| **Query** | ✅ Complete | HTTP POST |
| **Mutation** | ✅ Complete | HTTP POST |
| **Subscription** | ✅ Complete | WebSocket (`graphql-ws`) |

### ✅ Introspection

Rocket fully supports GraphQL introspection queries:
- `__schema` - Get the schema structure
- `__type` - Get information about a specific type  
- `__typename` - Get the type name of an object

The GraphQL Playground automatically uses introspection for autocomplete and documentation.

## TODO

### Testing & Quality
- [ ] Add comprehensive unit tests for core functionality
- [ ] Add integration tests for schema building and execution
- [ ] Add benchmark tests for performance monitoring
- [ ] Add test coverage reporting
- [ ] Set up test examples/demos in separate examples directory
- [ ] Add fuzzing for schema parsing and execution

### Error Handling & Validation
- [ ] Improve error messages with better context
- [ ] Add validation for resolver function signatures
- [ ] Add schema validation warnings
- [ ] Better error handling for malformed queries
- [ ] Add panic recovery middleware for resolvers

### Features & Functionality
- [x] Subscriptions support (WebSocket) - ✅ **Added in v0.3.0**
- [ ] DataLoader implementation for N+1 query prevention
- [ ] Custom scalar types (Date, DateTime, JSON, etc.)
- [ ] Field-level middleware/directives
- [ ] GraphQL Federation support
- [ ] Query complexity analysis
- [ ] Query depth limiting
- [ ] Rate limiting support
- [ ] Caching layer for queries
- [ ] Batch request support

### Developer Experience
- [ ] GoDoc comments for all public APIs
- [ ] Code generation tool for resolver boilerplate
- [ ] Schema validation CLI tool
- [ ] Better type safety for ResolveParams Args
- [ ] Middleware support for resolvers
- [ ] Context propagation utilities
- [ ] Logging integration hooks
- [ ] Metrics/observability hooks

### Performance
- [ ] Optimize field order preservation
- [ ] Add query result caching
- [ ] Parallel resolver execution where safe
- [ ] Reduce allocations in hot paths
- [ ] Profile and optimize schema building

### Documentation
- [ ] Add Go examples to godoc
- [ ] Create interactive tutorials
- [ ] Add migration guide from other GraphQL libraries
- [ ] Add troubleshooting guide
- [ ] API reference documentation
- [ ] Best practices guide

### Infrastructure
- [ ] Add Go 1.26+ support
- [ ] Test compatibility across Go versions
- [ ] Add GitHub Actions for automated releases
- [ ] Set up automated dependency updates (Dependabot)
- [ ] Add security scanning (CodeQL)

## Coming Soon

These are planned for future releases:

- [x] ~~Subscriptions support~~ - ✅ **Added in v0.3.0**
- [ ] DataLoader for N+1 prevention
- [ ] Custom scalar types
- [ ] Field-level middleware/directives
- [ ] GraphQL Federation
- [ ] Query complexity analysis

## Contributing

We welcome contributions from anyone who would like to get involved! Rocket is an open-source project and we'd love your help making it better.

### How to Contribute

- **Report bugs** - Open an issue if you find a bug
- **Suggest features** - Share your ideas for improvements
- **Submit pull requests** - Pick any item from the TODO list above, or fix bugs
- **Improve documentation** - Help make our docs clearer and more comprehensive
- **Write tests** - Increase test coverage and improve reliability

### Getting Started

1. Fork the repository
2. Create a branch for your contribution (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests if applicable
5. Ensure all tests pass (`go test ./...`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to your branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

Check out the [TODO](#todo) section above for ideas on what to work on, or feel free to contribute in any way you'd like!

## License

MIT

---

**Rocket**: A developer-friendly approach to GraphQL in Go 🚀

