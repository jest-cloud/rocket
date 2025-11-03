# Refactor Plan: Use Built-in Resolver Pattern

## Current Problem
We're creating a custom DataSource which is causing issues with:
- Argument passing (Input templates not evaluating)
- Complex field coordinate extraction
- Fighting against graphql-go-tools design

## Better Approach: Use LocalTypeFieldResolver

Instead of a custom DataSource, we should:

1. **Use graphql-go-tools' built-in resolver system**
   - `resolve.Context` has `SetTypeFieldResolver()`
   - This is the INTENDED way to provide custom resolvers

2. **Convert Rocket resolvers to graphql-go-tools resolver format**
   - Implement `resolve.TypeFieldResolver` interface
   - Map our `FieldResolveFn` to their expected signature

3. **Simpler execution flow**
   ```go
   resolverCtx := resolve.NewContext(ctx)
   resolverCtx.Variables = variablesJSON
   
   // Register Rocket resolvers as type field resolvers
   resolverCtx.SetTypeFieldResolver(&RocketTypeFieldResolver{
       resolvers: s.resolvers,
   })
   ```

## Benefits
- ✅ Arguments automatically extracted by graphql-go-tools
- ✅ No need for custom DataSource or Input templates
- ✅ No field coordinate extraction needed
- ✅ Simpler code, following intended patterns
- ✅ Mutations will work out of the box

## Implementation Steps
1. Remove custom DataSource (planner.go, source.go, datasource.go)
2. Create `RocketTypeFieldResolver` implementing `resolve.TypeFieldResolver`
3. Update `execution.go` to register our resolver with the context
4. Simplify schema.go (no more DataSource factory needed)

This is the "don't fight the framework" approach!

