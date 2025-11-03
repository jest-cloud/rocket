# Federation Implementation Plan

## Overview

Rocket is federation-ready (built on `graphql-go-tools` DataSource pattern), but needs federation-specific features exposed to users.

## Current Status ✅

**What Rocket Already Has:**
- ✅ Built on `graphql-go-tools` (federation-capable engine)
- ✅ DataSource pattern (core of federation)
- ✅ Can be composed with other subgraphs
- ✅ Query/Mutation/Subscription support
- ✅ Nested type resolvers
- ✅ Context support

**What's Missing:**
- ❌ Federation directives (`@key`, `@extends`, `@external`, etc.)
- ❌ Entity resolvers (`__resolveReference`)
- ❌ `_entities` query
- ❌ `_service` query with SDL
- ❌ Federation-specific resolver API

## Apollo Federation Requirements

### 1. Schema Directives

```graphql
# Federation schema directives
directive @key(fields: String!) on OBJECT | INTERFACE
directive @extends on OBJECT | INTERFACE
directive @external on FIELD_DEFINITION
directive @requires(fields: String!) on FIELD_DEFINITION
directive @provides(fields: String!) on FIELD_DEFINITION
directive @shareable on OBJECT | FIELD_DEFINITION
directive @inaccessible on FIELD_DEFINITION | OBJECT | INTERFACE | UNION | ARGUMENT_DEFINITION | SCALAR | ENUM | ENUM_VALUE | INPUT_OBJECT | INPUT_FIELD_DEFINITION
directive @tag(name: String!) on FIELD_DEFINITION | INTERFACE | OBJECT | UNION | ARGUMENT_DEFINITION | SCALAR | ENUM | ENUM_VALUE | INPUT_OBJECT | INPUT_FIELD_DEFINITION
directive @override(from: String!) on FIELD_DEFINITION
```

### 2. Service Queries

```graphql
# Required by federation gateway
type Query {
  _entities(representations: [_Any!]!): [_Entity]!
  _service: _Service!
}

# Federation types
scalar _Any
scalar _FieldSet
type _Service {
  sdl: String!
}
union _Entity = User | Post | Product  # All entities
```

### 3. Entity Resolvers

```go
// User must implement entity resolvers
type EntityResolvers interface {
    // __resolveReference for each entity type
    ResolveReference(typename string, representation map[string]interface{}) (interface{}, error)
}
```

## Implementation Tasks

### Phase 1: Core Federation Support

#### Task 1.1: Add Federation Directives to Schema Builder
**File:** `schema.go` or new `internal/federation/directives.go`

```go
// Federation directive definitions
const federationDirectives = `
directive @key(fields: String!) on OBJECT | INTERFACE
directive @extends on OBJECT | INTERFACE
directive @external on FIELD_DEFINITION
directive @requires(fields: String!) on FIELD_DEFINITION
directive @provides(fields: String!) on FIELD_DEFINITION
`

// Auto-inject into schema if federation enabled
func BuildSchema(config Config, modules ...ModuleResolvers) (*Schema, error) {
    if config.Federation.Enabled {
        // Prepend federation directives to schema
        schemaBytes = append([]byte(federationDirectives), schemaBytes...)
    }
    // ... rest of build
}
```

#### Task 1.2: Add Federation Config
**File:** `rocket.go`

```go
type Config struct {
    SchemaPath       string
    EnablePlayground bool
    ContextBuilder   func(r *http.Request) context.Context
    Federation       FederationConfig  // NEW
}

type FederationConfig struct {
    Enabled    bool   // Enable federation support
    ServiceName string // Subgraph name (optional)
}
```

#### Task 1.3: Create Entity Resolver Interface
**File:** `internal/types/types.go`

```go
// EntityResolveFn resolves an entity from its representation
// Used for federation __resolveReference
type EntityResolveFn func(p ResolveParams, representation map[string]interface{}) (interface{}, error)

// ModuleResolvers interface update
type ModuleResolvers interface {
    QueryResolvers() map[string]FieldResolveFn
    MutationResolvers() map[string]FieldResolveFn
    SubscriptionResolvers() map[string]SubscriptionResolveFn
    TypeResolvers() map[string]map[string]FieldResolveFn
    EntityResolvers() map[string]EntityResolveFn  // NEW: typename -> resolver
}
```

