# Federation Roadmap

## Quick Summary

**Goal:** Add full Apollo Federation support to Rocket  
**Timeline:** ~1 week  
**Version:** v0.5.0

## What's Needed

### 3 Core Components

```
1. Federation Directives
   ↓
2. Entity Resolvers (__resolveReference)
   ↓
3. Special Queries (_entities, _service)
```

## Current vs. Target State

### Current (v0.4.0) ✅
- Built on federation-capable engine (`graphql-go-tools`)
- DataSource pattern (foundation for federation)
- Can run as standalone service
- Full GraphQL support (Query/Mutation/Subscription)

### Target (v0.5.0) 🎯
- Federation directives (`@key`, `@extends`, `@external`)
- Entity resolvers (`EntityResolvers()` method)
- `_entities` query (resolve entities by representation)
- `_service` query (return SDL)
- Complete example with 2+ subgraphs
- Works with Apollo Gateway / Cosmo Router

## Implementation Overview

### 1. Add Federation Config

```go
schema, _ := rocket.BuildSchema(
    rocket.Config{
        SchemaPath: "schema.graphql",
        Federation: rocket.FederationConfig{
            Enabled: true,
            ServiceName: "users",
        },
    },
    resolvers,
)
```

### 2. Update ModuleResolvers Interface

```go
type ModuleResolvers interface {
    QueryResolvers() map[string]FieldResolveFn
    MutationResolvers() map[string]FieldResolveFn
    SubscriptionResolvers() map[string]SubscriptionResolveFn
    TypeResolvers() map[string]map[string]FieldResolveFn
    EntityResolvers() map[string]EntityResolveFn  // NEW
}
```

### 3. Implement Entity Resolvers

```go
func (r *Resolvers) EntityResolvers() map[string]rocket.EntityResolveFn {
    return map[string]rocket.EntityResolveFn{
        "User": func(p rocket.ResolveParams, rep map[string]interface{}) (interface{}, error) {
            id := rep["id"].(string)
            return r.userService.GetUser(p.Context, id)
        },
        "Post": func(p rocket.ResolveParams, rep map[string]interface{}) (interface{}, error) {
            id := rep["id"].(string)
            return r.postService.GetPost(p.Context, id)
        },
    }
}
```

### 4. Define Entities in Schema

```graphql
# Users Subgraph
type User @key(fields: "id") {
  id: ID!
  name: String!
  email: String!
}

# Posts Subgraph
extend type User @key(fields: "id") {
  id: ID! @external
  posts: [Post!]!
}

type Post @key(fields: "id") {
  id: ID!
  title: String!
  authorId: ID!
}
```

## Architecture

### Standalone (Current)
```
┌─────────────────┐
│   Client        │
└────────┬────────┘
         │
┌────────▼────────┐
│  Rocket API     │
│  (All types)    │
└─────────────────┘
```

### Federated (Target)
```
┌─────────────────┐
│   Client        │
└────────┬────────┘
         │
┌────────▼────────┐
│  Gateway/Router │
│ (Apollo/Cosmo)  │
└────┬──────┬─────┘
     │      │
┌────▼──┐ ┌─▼────┐
│Users  │ │Posts │
│Rocket │ │Rocket│
└───────┘ └──────┘
```

## Implementation Tasks

### Phase 1: Core (3 days)
- [ ] Add `FederationConfig` to Config
- [ ] Create `internal/federation/` package
- [ ] Inject federation directives into schema
- [ ] Add `EntityResolveFn` type
- [ ] Update `ModuleResolvers` interface
- [ ] Implement `_entities` query handler
- [ ] Implement `_service` query handler
- [ ] Auto-register federation queries

### Phase 2: Developer Experience (1 day)
- [ ] Helper functions for entity resolvers
- [ ] Entity key extraction helpers
- [ ] Clear error messages
- [ ] Schema validation

### Phase 3: Testing & Documentation (3 days)
- [ ] Unit tests for all federation features
- [ ] Integration tests with multiple entities
- [ ] Complete federation example (2 subgraphs + gateway)
- [ ] Update FEDERATION.md with implementation guide
- [ ] Update RESOLVERS.md with entity resolver section
- [ ] Migration guide (standalone → federated)

## Example: Users + Posts Federation

### Users Service (Port 4001)

**schema.graphql:**
```graphql
type User @key(fields: "id") {
  id: ID!
  name: String!
  email: String!
}

type Query {
  user(id: ID!): User
}
```

**resolvers.go:**
```go
func (r *Resolvers) EntityResolvers() map[string]rocket.EntityResolveFn {
    return map[string]rocket.EntityResolveFn{
        "User": func(p rocket.ResolveParams, rep map[string]interface{}) (interface{}, error) {
            id := rep["id"].(string)
            return r.users[id], nil
        },
    }
}
```

