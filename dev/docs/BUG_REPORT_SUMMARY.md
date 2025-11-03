# Auto-Resolution Bug Investigation Summary

## Issue
Auto-field resolution doesn't work for list/slice elements when returning `[]User` from query resolvers.

## Attempted Fixes

1. **Recursive Interface Unwrapping**: Added logic to recursively unwrap `interface{}` types that wrap structs, as graphql-go may pass list elements wrapped in interface{}.

2. **Map Handling**: Added handling for `map[string]interface{}` types, as graphql-go may convert structs to maps when processing lists.

3. **Capitalized Key Support**: Added fallback to check for capitalized map keys (e.g., "ID" instead of "id") in case graphql-go uses struct field names.

## Current Status
The issue persists despite these improvements. The Source type being passed to field resolvers for list elements is not being handled correctly.

## Next Steps
- Investigate graphql-go source code to understand how it processes list elements
- Add debug logging to see what Source type is actually being passed
- Consider if we need to handle this at the schema builder level rather than the resolver level
- Test if returning `[]*User` instead of `[]User` makes a difference

## Workaround
Use explicit type resolvers for list types until this is resolved. See `ISSUE_TEMPLATE.md` for details.