#### Task 1.4: Implement `_entities` Query
**File:** `internal/federation/entities.go` (new)

```go
package federation

import (
    "context"
    "fmt"
    "github.com/jest-cloud/rocket/internal/types"
)

// EntitiesResolver resolves the _entities query
func EntitiesResolver(entityResolvers map[string]types.EntityResolveFn) types.FieldResolveFn {
    return func(p types.ResolveParams) (interface{}, error) {
        representations, ok := p.Args["representations"].([]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid representations argument")
        }

        results := make([]interface{}, len(representations))
        
        for i, rep := range representations {
            repMap := rep.(map[string]interface{})
            
            // Extract __typename
            typename, ok := repMap["__typename"].(string)
            if !ok {
                return nil, fmt.Errorf("missing __typename in representation")
            }
            
            // Find entity resolver
            resolver, ok := entityResolvers[typename]
            if !ok {
                return nil, fmt.Errorf("no entity resolver for type: %s", typename)
            }
            
            // Resolve entity
            entity, err := resolver(p, repMap)
            if err != nil {
                return nil, fmt.Errorf("failed to resolve %s: %w", typename, err)
            }
            
            results[i] = entity
        }
        
        return results, nil
    }
}
```

#### Task 1.5: Implement `_service` Query
**File:** `internal/federation/service.go` (new)

```go
package federation

import (
    "github.com/jest-cloud/rocket/internal/types"
)

type ServiceResponse struct {
    SDL string `json:"sdl"`
}

// ServiceResolver returns the SDL for this subgraph
func ServiceResolver(schemaSDL string) types.FieldResolveFn {
    return func(p types.ResolveParams) (interface{}, error) {
        return &ServiceResponse{
            SDL: schemaSDL,
        }, nil
    }
}
```

#### Task 1.6: Auto-Register Federation Queries
**File:** `schema.go`

```go
func BuildSchema(config Config, modules ...ModuleResolvers) (*Schema, error) {
    // ... existing schema parsing ...
    
    // Create resolver registry
    resolvers := registry.NewResolverRegistry(modules...)
    
    // If federation enabled, add federation queries
    if config.Federation.Enabled {
        // Collect entity resolvers from all modules
        entityResolvers := make(map[string]types.EntityResolveFn)
        for _, module := range modules {
            for typename, resolver := range module.EntityResolvers() {
                entityResolvers[typename] = resolver
            }
        }
        
        // Auto-register _entities and _service
        resolvers.Query["_entities"] = federation.EntitiesResolver(entityResolvers)
        resolvers.Query["_service"] = federation.ServiceResolver(string(schemaBytes))
    }
    
    // ... rest of build
}
```

### Phase 2: Helper Functions & Developer Experience

#### Task 2.1: Federation Helper Functions
**File:** `federation.go` (new public API)

```go
package rocket

// EntityResolver is a helper to create entity resolvers
func EntityResolver(typename string, resolver EntityResolveFn) (string, EntityResolveFn) {
    return typename, resolver
}

// ExtractEntityKey extracts key fields from representation
func ExtractEntityKey(rep map[string]interface{}, keyField string) (interface{}, bool) {
    value, ok := rep[keyField]
    return value, ok
}

// Example usage in user code:
// func (r *Resolvers) EntityResolvers() map[string]rocket.EntityResolveFn {
//     return map[string]rocket.EntityResolveFn{
//         "User": func(p rocket.ResolveParams, rep map[string]interface{}) (interface{}, error) {
//             id, _ := rocket.ExtractEntityKey(rep, "id")
//             return r.userService.GetUser(p.Context, id.(string))
//         },
//     }
// }
```

#### Task 2.2: Schema Validation
**File:** `internal/federation/validate.go` (new)

```go
// Validate federation schema requirements
func ValidateFederationSchema(schema *ast.Document) error {
    // Check that entities have @key directives
    // Check that extended types use @external correctly
    // Validate that _entities and _service queries exist
    // Return helpful error messages
}
```