### Posts Service (Port 4002)

**schema.graphql:**
```graphql
extend type User @key(fields: "id") {
  id: ID! @external
  posts: [Post!]!
}

type Post @key(fields: "id") {
  id: ID!
  title: String!
  authorId: ID!
}

type Query {
  post(id: ID!): Post
}
```

**resolvers.go:**
```go
func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            "posts": func(p rocket.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return r.getPostsByUser(user.ID)
            },
        },
    }
}

func (r *Resolvers) EntityResolvers() map[string]rocket.EntityResolveFn {
    return map[string]rocket.EntityResolveFn{
        "Post": func(p rocket.ResolveParams, rep map[string]interface{}) (interface{}, error) {
            id := rep["id"].(string)
            return r.posts[id], nil
        },
    }
}
```

### Gateway

**Apollo Gateway:**
```javascript
const { ApolloGateway } = require('@apollo/gateway');
const { ApolloServer } = require('apollo-server');

const gateway = new ApolloGateway({
  serviceList: [
    { name: 'users', url: 'http://localhost:4001/graphql' },
    { name: 'posts', url: 'http://localhost:4002/graphql' },
  ],
});

const server = new ApolloServer({ gateway });
server.listen(4000).then(({ url }) => {
  console.log(`🚀 Gateway ready at ${url}`);
});
```

### Client Query

```graphql
query GetUserWithPosts {
  user(id: "1") {
    id
    name           # From Users service
    email          # From Users service
    posts {        # From Posts service
      id
      title
    }
  }
}
```

**What happens:**
1. Gateway receives query
2. Queries Users service for `user(id: "1")`
3. Gets `{ id: "1", name: "Alice", email: "alice@..." }`
4. Queries Posts service `_entities` with `{ __typename: "User", id: "1" }`
5. Posts service resolves User entity and returns posts
6. Gateway combines results
7. Client receives complete response

## Breaking Changes

**Only one breaking change:**

All resolvers must implement `EntityResolvers()`:

```go
// Add this to all existing resolvers
func (r *Resolvers) EntityResolvers() map[string]rocket.EntityResolveFn {
    return map[string]rocket.EntityResolveFn{}  // Empty map if no entities
}
```

## Success Criteria ✅

Federation is complete when:
- [x] Can mark types with `@key` directive
- [x] Can implement entity resolvers
- [x] `_entities` query works
- [x] `_service` query returns SDL
- [x] Works with Apollo Gateway
- [x] Works with Cosmo Router
- [x] Complete 2-subgraph example
- [x] Comprehensive documentation
- [x] All tests passing

## Benefits

### For Users
- ✅ **Microservices** - Split monolith into services
- ✅ **Team Autonomy** - Teams own subgraphs
- ✅ **Gradual Migration** - Migrate incrementally
- ✅ **Scale Independently** - Scale hot services
- ✅ **Technology Diversity** - Mix languages/frameworks

### For Rocket
- ✅ **Feature Parity** - Match Apollo Server capabilities
- ✅ **Enterprise Ready** - Support large-scale architectures
- ✅ **Go Ecosystem** - Native Go federation solution
- ✅ **Performance** - Built on `graphql-go-tools` (fastest)

## Next Steps

1. **Review this plan** - Confirm approach
2. **Start implementation** - Phase 1 (Core)
3. **Iterate** - Based on feedback
4. **Test thoroughly** - With real gateway
5. **Document** - Complete guides
6. **Release v0.5.0** - With federation support

## Timeline

```
Week 1:
  Day 1-2: Core implementation (directives, entity resolvers, queries)
  Day 3:   Developer experience (helpers, validation)
  Day 4-5: Testing and examples
  Day 6-7: Documentation and polish

Week 2:
  Release v0.5.0 with full federation support! 🚀
```

## Questions?

- How should we handle entity key extraction? (Helper function?)
- Should federation be opt-in or always available?
- Which gateway should we recommend? (Apollo vs Cosmo)
- Should we auto-detect `@key` directives or require config?
- Support Federation v1 or v2 spec?

## Resources

- [Apollo Federation Docs](https://www.apollographql.com/docs/federation/)
- [Cosmo Federation](https://cosmo-docs.wundergraph.com/)
- [Federation Spec](https://www.apollographql.com/docs/federation/federation-spec/)
- [graphql-go-tools](https://github.com/wundergraph/graphql-go-tools)

---

**Status:** Ready for implementation  
**Target Version:** v0.5.0  
**Estimated Effort:** 1 week  
**Priority:** High (frequently requested feature)

