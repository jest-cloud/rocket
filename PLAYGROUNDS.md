# GraphQL Playgrounds in Rocket

Rocket supports multiple GraphQL playground interfaces. Choose the one that fits your needs.

## Apollo Sandbox (Default - Recommended) ⭐

**Apollo Sandbox** is the official playground from Apollo GraphQL. It's modern, feature-rich, and provides the best developer experience.

### Usage

```go
// Default handler (uses Apollo Sandbox)
http.HandleFunc("/graphql", rocket.PlaygroundHandler("/graphql"))
```

### Features

- ✨ Beautiful modern UI
- 🔍 Intelligent autocomplete with schema-aware suggestions
- 📚 Rich documentation explorer with search
- 💾 Query history and saved operations
- 🔐 Header management for authentication
- 📊 Response time metrics
- 🎯 Real-time query linting and validation
- ☁️ Cloud sync (optional - requires Apollo account)
- 🎨 Dark/light theme support

### Why Apollo Sandbox?

- **Most stable** - Actively maintained by Apollo
- **Best autocomplete** - Rarely gets stuck
- **Modern UX** - Clean, intuitive interface
- **Production-ready** - Used by thousands of companies

## GraphiQL (Alternative)

**GraphiQL** is the reference implementation from the GraphQL Foundation. It's stable, lightweight, and battle-tested.

### Usage

```go
http.HandleFunc("/graphql", rocket.PlaygroundHandlerWithType(
    "/graphql", 
    rocket.PlaygroundTypeGraphiQL,
))
```

### Features

- 🎯 Simple, focused interface
- 📖 Good autocomplete
- 📚 Schema documentation
- 💡 Query history
- 🪶 Lightweight

## GraphQL Playground (Legacy)

**GraphQL Playground** is the older playground by Prisma. It's feature-rich but can be buggy with autocomplete.

### Usage

```go
http.HandleFunc("/graphql", rocket.PlaygroundHandlerWithType(
    "/graphql", 
    rocket.PlaygroundTypePlayground,
))
```

### Features

- 🎨 Dark theme
- 📚 Documentation explorer
- 💾 Query history
- ⚙️ Settings panel

### ⚠️ Known Issues

- Can get stuck on autocomplete
- Less actively maintained
- Slower introspection

## Choosing a Playground

| Feature | Apollo Sandbox | GraphiQL | Playground |
|---------|---------------|----------|------------|
| Stability | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Autocomplete | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| UI/UX | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| Features | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| Performance | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Maintenance | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |

**Recommendation**: Use **Apollo Sandbox** (default) for the best experience.

## Example: Gin Router

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/jest-cloud/rocket"
)

router := gin.Default()

// GraphQL API endpoint
router.POST("/graphql", gin.WrapH(rocket.Handler(schema)))

// Playground endpoint (GET)
router.GET("/graphql", gin.WrapH(rocket.PlaygroundHandler("/graphql")))

// Or specify a type
router.GET("/playground", gin.WrapH(
    rocket.PlaygroundHandlerWithType("/graphql", rocket.PlaygroundTypeApolloSandbox),
))
```

## Production Considerations

### Disable in Production

You may want to disable playgrounds in production:

```go
if os.Getenv("ENV") != "production" {
    router.GET("/graphql", gin.WrapH(rocket.PlaygroundHandler("/graphql")))
}
```

### Separate Endpoint

Use a separate endpoint for the playground:

```go
// API endpoint
router.POST("/graphql", gin.WrapH(rocket.Handler(schema)))

// Playground on separate path
router.GET("/playground", gin.WrapH(rocket.PlaygroundHandler("/graphql")))
```

## CDN and Security

All playgrounds load from CDNs:
- **Apollo Sandbox**: `embeddable-sandbox.cdn.apollographql.com`
- **GraphiQL**: `unpkg.com`
- **GraphQL Playground**: `cdn.jsdelivr.net`

If you have CSP (Content Security Policy) restrictions, you may need to whitelist these domains.

## Offline Support

For environments without internet access, you'll need to host the playground assets yourself. Contact the Rocket team for self-hosted playground bundles.

