# Migration from graphql-go to WunderGraph graphql-go-tools

## Goals
- Keep schema-first approach (schema.graphql files)
- Maintain same resolver pattern (maps of resolvers)
- Keep auto-resolution feature
- Keep modular architecture

## Key Differences

### graphql-go
- Uses `graphql.Schema` type
- Uses `graphql.Do()` for execution
- Uses `graphql.NewObject()` for building types
- Uses `graphql.ResolveParams` for resolvers

### graphql-go-tools (v2)
- Has its own AST (`ast.Document`)
- Has execution engine in `pkg/engine`
- Has resolver system in `pkg/engine/resolve`
- Uses `astparser` for parsing

## Migration Strategy

1. Replace AST parsing - use graphql-go-tools AST instead of gqlparser
2. Replace schema building - build execution plan instead of graphql.Schema
3. Replace execution - use graphql-go-tools engine instead of graphql.Do()
4. Keep resolver interface - wrap our resolvers to work with graphql-go-tools

## Files to Update
- `schema.go` - Use graphql-go-tools AST parsing
- `schema_builder.go` - Build execution plan instead of graphql.Schema
- `execution.go` - Use graphql-go-tools execution engine
- `default_resolver.go` - Adapt to graphql-go-tools resolver interface
- `go.mod` - Replace dependencies
