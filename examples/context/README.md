# Context Example

This example demonstrates two ways to pass context (like authenticated users) to GraphQL resolvers in Rocket.

## Patterns

### Option 1: Middleware Pattern (Recommended)
Use standard HTTP middleware to enrich the request context before it reaches Rocket.

**Pros:**
- Standard Go pattern
- Works with any router/framework
- Easy to compose multiple middlewares
- Testable independently

**Example:**
```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        userID, _ := verifyJWT(token)
        
        ctx := context.WithValue(r.Context(), UserIDKey, userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

handler := rocket.Handler(schema)
handler = AuthMiddleware(handler)
http.Handle("/graphql", handler)
```

### Option 2: Context Builder (Apollo-style)
Configure a context builder function in the schema config.

**Pros:**
- Similar to Apollo GraphQL pattern
- All context logic in one place
- No separate middleware needed

**Example:**
```go
schema, _ := rocket.BuildSchema(
    rocket.Config{
        SchemaPath: "schema.graphql",
        ContextBuilder: func(r *http.Request) context.Context {
            ctx := r.Context()
            token := r.Header.Get("Authorization")
            userID, _ := verifyJWT(token)
            return context.WithValue(ctx, UserIDKey, userID)
        },
    },
    resolvers,
)
```

## Running the Example

```bash
go run main.go
```

Then visit http://localhost:8080 to use the GraphQL playground.

## Testing Queries

### Public Query (no auth required)
```graphql
{
  hello
}
```

### Protected Queries (require authentication)

Add this header in the playground:
```json
{
  "Authorization": "Bearer YOUR_TOKEN"
}
```

Then run:
```graphql
{
  currentUser {
    id
    name
    email
  }
}
```

Or:
```graphql
{
  me {
    id
    name
    email
  }
}
```

## Switching Patterns

In `main.go`, uncomment the pattern you want to try:

```go
// Option 1: Middleware
runWithMiddleware()

// Option 2: Context Builder
// runWithContextBuilder()
```

Both produce the same result - choose based on your preference!

## See Also

- [Context Documentation](../../dev/docs/CONTEXT.md)
- [Resolvers Guide](../../dev/docs/RESOLVERS.md)

