package rocket

import (
	"github.com/graphql-go/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

// schemaBuilder builds a graphql-go schema from parsed AST and resolvers
type schemaBuilder struct {
	resolvers    *ResolverRegistry
	parsedSchema *ast.Schema
	types        map[string]graphql.Type
	inputs       map[string]graphql.Type
	enums        map[string]*graphql.Enum
}

// newSchemaBuilder creates a new schema builder
func newSchemaBuilder(resolvers *ResolverRegistry, parsedSchema *ast.Schema) *schemaBuilder {
	return &schemaBuilder{
		resolvers:    resolvers,
		parsedSchema: parsedSchema,
		types:        make(map[string]graphql.Type),
		inputs:       make(map[string]graphql.Type),
		enums:        make(map[string]*graphql.Enum),
	}
}

// build builds the executable graphql-go schema
func (b *schemaBuilder) build() (graphql.Schema, error) {
	// First pass: Build enums
	for _, def := range b.parsedSchema.Types {
		if def.BuiltIn {
			continue
		}
		if def.Kind == ast.Enum {
			b.buildEnum(def)
		}
	}

	// Second pass: Build input types
	for _, def := range b.parsedSchema.Types {
		if def.BuiltIn {
			continue
		}
		if def.Kind == ast.InputObject {
			b.buildInputType(def)
		}
	}

	// Third pass: Build output types
	for _, def := range b.parsedSchema.Types {
		if def.BuiltIn {
			continue
		}
		if def.Kind == ast.Object {
			b.buildType(def)
		}
	}

	// Build root query and mutation
	var queryType *graphql.Object
	var mutationType *graphql.Object

	if b.parsedSchema.Query != nil {
		queryType = b.buildQueryType(b.parsedSchema.Query)
	}
	if b.parsedSchema.Mutation != nil {
		mutationType = b.buildMutationType(b.parsedSchema.Mutation)
	}

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})
}

// buildEnum builds a GraphQL enum type
func (b *schemaBuilder) buildEnum(def *ast.Definition) *graphql.Enum {
	if existing, ok := b.enums[def.Name]; ok {
		return existing
	}

	values := graphql.EnumValueConfigMap{}
	for _, value := range def.EnumValues {
		values[value.Name] = &graphql.EnumValueConfig{
			Value: value.Name,
		}
	}

	enum := graphql.NewEnum(graphql.EnumConfig{
		Name:   def.Name,
		Values: values,
	})

	b.enums[def.Name] = enum
	return enum
}

// buildInputType builds a GraphQL input type
func (b *schemaBuilder) buildInputType(def *ast.Definition) graphql.Input {
	if existing, ok := b.inputs[def.Name]; ok {
		return existing
	}

	fields := graphql.InputObjectConfigFieldMap{}

	inputObj := graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   def.Name,
		Fields: fields,
	})

	b.inputs[def.Name] = inputObj

	// Build fields
	for _, field := range def.Fields {
		fields[field.Name] = &graphql.InputObjectFieldConfig{
			Type: b.convertInputType(field.Type),
		}
	}

	return inputObj
}

// buildType builds a GraphQL object type
func (b *schemaBuilder) buildType(def *ast.Definition) *graphql.Object {
	if def.Name == "Query" || def.Name == "Mutation" {
		return nil
	}

	if existing, ok := b.types[def.Name]; ok {
		if obj, ok := existing.(*graphql.Object); ok {
			return obj
		}
	}

	// Use FieldsThunk to allow lazy evaluation for circular references
	obj := graphql.NewObject(graphql.ObjectConfig{
		Name: def.Name,
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			fields := graphql.Fields{}
			for _, field := range def.Fields {
				fields[field.Name] = b.buildField(def.Name, field)
			}
			return fields
		}),
	})

	b.types[def.Name] = obj
	return obj
}

// buildField builds a GraphQL field
func (b *schemaBuilder) buildField(typeName string, field *ast.FieldDefinition) *graphql.Field {
	return &graphql.Field{
		Type:    b.convertOutputType(field.Type),
		Resolve: b.getResolver(typeName, field.Name),
	}
}

// getResolver gets the appropriate resolver for a type field
func (b *schemaBuilder) getResolver(typeName, fieldName string) graphql.FieldResolveFn {
	// Check for custom resolver
	if resolver, ok := b.resolvers.GetTypeResolver(typeName, fieldName); ok {
		return b.wrapResolver(resolver)
	}

	// Use default field resolver
	return b.wrapResolver(DefaultFieldResolver)
}

