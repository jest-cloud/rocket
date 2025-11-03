# Rocket Subscriptions Example

This example demonstrates how to implement GraphQL subscriptions with Rocket using WebSockets and the `graphql-ws` protocol.

## Features

- ✅ Real-time message subscriptions
- ✅ Countdown subscription (demo)
- ✅ User status change subscriptions
- ✅ WebSocket transport with `graphql-ws` protocol
- ✅ Context cancellation support
- ✅ Works with Apollo Client, Relay, and other GraphQL clients

## Running the Example

```bash
# From the rocket directory
go run examples/subscriptions/main.go
```

Then open your browser to:
- **Home:** http://localhost:8080/
- **Playground:** http://localhost:8080/playground
- **GraphQL API:** http://localhost:8080/graphql (POST)
- **GraphQL WebSocket:** ws://localhost:8080/graphql/ws

## Example Usage

### 1. Subscribe to Messages

In the playground, try this subscription:

```graphql
subscription {
  messageAdded {
    id
    text
    userId
    timestamp
    user {
      name
      email
    }
  }
}
```

Then in another tab, send a message:

```graphql
mutation {
  sendMessage(text: "Hello, world!", userId: "alice") {
    id
    text
  }
}
```

You should see the message appear in the subscription!

### 2. Countdown Subscription

```graphql
subscription {
  countdown(from: 10)
}
```

This will emit numbers from 10 down to 0, one per second.

### 3. User Status Subscription

```graphql
subscription {
  userStatus(userId: "user-123") {
    userId
    status
    timestamp
  }
}
```

This will emit status changes (online → away → busy → offline) every 3 seconds.

## Using with Apollo Client

```typescript
import { ApolloClient, InMemoryCache, split, HttpLink } from '@apollo/client';
import { GraphQLWsLink } from '@apollo/client/link/subscriptions';
import { getMainDefinition } from '@apollo/client/utilities';
import { createClient } from 'graphql-ws';

// HTTP link for queries and mutations
const httpLink = new HttpLink({
  uri: 'http://localhost:8080/graphql',
});

// WebSocket link for subscriptions
const wsLink = new GraphQLWsLink(
  createClient({
    url: 'ws://localhost:8080/graphql/ws',
  })
);

// Split based on operation type
const splitLink = split(
  ({ query }) => {
    const definition = getMainDefinition(query);
    return (
      definition.kind === 'OperationDefinition' &&
      definition.operation === 'subscription'
    );
  },
  wsLink,
  httpLink
);

const client = new ApolloClient({
  link: splitLink,
  cache: new InMemoryCache(),
});

// Use subscription
client
  .subscribe({
    query: gql`
      subscription {
        messageAdded {
          id
          text
          user {
            name
          }
        }
      }
    `,
  })
  .subscribe({
    next: (data) => console.log('New message:', data),
    error: (err) => console.error('Error:', err),
  });
```

## Implementation Details

### Resolver Pattern

Subscription resolvers return a **channel** that emits values over time:

```go
func (r *Resolvers) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
    return map[string]rocket.SubscriptionResolveFn{
        "messageAdded": func(p rocket.ResolveParams) (<-chan interface{}, error) {
            ch := make(chan interface{})
            
            go func() {
                defer close(ch)
                for {
                    select {
                    case <-p.Context.Done():
                        return // Client disconnected
                    case msg := <-globalMessageChannel:
                        ch <- msg // Send to this subscriber
                    }
                }
            }()
            
            return ch, nil
        },
    }
}
```

### WebSocket Transport

Rocket implements the `graphql-ws` protocol (https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md):

1. Client connects to `/graphql/ws`
2. Client sends `connection_init`
3. Server responds with `connection_ack`
4. Client sends `subscribe` with query
5. Server streams `next` messages with data
6. Server sends `complete` when done
7. Client can send `complete` to stop subscription

### Context Cancellation

Always check `p.Context.Done()` in your subscription resolvers to properly handle client disconnections:

```go
for {
    select {
    case <-p.Context.Done():
        // Client disconnected, clean up and return
        return
    case value := <-source:
        ch <- value
    }
}
```

## Protocol Support

- ✅ **graphql-ws** (recommended, used by Apollo Client 3+)
- ❌ **subscriptions-transport-ws** (legacy, deprecated)

## Browser Testing

You can also test subscriptions using the browser console:

```javascript
const ws = new WebSocket('ws://localhost:8080/graphql/ws');

ws.onopen = () => {
  // Send connection_init
  ws.send(JSON.stringify({ type: 'connection_init' }));
  
  // After receiving connection_ack, send subscribe
  setTimeout(() => {
    ws.send(JSON.stringify({
      id: '1',
      type: 'subscribe',
      payload: {
        query: 'subscription { countdown(from: 5) }'
      }
    }));
  }, 100);
};

ws.onmessage = (event) => {
  console.log('Received:', JSON.parse(event.data));
};
```

## Next Steps

- Add authentication to subscriptions
- Implement subscription filters
- Add subscription batching
- Implement persisted subscriptions
- Add metrics and monitoring

