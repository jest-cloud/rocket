# Context

## Overview

Context in Rocket works similar to **Apollo GraphQL's context pattern**. It allows you to pass request-specific data (like authenticated users, database connections, etc.) to all your resolvers.

## How It Works

### Current Implementation

Rocket currently passes the HTTP request context directly to resolvers:

```go
// In handler
result := schema.Execute(r.Context(), req.Query, req.Variables, req.OperationName)

// In resolver
func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "currentUser": func(p rocket.ResolveParams) (interface{}, error) {
            userID := p.Context.Value("userID").(string)
            return r.service.GetUser(p.Context, userID)
        },
    }
}
```

### Apollo GraphQL Pattern (TypeScript)

For comparison, here's how Apollo does it:

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  context: ({ req }) => {
    // Build context from request
    const user = getUserFromToken(req.headers.authorization);
    return { user, db, dataSources };
  },
});

// In resolver
const resolvers = {
  Query: {
    currentUser: (parent, args, context) => {
      return context.user; // Access user from context
    },
  },
};
```

## Rocket Pattern (Recommended)

### Option 1: Middleware (Current Best Practice)

Use HTTP middleware to add data to context before it reaches Rocket:

```go
package main

import (
    "context"
    "net/http"
    "strings"
    "github.com/jest-cloud/rocket"
)