### Phase 3: Testing & Documentation

#### Task 3.1: Federation Tests
**File:** `test/federation_test.go` (new)

```go
func TestFederationEntities(t *testing.T) {
    // Test _entities query
    // Test entity resolution
    // Test multiple entities
    // Test missing entity resolver
}

func TestFederationService(t *testing.T) {
    // Test _service query returns SDL
}

func TestFederationDirectives(t *testing.T) {
    // Test @key directive in schema
    // Test @extends, @external, etc.
}
```

#### Task 3.2: Federation Example
**File:** `examples/federation/` (new)

Directory structure:
```
examples/federation/
├── README.md
├── gateway/           # Apollo Gateway or Cosmo Router config
├── users-subgraph/    # User service (Rocket)
│   ├── main.go
│   ├── schema.graphql
│   └── resolvers.go
├── posts-subgraph/    # Post service (Rocket)
│   ├── main.go
│   ├── schema.graphql
│   └── resolvers.go
└── docker-compose.yml # Run all services
```

**users-subgraph/schema.graphql:**
```graphql
type User @key(fields: "id") {
  id: ID!
  name: String!
  email: String!
}

type Query {
  user(id: ID!): User
  users: [User!]!
}
```

**posts-subgraph/schema.graphql:**
```graphql
extend type User @key(fields: "id") {
  id: ID! @external
  posts: [Post!]!
}

type Post @key(fields: "id") {
  id: ID!
  title: String!
  content: String!
  authorId: ID!
}

type Query {
  post(id: ID!): Post
  posts: [Post!]!
}
```

**users-subgraph/main.go:**
```go
package main

import (
    "context"
    "log"
    "net/http"
    "github.com/jest-cloud/rocket"
)

type Resolvers struct {
    users map[string]*User
}

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "user": func(p rocket.ResolveParams) (interface{}, error) {
            id := p.Args["id"].(string)
            return r.users[id], nil
        },
        "users": func(p rocket.ResolveParams) (interface{}, error) {
            users := make([]*User, 0, len(r.users))
            for _, u := range r.users {
                users = append(users, u)
            }
            return users, nil
        },
    }
}

// NEW: Entity resolvers for federation
func (r *Resolvers) EntityResolvers() map[string]rocket.EntityResolveFn {
    return map[string]rocket.EntityResolveFn{
        "User": func(p rocket.ResolveParams, rep map[string]interface{}) (interface{}, error) {
            id := rep["id"].(string)
            return r.users[id], nil
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

func main() {
    // Mock data
    resolvers := &Resolvers{
        users: map[string]*User{
            "1": {ID: "1", Name: "Alice", Email: "alice@example.com"},
            "2": {ID: "2", Name: "Bob", Email: "bob@example.com"},
        },
    }

    schema, _ := rocket.BuildSchema(
        rocket.Config{
            SchemaPath: "schema.graphql",
            Federation: rocket.FederationConfig{
                Enabled:     true,
                ServiceName: "users",
            },
        },
        resolvers,
    )

    http.Handle("/graphql", rocket.Handler(schema))
    
    log.Println("🚀 Users subgraph running on :4001")
    log.Fatal(http.ListenAndServe(":4001", nil))
}
```

#### Task 3.3: Documentation Updates
**File:** `dev/docs/FEDERATION.md`

Update with:
- Complete implementation guide
- Entity resolver patterns
- Federation directive usage
- Gateway setup (Apollo Gateway / Cosmo Router)
- Complete working example
- Migration guide (standalone → federated)

## Implementation Checklist

### Core (v0.5.0)
- [ ] Add `FederationConfig` to `Config` struct
- [ ] Create `internal/federation/` package
- [ ] Implement federation directives injection
- [ ] Add `EntityResolveFn` type
- [ ] Update `ModuleResolvers` interface with `EntityResolvers()`
- [ ] Implement `_entities` query handler
- [ ] Implement `_service` query handler
- [ ] Auto-register federation queries when enabled
- [ ] Add federation schema validation

