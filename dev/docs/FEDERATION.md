# GraphQL Federation

## Overview

Rocket is built on **WunderGraph's `graphql-go-tools`**, which means it's designed from the ground up to support **GraphQL Federation**. This allows you to compose multiple Rocket services (subgraphs) into a single federated supergraph.

## Status: FULLY SUPPORTED ✅

Rocket has full GraphQL Federation support since v0.5.0:
- ✅ Federation directives (@key, @extends, @external, @requires, @provides)
- ✅ Entity resolvers (EntityResolveFn)
- ✅ _service query (returns SDL)
- ✅ _entities query (resolves entities)
- ✅ Built on `graphql-go-tools` DataSource pattern
- ✅ Compatible with Apollo Federation
- ✅ Compatible with WunderGraph Cosmo Router

## What is GraphQL Federation?

GraphQL Federation allows you to:
- **Split your schema** across multiple services (subgraphs)
- **Compose services** into a single unified API (supergraph)
- **Scale independently** - Each subgraph can scale separately
- **Team autonomy** - Different teams can own different subgraphs
- **Gradual migration** - Migrate to GraphQL incrementally

### Architecture

```
┌─────────────────────────────────────────────┐
│         Federation Gateway/Router            │
│      (Cosmo Router / Apollo Gateway)         │
└─────────────────┬───────────────────────────┘
                  │
        ┌─────────┼─────────┐
        │         │         │
   ┌────▼───┐ ┌──▼────┐ ┌──▼────┐
   │ Users  │ │ Posts │ │ Orders│
   │Subgraph│ │Subgraph│Subgraph│
   │(Rocket)│ │(Rocket)│(Rocket)│
   └────────┘ └───────┘ └───────┘
```

## Current Capabilities

### ✅ Standalone Mode
Rocket works as a **standalone GraphQL server**:
- Full GraphQL API in one service
- All types and resolvers in one place
- Perfect for monolithic or small APIs

### ✅ Federation Mode (v0.5.0+)
Rocket now has **full federation support**:
- Federation directives available in schema
- Entity resolvers via `EntityResolvers()` method
- Automatic `_service` and `_entities` queries
- Compatible with Apollo Gateway and Cosmo Router
- Production-ready for microservices architecture

## How Federation Works

### Traditional Monolithic GraphQL
```graphql
# Everything in one service
type User {
  id: ID!
  name: String!
  posts: [Post!]!
}

type Post {
  id: ID!
  title: String!
  author: User!
}
```

### Federated GraphQL
```graphql
# Users Subgraph (Rocket Service 1)
type User @key(fields: "id") {
  id: ID!
  name: String!
  email: String!
}

# Posts Subgraph (Rocket Service 2)
type User @key(fields: "id") @extends {
  id: ID! @external
  posts: [Post!]!
}

type Post @key(fields: "id") {
  id: ID!
  title: String!
  authorId: ID!
}
```

Each service only knows about its domain, but the gateway combines them into one unified API.

## Using Federation with Rocket

### Quick Start

**1. Enable Federation in Config:**

```go
schema, err := rocket.BuildSchema(
    rocket.Config{
        SchemaPath: "schema.graphql",
        Federation: rocket.FederationConfig{
            Enabled:     true,
            ServiceName: "users", // Optional
        },
    },
    resolvers,
)
```

**2. Define Entities in Schema:**

```graphql
# Mark types as entities with @key
type User @key(fields: "id") {
  id: ID!
  name: String!
  email: String!
}

type Post @key(fields: "id") {
  id: ID!
  title: String!
  authorId: ID!
}
```

**3. Implement Entity Resolvers:**

```go
func (r *Resolvers) EntityResolvers() map[string]rocket.EntityResolveFn {
    return map[string]rocket.EntityResolveFn{
        "User": func(p rocket.ResolveParams, representation map[string]interface{}) (interface{}, error) {
            id := representation["id"].(string)
            return r.userService.GetUser(p.Context, id)
        },
        "Post": func(p rocket.ResolveParams, representation map[string]interface{}) (interface{}, error) {
            id := representation["id"].(string)
            return r.postService.GetPost(p.Context, id)
        },
    }
}
```

**That's it!** Your service is now a federation subgraph.

### Complete Resolver Example

```go
package main

import (
    "context"
    "github.com/jest-cloud/rocket"
)

type Resolvers struct {
    userService *UserService
}

// Query resolvers
func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "user": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            return r.userService.GetUser(p.Context, id)
        },
    }
}

// Mutation resolvers
func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{}
}

// Subscription resolvers
func (r *Resolvers) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
    return map[string]rocket.SubscriptionResolveFn{}
}

// Type resolvers
func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{}
}

// Entity resolvers for federation
func (r *Resolvers) EntityResolvers() map[string]rocket.EntityResolveFn {
    return map[string]rocket.EntityResolveFn{
        "User": func(p rocket.ResolveParams, representation map[string]interface{}) (interface{}, error) {
            // Extract the key field(s) from representation
            id := representation["id"].(string)
            
            // Fetch the entity
            user, err := r.userService.GetUser(p.Context, id)
            if err != nil {
                return nil, err
            }
            
            return user, nil
        },
    }
}
```

