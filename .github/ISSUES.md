# Known Issues

## Auto-resolution not working for list/slice elements

**Status**: Investigating

**Description**: When returning slices of structs (e.g., `[]User`), auto-field resolution doesn't work for resolving fields on list elements. This requires explicit type resolvers as a workaround.

**Workaround**: Add explicit type resolvers for the fields:

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

**Investigation**: The issue appears to be related to how graphql-go processes list elements and passes them to field resolvers. The Source type might be wrapped in interface{} or converted to maps in a way that the default resolver doesn't handle correctly.

**Related**: See `ISSUE_TEMPLATE.md` for detailed reproduction steps.