// wrapResolver wraps a Rocket FieldResolveFn into a graphql.FieldResolveFn
func (b *schemaBuilder) wrapResolver(fn FieldResolveFn) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		rocketParams := ResolveParams{
			Source:  p.Source,
			Args:    p.Args,
			Context: p.Context,
			Info: ResolveInfo{
				FieldName:  p.Info.FieldName,
				ParentType: p.Info.ParentType.Name(),
				ReturnType: p.Info.ReturnType.String(),
			},
		}
		return fn(rocketParams)
	}
}

// convertOutputType converts ast.Type to graphql.Output
func (b *schemaBuilder) convertOutputType(t *ast.Type) graphql.Output {
	if t.NonNull {
		inner := &ast.Type{
			NamedType: t.NamedType,
			Elem:      t.Elem,
		}
		return graphql.NewNonNull(b.convertOutputType(inner))
	}

	if t.Elem != nil {
		return graphql.NewList(b.convertOutputType(t.Elem))
	}

	switch t.NamedType {
	case "String":
		return graphql.String
	case "Int":
		return graphql.Int
	case "Float":
		return graphql.Float
	case "Boolean":
		return graphql.Boolean
	case "ID":
		return graphql.ID
	default:
		if enum, ok := b.enums[t.NamedType]; ok {
			return enum
		}
		if obj, ok := b.types[t.NamedType]; ok {
			return obj
		}
		return graphql.String
	}
}

// convertInputType converts ast.Type to graphql.Input
func (b *schemaBuilder) convertInputType(t *ast.Type) graphql.Input {
	if t.NonNull {
		inner := &ast.Type{
			NamedType: t.NamedType,
			Elem:      t.Elem,
		}
		return graphql.NewNonNull(b.convertInputType(inner))
	}

	if t.Elem != nil {
		return graphql.NewList(b.convertInputType(t.Elem))
	}

	switch t.NamedType {
	case "String":
		return graphql.String
	case "Int":
		return graphql.Int
	case "Float":
		return graphql.Float
	case "Boolean":
		return graphql.Boolean
	case "ID":
		return graphql.ID
	default:
		if enum, ok := b.enums[t.NamedType]; ok {
			return enum
		}
		if input, ok := b.inputs[t.NamedType]; ok {
			return input
		}
		return graphql.String
	}
}

// buildQueryType builds the root Query type
func (b *schemaBuilder) buildQueryType(queryDef *ast.Definition) *graphql.Object {
	fields := graphql.Fields{}

	for _, field := range queryDef.Fields {
		// Skip introspection fields - graphql-go adds these automatically
		// Fields starting with "__" are reserved by GraphQL for introspection
		if len(field.Name) >= 2 && field.Name[0] == '_' && field.Name[1] == '_' {
			continue
		}

		args := graphql.FieldConfigArgument{}
		for _, arg := range field.Arguments {
			args[arg.Name] = &graphql.ArgumentConfig{
				Type: b.convertInputType(arg.Type),
			}
		}

		// Get query resolver
		var resolver graphql.FieldResolveFn
		if r, ok := b.resolvers.GetQueryResolver(field.Name); ok {
			resolver = b.wrapResolver(r)
		}

		fields[field.Name] = &graphql.Field{
			Type:    b.convertOutputType(field.Type),
			Args:    args,
			Resolve: resolver,
		}
	}

	return graphql.NewObject(graphql.ObjectConfig{
		Name:   "Query",
		Fields: fields,
	})
}

// buildMutationType builds the root Mutation type
func (b *schemaBuilder) buildMutationType(mutationDef *ast.Definition) *graphql.Object {
	fields := graphql.Fields{}

	for _, field := range mutationDef.Fields {
		// Skip introspection fields - graphql-go adds these automatically
		// Fields starting with "__" are reserved by GraphQL for introspection
		if len(field.Name) >= 2 && field.Name[0] == '_' && field.Name[1] == '_' {
			continue
		}

		args := graphql.FieldConfigArgument{}
		for _, arg := range field.Arguments {
			args[arg.Name] = &graphql.ArgumentConfig{
				Type: b.convertInputType(arg.Type),
			}
		}

		// Get mutation resolver
		var resolver graphql.FieldResolveFn
		if r, ok := b.resolvers.GetMutationResolver(field.Name); ok {
			resolver = b.wrapResolver(r)
		}

		fields[field.Name] = &graphql.Field{
			Type:    b.convertOutputType(field.Type),
			Args:    args,
			Resolve: resolver,
		}
	}

	return graphql.NewObject(graphql.ObjectConfig{
		Name:   "Mutation",
		Fields: fields,
	})
}