// AuthMiddleware extracts user from JWT and adds to context
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from header
        token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
        
        // Verify and decode token
        userID, err := verifyJWT(token)
        if err != nil {
            // Optional: fail here or continue without user
            next.ServeHTTP(w, r)
            return
        }
        
        // Add user to context
        ctx := context.WithValue(r.Context(), "userID", userID)
        ctx = context.WithValue(ctx, "token", token)
        
        // Continue with enriched context
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func main() {
    schema, _ := rocket.BuildSchema(
        rocket.Config{SchemaPath: "schema.graphql"},
        resolvers,
    )
    
    // Wrap handler with middleware
    handler := AuthMiddleware(rocket.Handler(schema))
    
    http.Handle("/graphql", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Option 2: Context Builder (Apollo-style)

✅ **Available in v0.3.0+**

Configure a context builder function that runs per-request:

```go
schema, _ := rocket.BuildSchema(
    rocket.Config{
        SchemaPath: "schema.graphql",
        ContextBuilder: func(r *http.Request) context.Context {
            // Build context from request
            token := r.Header.Get("Authorization")
            userID, _ := verifyJWT(token)
            
            ctx := r.Context()
            ctx = context.WithValue(ctx, "userID", userID)
            ctx = context.WithValue(ctx, "db", db)
            return ctx
        },
    },
    resolvers,
)

// No middleware needed - use handler directly
http.Handle("/graphql", rocket.Handler(schema))
```

**Pros:**
- Similar to Apollo GraphQL pattern
- All context logic in one place
- No separate middleware needed
- Clean and concise

**When to use:** When you prefer Apollo-style patterns or want all context logic centralized.

## Complete Example

### Main Setup with Middleware

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "github.com/jest-cloud/rocket"
)

// Custom context keys (prevents collisions)
type contextKey string

const (
    UserIDKey contextKey = "userID"
    UserKey   contextKey = "user"
    DBKey     contextKey = "db"
)

// AuthMiddleware adds user info to context
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractToken(r)
        
        ctx := r.Context()
        
        if token != "" {
            // Verify token
            userID, err := verifyJWT(token)
            if err == nil {
                // Add user ID to context
                ctx = context.WithValue(ctx, UserIDKey, userID)
                
                // Optionally fetch and add full user
                user, err := getUserByID(userID)
                if err == nil {
                    ctx = context.WithValue(ctx, UserKey, user)
                }
            }
        }
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// DBMiddleware adds database connection to context
func DBMiddleware(db *sql.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := context.WithValue(r.Context(), DBKey, db)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func extractToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    return strings.TrimPrefix(auth, "Bearer ")
}

func main() {
    db := initDB()
    
    schema, _ := rocket.BuildSchema(
        rocket.Config{SchemaPath: "schema.graphql"},
        &Resolvers{},
    )
    
    // Chain middlewares
    handler := rocket.Handler(schema)
    handler = AuthMiddleware(handler)
    handler = DBMiddleware(db)(handler)
    
    http.Handle("/graphql", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Resolvers Using Context

```go
package main

import "github.com/jest-cloud/rocket"

type Resolvers struct{}

func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        // Public query - no auth required
        "hello": func(p rocket.ResolveParams) (interface{}, error) {
            return "Hello, World!", nil
        },
        
        // Protected query - requires auth
        "currentUser": func(p rocket.ResolveParams) (interface{}, error) {
            // Get user from context
            user, ok := p.Context.Value(UserKey).(*User)
            if !ok {
                return nil, fmt.Errorf("unauthorized: user not authenticated")
            }
            return user, nil
        },
        
        // Query using user ID
        "myPosts": func(p rocket.ResolveParams) (interface{}, error) {
            userID, ok := p.Context.Value(UserIDKey).(string)
            if !ok {
                return nil, fmt.Errorf("unauthorized")
            }
            
            // Use database from context
            db := p.Context.Value(DBKey).(*sql.DB)
            
            return getPostsByUser(db, userID)
        },
    }
}

func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
    return map[string]rocket.FieldResolveFn{
        "createPost": func(p rocket.ResolveParams) (interface{}, error) {
            // Require authentication
            userID, ok := p.Context.Value(UserIDKey).(string)
            if !ok {
                return nil, fmt.Errorf("unauthorized: must be logged in")
            }
            
            input := p.Args["input"].(map[string]interface{})
            
            // User is automatically the author
            post := &Post{
                Title:    input["title"].(string),
                Content:  input["content"].(string),
                AuthorID: userID,
            }
            
            db := p.Context.Value(DBKey).(*sql.DB)
            return createPost(db, post)
        },
    }
}

func (r *Resolvers) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
    return map[string]rocket.SubscriptionResolveFn{}
}

func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{}
}
```

## Context Best Practices

### 1. Use Type-Safe Keys

Define custom types for context keys to avoid collisions:

```go
type contextKey string

const (
    UserIDKey contextKey = "userID"
    UserKey   contextKey = "user"
)

// Good - type safe
ctx = context.WithValue(ctx, UserIDKey, userID)

// Bad - string keys can collide
ctx = context.WithValue(ctx, "user", user)
```

### 2. Check Context Values Safely

Always check if context value exists:

```go
// Good - safe
userID, ok := p.Context.Value(UserIDKey).(string)
if !ok {
    return nil, fmt.Errorf("unauthorized")
}

// Bad - will panic if not set
userID := p.Context.Value(UserIDKey).(string)
```

### 3. Fail Early for Required Auth

```go
"createPost": func(p rocket.ResolveParams) (interface{}, error) {
    // Check auth FIRST
    userID, ok := p.Context.Value(UserIDKey).(string)
    if !ok {
        return nil, fmt.Errorf("unauthorized")
    }
    
    // Then continue with business logic
    input := p.Args["input"].(map[string]interface{})
    // ...
}
```

### 4. Don't Store Too Much in Context

```go
// Good - store references
ctx = context.WithValue(ctx, DBKey, db)
ctx = context.WithValue(ctx, UserIDKey, userID)

// Bad - storing large objects
ctx = context.WithValue(ctx, "allUsers", getAllUsers()) // Don't do this
```

## Common Context Patterns

### Authentication

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        
        if token != "" {
            userID, err := verifyJWT(token)
            if err == nil {
                ctx := context.WithValue(r.Context(), UserIDKey, userID)
                r = r.WithContext(ctx)
            }
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### Request Tracing

```go
func TracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        traceID := generateTraceID()
        ctx := context.WithValue(r.Context(), "traceID", traceID)
        
        // Add to response headers
        w.Header().Set("X-Trace-ID", traceID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Rate Limiting

```go
func RateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := r.Header.Get("X-User-ID")
        
        if !checkRateLimit(userID) {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        
        ctx := context.WithValue(r.Context(), "rateLimitRemaining", getRemainingQuota(userID))
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Database Transactions

```go
func TransactionMiddleware(db *sql.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Only use transactions for mutations
            if strings.Contains(r.Header.Get("Content-Type"), "mutation") {
                tx, _ := db.BeginTx(r.Context(), nil)
                defer tx.Rollback()
                
                ctx := context.WithValue(r.Context(), "tx", tx)
                next.ServeHTTP(w, r.WithContext(ctx))
                
                tx.Commit()
            } else {
                next.ServeHTTP(w, r)
            }
        })
    }
}
```

## Gin Framework Integration

Using Rocket with Gin:

```go
package main

import (
    "context"
    "github.com/gin-gonic/gin"
    "github.com/jest-cloud/rocket"
)

func main() {
    r := gin.Default()
    
    schema, _ := rocket.BuildSchema(
        rocket.Config{SchemaPath: "schema.graphql"},
        resolvers,
    )
    
    // Auth middleware
    r.Use(func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        
        if token != "" {
            userID, _ := verifyJWT(token)
            // Store in Gin context
            c.Set("userID", userID)
        }
        
        c.Next()
    })
    
    // GraphQL handler
    r.POST("/graphql", func(c *gin.Context) {
        // Build context from Gin
        ctx := c.Request.Context()
        
        // Transfer data from Gin context to Go context
        if userID, exists := c.Get("userID"); exists {
            ctx = context.WithValue(ctx, UserIDKey, userID)
        }
        
        // Parse GraphQL request
        var req struct {
            Query         string                 `json:"query"`
            Variables     map[string]interface{} `json:"variables"`
            OperationName string                 `json:"operationName"`
        }
        c.BindJSON(&req)
        
        // Execute with enriched context
        result := schema.Execute(ctx, req.Query, req.Variables, req.OperationName)
        
        c.JSON(200, result)
    })
    
    r.Run(":8080")
}
```

## Testing with Context

### Unit Test

```go
func TestResolverWithAuth(t *testing.T) {
    resolver := &Resolvers{}
    
    // Create context with user
    ctx := context.Background()
    ctx = context.WithValue(ctx, UserIDKey, "user-123")
    
    params := rocket.ResolveParams{
        Context: ctx,
        Args:    map[string]interface{}{},
    }
    
    result, err := resolver.QueryResolvers()["currentUser"](params)
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Integration Test

```go
func TestAuthenticatedQuery(t *testing.T) {
    schema, _ := rocket.BuildSchema(
        rocket.Config{SchemaPath: "schema.graphql"},
        &Resolvers{},
    )
    
    // Create authenticated context
    ctx := context.Background()
    ctx = context.WithValue(ctx, UserIDKey, "user-123")
    
    query := `{ currentUser { id name } }`
    result := schema.Execute(ctx, query, nil, "")
    
    assert.Empty(t, result.Errors)
    assert.NotNil(t, result.Data)
}
```

## Complete Example: Context Builder

✅ **Available in v0.3.0+**

```go
package main

import (
    "context"
    "net/http"
    "github.com/jest-cloud/rocket"
)

func main() {
    schema, _ := rocket.BuildSchema(
        rocket.Config{
            SchemaPath: "schema.graphql",
            // Apollo-style context builder
            ContextBuilder: func(r *http.Request) context.Context {
                ctx := r.Context()
                
                // Extract user
                token := r.Header.Get("Authorization")
                if userID, err := verifyJWT(token); err == nil {
                    ctx = context.WithValue(ctx, UserIDKey, userID)
                    
                    // Optionally fetch full user
                    if user, err := getUserByID(userID); err == nil {
                        ctx = context.WithValue(ctx, UserKey, user)
                    }
                }
                
                // Add services
                ctx = context.WithValue(ctx, DBKey, db)
                ctx = context.WithValue(ctx, "dataSources", dataSources)
                
                return ctx
            },
        },
        resolvers,
    )
    
    // No middleware needed - context builder handles everything
    http.Handle("/graphql", rocket.Handler(schema))
    http.ListenAndServe(":8080", nil)
}
```

## Comparison: Apollo vs Rocket

### Apollo (TypeScript)
```typescript
const server = new ApolloServer({
  context: ({ req }) => ({
    user: getUserFromToken(req.headers.authorization),
    db: database,
  }),
  resolvers: {
    Query: {
      me: (_, __, context) => context.user,
    },
  },
});
```

### Rocket (Current)
```go
// Middleware approach
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := getUserFromToken(r.Header.Get("Authorization"))
        ctx := context.WithValue(r.Context(), UserKey, user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Resolver
"me": func(p rocket.ResolveParams) (interface{}, error) {
    return p.Context.Value(UserKey), nil
}
```

Both achieve the same goal, just with different patterns suited to their languages.

## Related Documentation

- **[Resolvers](./RESOLVERS.md)** - Using context in resolvers
- **[Queries](./QUERIES.md)** - Query patterns with context
- **[Mutations](./MUTATIONS.md)** - Mutation auth with context

## Summary

Rocket supports **two patterns** for passing context to resolvers (both available in v0.3.0+):

### Middleware Pattern
- Standard Go HTTP middleware
- Flexible and composable
- Works with any router
- **Recommended for most use cases**

```go
handler := rocket.Handler(schema)
handler = AuthMiddleware(handler)
http.Handle("/graphql", handler)
```

### Context Builder Pattern
- Apollo GraphQL style
- All context logic in one place
- No separate middleware
- **Great for Apollo developers**

```go
schema, _ := rocket.BuildSchema(
    rocket.Config{
        ContextBuilder: func(r *http.Request) context.Context {
            // Build context here
        },
    },
    resolvers,
)
```

**Both achieve the same result** - choose based on your preference and architecture!

See the [complete example](../../examples/context/) for working code.

