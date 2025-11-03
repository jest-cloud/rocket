Of course. Here is a set of instructions formatted for a Large Language Model, outlining the core recommendations for building the "Rocket" GraphQL library.

***

### LLM Task: Implement the "Rocket" Go GraphQL Library

Your task is to create a Go GraphQL library named "Rocket". The primary goal of Rocket is to provide a **schema-first** developer experience that is more flexible and requires less boilerplate than existing solutions like `gqlgen`.

The library's core innovation is its **Hybrid Resolver Model**, which combines an explicit, type-safe resolver map with a smart, reflection-based fallback mechanism. The entire engine MUST be built on top of the high-performance `wundergraph/graphql-go-tools` package.

---

### 1. The Hybrid Resolver Model

This is the central feature of Rocket. Developers should only need to write resolvers for fields that require custom logic.

#### 1.1. The Explicit Resolver Map

The developer defines their resolvers in a set of structs that mirror the GraphQL schema's structure. This provides compile-time safety.

**`schema.graphql`**
```graphql
type Query {
  viewer: User!
}

type User {
  id: ID!
  name: String!
  friends: [User!]!
}
```

**`resolvers.go` (Developer's Code)**
```go
// The developer defines these structs to hold their resolver functions.
// The field names MUST match the schema for compile-time validation.
type RootResolvers struct {
    Query QueryResolvers
    User   UserResolvers
}

type QueryResolvers struct {
    Viewer func(ctx context.Context) (*UserModel, error)
}

type UserResolvers struct {
    // NOTE: These fields are OPTIONAL. This is the key feature.
    ID      func(ctx context.Context, user *UserModel) (string, error)
    Name    func(ctx context.Context, user *UserModel) (string, error)
    Friends func(ctx context.Context, user *UserModel) ([]*UserModel, error)
}
```

#### 1.2. The Smart Fallback Mechanism

If a resolver is **not** provided in the explicit map, the Rocket engine MUST attempt to resolve the value using the following fallback order:

1.  **Check for Struct Field:** Inspect the parent Go struct for a public field with a matching name (e.g., GraphQL `name` maps to Go `Name`).
2.  **Check for Struct Method:** If no field is found, inspect the parent Go struct for a public method with a matching name (e.g., GraphQL `name` maps to Go method `Name()`).
3.  **Handle Failure:** If neither is found, proceed to the Schema-Aware Resolution logic (Section 2).

#### 1.3. Example of the Hybrid Model in Action

This design dramatically reduces boilerplate for the user.

**`models.go` (Developer's internal model)**
```go
type UserModel struct {
    ID   string // Matches schema `id` via fallback
    // `name` is missing, so it requires an explicit resolver
    FirstName string
    LastName  string
}
```

**`main.go` (Developer's implementation)**
```go
func main() {
    // --- Only the necessary resolvers are defined ---
    resolvers := RootResolvers{
        Query: QueryResolvers{
            // The root query resolver is always required.
            Viewer: func(ctx context.Context) (*UserModel, error) {
                return &UserModel{ID: "1", FirstName: "John", LastName: "Doe"}, nil
            },
        },
        User: UserResolvers{
            // `ID` is handled by the fallback.
            // `name` requires custom logic, so we provide an explicit resolver.
            Name: func(ctx context.Context, user *UserModel) (string, error) {
                return user.FirstName + " " + user.LastName, nil
            },
            // `friends` requires custom logic.
            Friends: func(ctx context.Context, user *UserModel) ([]*UserModel, error) {
                return []*UserModel{}, nil
            },
        },
    }

    server := rocket.NewServer("schema.graphql", resolvers)
    // ...
}
```

---

### 2. Schema-Aware Resolution Logic

When a field cannot be resolved (either explicitly or via fallback), the library MUST respect the nullability defined in the GraphQL schema.

-   **If the schema field is Nullable** (e.g., `bio: String`): The engine should resolve the value to `null`.
-   **If the schema field is Non-Nullable** (e.g., `name: String!`): The engine MUST return a GraphQL error for that field, which will cause its parent object to become `null` in the response, as per the specification.

**Example Error Response:**
```json
{
  "errors": [
    {
      "message": "Cannot return null for non-nullable field User.name.",
      "path": ["viewer", "name"]
    }
  ],
  "data": {
    "viewer": null
  }
}
```

---

### 3. Core Technical Foundation: `wundergraph/graphql-go-tools`

The library **MUST** be built on `wundergraph/graphql-go-tools`. Do not use other parsers or build an execution engine from scratch.

#### 3.1. The Adapter Pattern

The primary role of Rocket's runtime is to be an **adapter** between the user's simple `RootResolvers` map and the powerful `wundergraph` execution engine.

You will implement the `resolve.DataSource` interface from the `wundergraph` library. The core of your library will be an adapter that satisfies this interface.

**Conceptual Implementation:**
```go
import (
    "github.com/wundergraph/graphql-go-tools/pkg/engine/resolve"
)

// RocketDataSource is the adapter. It wraps the user's resolvers.
type RocketDataSource struct {
    ctx       context.Context
    resolvers RootResolvers
}

// Implement the DataSource interface.
// This method is the core of the runtime engine.
func (r *RocketDataSource) Load(ctx context.Context, parent []byte, sourceChan chan<- []byte) {
    // This is where you implement the Hybrid Resolver logic:
    // 1. Identify the field being requested by the wundergraph engine.
    // 2. Check for an explicit resolver in r.resolvers.
    // 3. If not found, use reflection for the fallback mechanism.
    // 4. If still not found, apply the schema-aware null/error logic.
    // 5. Write the final result bytes to the sourceChan.
}
```

---

### 4. Recommended Feature: Configurable Server Modes

To provide the best of both worlds (safety in production, flexibility in development), implement a configurable server mode.

-   **`ProductionMode` (Default):** At server startup, perform a strict validation. The server MUST panic and exit if any field in the schema does not have a valid resolution path (either an explicit resolver or a valid fallback field/method).
-   **`DevelopmentMode` (Optional):** At startup, skip the strict validation. Allow the server to run even with unresolvable fields, relying on the runtime schema-aware logic to return `null` or errors per-query.

**User API Example:**
```go
// Default, safe behavior for production.
prodServer := rocket.NewServer(schema, resolvers)

// Optional, permissive behavior for local development.
devServer := rocket.NewServer(schema, resolvers, rocket.WithMode(rocket.DevelopmentMode))
```