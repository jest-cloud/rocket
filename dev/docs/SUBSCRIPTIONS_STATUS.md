# Subscriptions Status

## Current State: FULLY IMPLEMENTED ✅

Subscriptions are **fully implemented** in Rocket v0.3.0! Real-time GraphQL subscriptions work via WebSocket with the `graphql-ws` protocol.

## What's Implemented ✅

### 1. Subscription Types ✅
```go
// SubscriptionResolveFn returns a channel that emits values over time
type SubscriptionResolveFn func(p ResolveParams) (<-chan interface{}, error)
```

### 2. Resolver Registry ✅
```go
type ResolverRegistry struct {
    Query        map[string]FieldResolveFn
    Mutation     map[string]FieldResolveFn
    Subscription map[string]SubscriptionResolveFn  // ✅ Added
    Types        map[string]map[string]FieldResolveFn
}
```

### 3. WebSocket Transport ✅
- **graphql-ws protocol** implementation
- Connection lifecycle (init, ack, ping/pong)
- Multiple subscriptions per connection
- Proper cleanup on disconnect

### 4. Subscription Execution ✅
```go
// In schema.go
func (s *Schema) ExecuteSubscription(ctx context.Context, query string, 
    variables map[string]interface{}, operationName string) (<-chan interface{}, error)
```

### 5. Context Cancellation ✅
- Subscriptions stop when client disconnects
- Proper cleanup of resources
- No goroutine leaks

### 6. Nested Field Resolution ✅
- Resolves nested fields in subscription payloads
- Works with both scalar and object types
- Handles selection sets correctly

## Quick Start

## Installation

```bash
go get github.com/jest-cloud/rocket@v0.3.0
```

## Usage Example

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

## Testing

All subscription tests pass! ✅

```bash
cd /path/to/rocket
go test -v ./test -run TestSubscription
```

Output:
```
=== RUN   TestSubscriptionExecution
=== RUN   TestSubscriptionExecution/subscription_via_HTTP_returns_error
    ✓ Subscription over HTTP correctly returns error
=== RUN   TestSubscriptionExecution/countdown_subscription
    ✓ Countdown subscription received 4 events (3,2,1,0)
=== RUN   TestSubscriptionExecution/messageAdded_subscription
    ✓ MessageAdded subscription received 3 events with nested fields
=== RUN   TestSubscriptionExecution/context_cancellation
    ✓ Subscription stopped after context cancellation
--- PASS: TestSubscriptionExecution (1.01s)
```

## Complete Example

See the [full working example](../../examples/subscriptions/) with:
- Real-time chat
- Countdown demo
- User status updates
- Apollo Client integration guide

## References

- [graphql-go-tools subscription support](https://github.com/wundergraph/graphql-go-tools)
- [graphql-ws protocol](https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md)
- [GraphQL Subscriptions spec](https://github.com/graphql/graphql-spec/blob/main/spec/Section%206%20--%20Execution.md#subscription)

## Status Summary

| Feature | Status | Version |
|---------|--------|---------|
| Query Execution | ✅ Complete | v0.1.0+ |
| Mutation Execution | ✅ Complete | v0.2.0+ |
| Subscription Execution | ✅ Complete | v0.3.0+ |
| WebSocket Handler | ✅ Complete | v0.3.0+ |
| graphql-ws Protocol | ✅ Complete | v0.3.0+ |
| Context Cancellation | ✅ Complete | v0.3.0+ |
| Nested Field Resolution | ✅ Complete | v0.3.0+ |

**Current Capability: 100% (3 of 3 operation types)** 🎉

