// Package rocket provides Apollo GraphQL-like patterns for Go
// Bringing TypeScript developer experience to Golang GraphQL development
package rocket

import (
	"context"
)

// ResolveParams contains all the information for resolving a field
type ResolveParams struct {
	Source  interface{}
	Args    map[string]interface{}
	Context context.Context
	Info    ResolveInfo
}

// ResolveInfo contains metadata about the field being resolved
type ResolveInfo struct {
	FieldName      string
	Path           []string
	ParentType     string
	ReturnType     string
	SelectionSet   interface{} // Will be wundergraph AST selection set
}

// FieldResolveFn is a function that resolves a field value
type FieldResolveFn func(p ResolveParams) (interface{}, error)

// ModuleResolvers is the interface that all module resolvers must implement
// This is similar to how you export resolvers in TypeScript Apollo
type ModuleResolvers interface {
	QueryResolvers() map[string]FieldResolveFn
	MutationResolvers() map[string]FieldResolveFn
	TypeResolvers() map[string]map[string]FieldResolveFn
}

// Config holds configuration for building a GraphQL schema
type Config struct {
	SchemaPath       string // Path to the compiled schema.graphql file
	EnablePlayground bool   // Enable GraphQL playground (default: false)
}

// Config defaults
const (
	DefaultPreserveOrder = true
)

