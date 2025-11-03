# Subscriptions Status

## Current State: NOT IMPLEMENTED ❌

Subscriptions are **not currently implemented** in Rocket. Here's what we have and what's needed:

## What Exists

### 1. Operation Type Detection ✅
```go
// In schema.go - extractFieldsAndType()
operationType := opDef.OperationType // Can be Query, Mutation, or Subscription
```

### 2. Planner Support ✅
```go
// In internal/datasource/planner.go
if p.factory.operationType == ast.OperationTypeSubscription {
    typeName = "Subscription"
}
```

### 3. Schema Support ✅
- Can parse schemas with `type Subscription { ... }`
- graphql-go-tools has full subscription support

## What's Missing

### 1. Subscription Execution Handler ❌
Currently we only handle Query and Mutation:
```go
// In schema.go Execute()
if operationType == ast.OperationTypeMutation {
    return s.executeMutationDirectly(...)
}
// Queries go to DataSource
// Subscriptions: NOT HANDLED
```

**Need**: Add subscription detection and execution path

### 2. WebSocket/SSE Transport ❌
Subscriptions require long-lived connections:
- **WebSocket** - bidirectional, real-time
- **SSE (Server-Sent Events)** - unidirectional, simpler

**Current**: Only HTTP POST/GET handlers exist
**Need**: WebSocket or SSE handler

### 3. Subscription Resolver Pattern ❌
Current resolver registry has:
- `Query map[string]FieldResolveFn`
- `Mutation map[string]FieldResolveFn`
- `Types map[string]map[string]FieldResolveFn`

**Need**: Add `Subscription map[string]SubscriptionResolveFn`

Where `SubscriptionResolveFn` returns a channel or async iterator:
```go
type SubscriptionResolveFn func(p ResolveParams) (<-chan interface{}, error)
```

### 4. Subscription Manager ❌
Need a manager to:
- Track active subscriptions per client
- Handle subscription lifecycle (subscribe/unsubscribe)
- Broadcast events to subscribers
- Clean up on disconnect

## Implementation Plan

### Phase 1: Basic Infrastructure (4-6 hours)
1. **Add Subscription to Registry**
   ```go
   type ResolverRegistry struct {
       Query        map[string]FieldResolveFn
       Mutation     map[string]FieldResolveFn
       Subscription map[string]SubscriptionResolveFn // NEW
       Types        map[string]map[string]FieldResolveFn
   }
   ```

2. **Add Subscription Detection**
   ```go
   // In schema.go Execute()
   if operationType == ast.OperationTypeSubscription {
       return nil, fmt.Errorf("subscriptions require WebSocket transport")
   }
   ```

3. **Add WebSocket Handler**
   ```go
   // In internal/http/websocket.go
   func WebSocketHandler(schema *Schema) http.HandlerFunc {
       // Upgrade connection
       // Handle GraphQL-WS protocol
       // Execute subscriptions
   }
   ```

### Phase 2: Subscription Execution (6-8 hours)
1. **Implement graphql-ws Protocol**
   - Handle connection_init
   - Handle subscribe
   - Handle complete
   - Send next, error, complete

2. **Execute Subscription Resolvers**
   ```go
   func (s *Schema) executeSubscription(
       ctx context.Context,
       queryDoc *ast.Document,
       variables map[string]interface{},
       fieldName string,
   ) (<-chan interface{}, error) {
       // Get subscription resolver
       // Call resolver (returns channel)
       // Stream results to client
   }
   ```

3. **Connection Management**
   - Track active subscriptions
   - Handle client disconnects
   - Clean up resources

### Phase 3: Testing & Polish (4-6 hours)
1. Add subscription tests
2. Handle edge cases (reconnect, error handling)
3. Add examples
4. Documentation

**Total Estimated Effort: 14-20 hours**

## Example Usage (Future)

### Schema
```graphql
type Subscription {
  messageAdded(roomId: ID!): Message!
  userStatusChanged(userId: ID!): UserStatus!
}
```

### Resolver
```go
subscriptionResolvers := map[string]rocket.SubscriptionResolveFn{
    "messageAdded": func(p rocket.ResolveParams) (<-chan interface{}, error) {
        roomId := p.Args["roomId"].(string)
        
        // Create channel
        messageChan := make(chan interface{})
        
        // Subscribe to events
        go func() {
            for msg := range messageService.Subscribe(roomId) {
                messageChan <- msg
            }
            close(messageChan)
        }()
        
        return messageChan, nil
    },
}
```

### Client
```typescript
const subscription = client.subscribe({
  query: `
    subscription($roomId: ID!) {
      messageAdded(roomId: $roomId) {
        id
        text
        createdAt
      }
    }
  `,
  variables: { roomId: "room-1" }
})

subscription.subscribe({
  next: (data) => console.log("New message:", data),
  error: (err) => console.error("Error:", err),
  complete: () => console.log("Subscription ended")
})
```

## Protocols to Consider

### 1. graphql-ws (Recommended) ✅
- Modern GraphQL subscription protocol
- WebSocket-based
- Used by Apollo Client, Relay
- Spec: https://github.com/enisdenjo/graphql-ws

### 2. subscriptions-transport-ws (Legacy)
- Older protocol (deprecated)
- Used by older Apollo implementations
- Should not use for new implementations

### 3. SSE (Server-Sent Events)
- Simpler than WebSocket
- Unidirectional (server → client)
- HTTP-based
- Good for simple use cases

## Dependencies Needed

```go
// For WebSocket
"github.com/gorilla/websocket" // Already in fire-print, add to rocket

// For graphql-ws protocol
// Could implement ourselves or use existing library
```

## Current Workaround

For real-time features without subscriptions, use:
1. **Polling** - Client polls every N seconds
2. **Separate WebSocket** - Custom WebSocket outside GraphQL (like fire-print does)
3. **Webhooks** - Server pushes to client URL

## Priority Assessment

**Subscriptions are NOT critical for MVP** because:
- ✅ All queries work
- ✅ All mutations work
- ✅ Most apps don't need real-time subscriptions
- ✅ Can use polling or separate WebSocket as workaround

**Subscriptions ARE important for:**
- Real-time chat applications
- Live dashboards
- Collaborative editing
- Event streaming

## Decision

**Recommendation**: 
- Document as "Future Feature"
- Add to roadmap
- Implement when there's a concrete use case requiring subscriptions
- For now, polling or separate WebSocket is sufficient

## References

- [graphql-go-tools subscription support](https://github.com/wundergraph/graphql-go-tools)
- [graphql-ws protocol](https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md)
- [GraphQL Subscriptions spec](https://github.com/graphql/graphql-spec/blob/main/spec/Section%206%20--%20Execution.md#subscription)

## Status Summary

| Feature | Status | Effort |
|---------|--------|--------|
| Query Execution | ✅ Complete | Done |
| Mutation Execution | ✅ Complete | Done |
| Subscription Detection | ⚠️ Partial | Done |
| Subscription Execution | ❌ Not Implemented | 14-20 hours |
| WebSocket Handler | ❌ Not Implemented | Included above |
| graphql-ws Protocol | ❌ Not Implemented | Included above |

**Current Capability: 66% (2 of 3 operation types)**