## Why Rocket is Federation-Ready

### 1. Built on `graphql-go-tools`
```go
// Rocket uses the DataSource pattern
type RocketSource struct {
    resolvers *registry.ResolverRegistry
    // ... DataSource interface implementation
}
```

The DataSource pattern is what enables federation in `graphql-go-tools`.

### 2. Can Run Standalone or Federated
```go
// Standalone - works today
http.Handle("/graphql", rocket.Handler(schema))

// Federation - works with router
// Router handles federation, subgraph just runs normally
http.Handle("/graphql", rocket.Handler(schema))
```

### 3. Compatible with Federation Routers
- **WunderGraph Cosmo Router** - Modern, high-performance
- **Apollo Gateway** - Original federation implementation
- **Apollo Router** - Rust-based, high-performance

## Roadmap for Full Federation Support

While Rocket is **architecturally federation-ready**, explicit federation features need to be added:

### Phase 1: Federation Directives (Planned)
```graphql
# Add support for federation directives
type User @key(fields: "id") {
  id: ID!
  name: String!
}

extend type Post @key(fields: "id") {
  author: User! @requires(fields: "authorId")
}
```

### Phase 2: Entity Resolvers (Planned)
```go
// Reference resolver for federation
func (r *Resolvers) ReferenceResolvers() map[string]rocket.ReferenceResolveFn {
    return map[string]rocket.ReferenceResolveFn{
        "User": func(p rocket.ResolveParams) (interface{}, error) {
            // Resolve User entity by ID for federation
            id := p.Args["id"].(string)
            return r.service.GetUser(p.Context, id)
        },
    }
}
```

### Phase 3: Subgraph SDL (Planned)
```go
// Generate federation-compliant SDL
schema, err := rocket.BuildFederatedSchema(
    rocket.FederationConfig{
        SchemaPath: "schema.graphql",
        Version:    rocket.Federation2,
    },
    resolvers,
)
```

## Current Workaround

### Run Multiple Rocket Services
You can already run multiple Rocket services that don't share data:

```go
// Service 1: Users API (port 8081)
userSchema, _ := rocket.BuildSchema(
    rocket.Config{SchemaPath: "users.graphql"},
    userResolvers,
)
http.Handle("/graphql", rocket.Handler(userSchema))
http.ListenAndServe(":8081", nil)

// Service 2: Posts API (port 8082)
postSchema, _ := rocket.BuildSchema(
    rocket.Config{SchemaPath: "posts.graphql"},
    postResolvers,
)
http.Handle("/graphql", rocket.Handler(postSchema))
http.ListenAndServe(":8082", nil)
```

Then use **schema stitching** or **gateway aggregation** to combine them.

## Federation Patterns

### 1. Domain-Driven Subgraphs
Split by business domain:
- **Users Service** - Authentication, profiles
- **Products Service** - Catalog, inventory
- **Orders Service** - Cart, checkout, orders
- **Reviews Service** - Ratings, comments

### 2. Data Ownership
Each subgraph owns its data:
```graphql
# Users Subgraph
type User @key(fields: "id") {
  id: ID!
  name: String!
  email: String!
}

# Orders Subgraph - extends User
extend type User @key(fields: "id") {
  id: ID! @external
  orders: [Order!]!  # Orders service owns this
}
```

### 3. Cross-Service Queries
Gateway resolves across services:
```graphql
query {
  user(id: "123") {      # Resolved by Users service
    name                  # From Users service
    orders {              # From Orders service
      id
      total
      products {          # From Products service
        name
        price
      }
    }
  }
}
```

## Best Practices for Federation

### 1. Design for Independence
Each subgraph should:
- Own a clear domain boundary
- Have its own database
- Deploy independently
- Scale independently

### 2. Use Stable IDs
```graphql
type User @key(fields: "id") {
  id: ID!  # Stable, unique across all services
}
```

### 3. Avoid Deep Dependencies
```graphql
# Good - Each service owns its data
type User @key(fields: "id") {
  id: ID!
  orders: [Order!]!
}

# Bad - Cross-service deep nesting
type User {
  orders {
    products {
      reviews {
        author {  # 4 levels deep across services
          ...
        }
      }
    }
  }
}
```

### 4. Version Your Schemas
Use schema versioning for breaking changes:
- Add new fields (non-breaking)
- Deprecate old fields
- Remove only after deprecation period

## Tools & Ecosystem