### Developer Experience (v0.5.0)
- [ ] Add helper functions for entity resolvers
- [ ] Add helper for extracting entity keys
- [ ] Clear error messages for missing entity resolvers
- [ ] Update all existing test resolvers with empty `EntityResolvers()`

### Testing (v0.5.0)
- [ ] Unit tests for `_entities` query
- [ ] Unit tests for `_service` query
- [ ] Unit tests for entity resolution
- [ ] Integration test with multiple entities
- [ ] Test error cases (missing resolver, invalid representation)

### Documentation (v0.5.0)
- [ ] Update `FEDERATION.md` with implementation guide
- [ ] Create federation example (2 subgraphs + gateway)
- [ ] Add entity resolver section to `RESOLVERS.md`
- [ ] Update `README.md` features list
- [ ] Add migration guide (standalone → federated)
- [ ] Create federation quick start guide

### Examples (v0.5.0)
- [ ] Create `examples/federation/` directory
- [ ] Users subgraph example
- [ ] Posts subgraph example
- [ ] Gateway configuration (Apollo/Cosmo)
- [ ] Docker Compose setup
- [ ] README with setup instructions

### Advanced (v0.6.0+)
- [ ] Support for `@requires` directive
- [ ] Support for `@provides` directive
- [ ] Support for `@shareable` directive (Federation 2)
- [ ] Federation 2 full support
- [ ] Automatic SDL generation for gateway
- [ ] Federation schema composition validation
- [ ] Performance optimizations for entity resolution

## Breaking Changes

The `ModuleResolvers` interface will need to add `EntityResolvers()`:

```go
// Before (v0.4.0)
type ModuleResolvers interface {
    QueryResolvers() map[string]FieldResolveFn
    MutationResolvers() map[string]FieldResolveFn
    SubscriptionResolvers() map[string]SubscriptionResolveFn
    TypeResolvers() map[string]map[string]FieldResolveFn
}

// After (v0.5.0)
type ModuleResolvers interface {
    QueryResolvers() map[string]FieldResolveFn
    MutationResolvers() map[string]FieldResolveFn
    SubscriptionResolvers() map[string]SubscriptionResolveFn
    TypeResolvers() map[string]map[string]FieldResolveFn
    EntityResolvers() map[string]EntityResolveFn  // NEW
}
```

**Migration:** Existing code needs to add empty `EntityResolvers()` method:

```go
func (r *Resolvers) EntityResolvers() map[string]rocket.EntityResolveFn {
    return map[string]rocket.EntityResolveFn{}
}
```

## Federation Gateway Options

Rocket subgraphs will work with:

1. **Apollo Gateway** - Most popular, mature
2. **WunderGraph Cosmo Router** - Modern, high-performance (Go-based)
3. **Netflix DGS Gateway** - Java ecosystem
4. **Custom Gateway** - Using `graphql-go-tools` directly

## Timeline Estimate

- **Phase 1 (Core):** 2-3 days
- **Phase 2 (DX):** 1 day
- **Phase 3 (Testing/Docs/Examples):** 2-3 days

**Total: ~1 week for v0.5.0 with full federation support**

## Success Criteria

✅ Federation is successfully implemented when:
1. Can define entities with `@key` directive
2. Can implement `__resolveReference` via `EntityResolvers()`
3. `_entities` and `_service` queries work
4. Can compose 2+ Rocket services into federated supergraph
5. Works with Apollo Gateway or Cosmo Router
6. Complete example with multiple subgraphs
7. Comprehensive documentation
8. All tests passing

## Future Enhancements (v0.6.0+)

- [ ] Federation 2.0 full support
- [ ] `@shareable` for shared types
- [ ] `@inaccessible` for internal fields
- [ ] `@override` for field migration
- [ ] Automatic entity key extraction
- [ ] Federation tracing
- [ ] Schema composition tooling
- [ ] Federation health checks

## References

- [Apollo Federation Spec](https://www.apollographql.com/docs/federation/federation-spec/)
- [WunderGraph Cosmo](https://cosmo-docs.wundergraph.com/)
- [graphql-go-tools](https://github.com/wundergraph/graphql-go-tools)

