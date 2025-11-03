# Bug: Auto-resolution not working for list/slice elements

## Description

When returning slices of structs (e.g., `[]User`), the auto-field resolution doesn't work for resolving fields on list elements. This requires explicit type resolvers as a workaround.

## Steps to Reproduce

1. Create a GraphQL schema with a list type:
```graphql
type Query {
  users: [User!]!
}

type User {
  id: ID!
  name: String!
  email: String!
}
```

2. Return a slice of structs:
```go
"users": func(p rocket.ResolveParams) (interface{}, error) {
    return []User{
        {ID: "1", Name: "Alice", Email: "alice@example.com"},
        {ID: "2", Name: "Bob", Email: "bob@example.com"},
    }, nil
}
```

3. Query the list:
```graphql
query {
  users {
    id
    name
    email
  }
}
```

## Expected Behavior

Fields should auto-resolve from the struct fields using reflection, similar to how it works for single objects.

## Actual Behavior

Returns error: `Cannot return null for non-nullable field User.id.`

## Workaround

Add explicit type resolvers:
```go
func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
    return map[string]map[string]rocket.FieldResolveFn{
        "User": {
            "id": func(p rocket.ResolveParams) (interface{}, error) {
                switch v := p.Source.(type) {
                case *User:
                    return v.ID, nil
                case User:
                    return v.ID, nil
                default:
                    return nil, nil
                }
            },
            // ... repeat for each field
        },
    }
}
```

## Environment

- Rocket version: v0.1.0
- Go version: 1.25+
- graphql-go version: v0.8.1

## Investigation Notes

The issue appears to be related to how graphql-go processes list elements and passes them to field resolvers. The Source type might be wrapped in interface{} or converted to maps in a way that the default resolver doesn't handle correctly.