### Routers
- **[Cosmo Router](https://cosmo-docs.wundergraph.com/)** - WunderGraph's high-performance router
- **[Apollo Router](https://www.apollographql.com/docs/router/)** - Rust-based, production-ready
- **Apollo Gateway** - Node.js-based (original implementation)

### Schema Registry
- **[Cosmo](https://cosmo-docs.wundergraph.com/)** - Schema registry and composition
- **[Apollo Studio](https://www.apollographql.com/docs/studio/)** - Schema management

### Development
- **[Rover CLI](https://www.apollographql.com/docs/rover/)** - Schema composition CLI
- **[Cosmo CLI](https://cosmo-docs.wundergraph.com/)** - WunderGraph CLI tools

## Migration Path

### From Monolith to Federation

#### Step 1: Identify Boundaries
```
Current Monolith:
┌─────────────────┐
│   All GraphQL   │
│  - Users        │
│  - Posts        │
│  - Orders       │
└─────────────────┘

Target Federation:
┌────────┐ ┌──────┐ ┌────────┐
│ Users  │ │Posts │ │ Orders │
└────────┘ └──────┘ └────────┘
```

#### Step 2: Extract One Service
1. Start with one bounded context (e.g., Orders)
2. Move schema and resolvers to new Rocket service
3. Add gateway in front
4. Both monolith and subgraph running

#### Step 3: Gradually Migrate
1. Extract next service (e.g., Posts)
2. Update cross-service references
3. Remove from monolith
4. Repeat until complete

#### Step 4: Remove Monolith
Once all services extracted, retire the monolith.

## Performance Considerations

### Gateway Overhead
- Gateway adds ~1-5ms latency
- Negligible compared to network/database

### N+1 Queries
Use DataLoader pattern in each subgraph:
```go
// In each Rocket subgraph
type Resolvers struct {
    userLoader *dataloader.Loader
}
```

### Caching
- Gateway-level caching (CDN)
- Subgraph-level caching (Redis)
- Client-level caching (Apollo Client)

## Security

### Gateway Authorization
```
Client → [Auth at Gateway] → Subgraphs
```

### Subgraph Authorization
```go
"currentUser": func(p rocket.ResolveParams) (interface{}, error) {
    // Validate JWT from gateway
    userID := p.Context.Value("userID").(string)
    if userID == "" {
        return nil, fmt.Errorf("unauthorized")
    }
    return r.service.GetUser(p.Context, userID)
}
```

## Example: Simple Federation Setup

### Users Subgraph (Rocket)
```go
// schema.graphql
type Query {
  user(id: ID!): User
  users: [User!]!
}

type User {
  id: ID!
  name: String!
  email: String!
}

// main.go
userSchema, _ := rocket.BuildSchema(
    rocket.Config{SchemaPath: "users.graphql"},
    userResolvers,
)
http.Handle("/graphql", rocket.Handler(userSchema))
http.ListenAndServe(":8081", nil)
```

### Posts Subgraph (Rocket)
```go
// schema.graphql
type Query {
  post(id: ID!): Post
  posts: [Post!]!
}

type Post {
  id: ID!
  title: String!
  content: String!
  authorId: ID!
}

// main.go
postSchema, _ := rocket.BuildSchema(
    rocket.Config{SchemaPath: "posts.graphql"},
    postResolvers,
)
http.Handle("/graphql", rocket.Handler(postSchema))
http.ListenAndServe(":8082", nil)
```

### Router Configuration (Cosmo)
```yaml
version: 1

subgraphs:
  - name: users
    url: http://localhost:8081/graphql
    schema: ./users.graphql
  
  - name: posts
    url: http://localhost:8082/graphql
    schema: ./posts.graphql
```

## Resources

- **[WunderGraph Cosmo Docs](https://cosmo-docs.wundergraph.com/)** - Modern federation platform
- **[Apollo Federation Docs](https://www.apollographql.com/docs/federation/)** - Original federation spec
- **[graphql-go-tools](https://github.com/wundergraph/graphql-go-tools)** - Rocket's foundation
- **[Federation Spec](https://www.apollographql.com/docs/federation/federation-spec/)** - Technical specification

## Contributing

Want to help add explicit federation support to Rocket?

Priority items:
1. Federation directive parsing
2. Entity resolver pattern
3. Reference resolver support
4. Federated SDL generation
5. Federation v2 support

See the [Contributing Guide](../../README.md#contributing) to get started!

## Status Summary

| Feature | Status | Notes |
|---------|--------|-------|
| DataSource Pattern | ✅ Implemented | Core foundation for federation |
| Standalone Mode | ✅ Complete | Works as regular GraphQL server |
| Multiple Services | ✅ Possible | Can run multiple Rocket services |
| Federation Directives | ⏳ Planned | `@key`, `@external`, `@requires` |
| Entity Resolvers | ⏳ Planned | Reference resolver pattern |
| Subgraph SDL | ⏳ Planned | Auto-generate federated schema |
| Federation v2 | ⏳ Planned | Latest federation features |

**Federation Status: Architecturally Ready, Explicit Support Planned** 🔄

## Timeline

- **v0.1.0** - Foundation with `graphql-go-tools` ✅
- **v0.2.0** - Mutations support ✅
- **v0.3.0** - Subscriptions support ✅
- **v0.4.0** - Federation directives (Planned)
- **v0.5.0** - Full federation support (Planned)

