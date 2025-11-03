// Package rocket provides developer-friendly GraphQL patterns for Go
// Bringing a modern, DX-focused approach to GraphQL development in Golang
package rocket

import (
	"github.com/jest-cloud/rocket/internal/compiler"
	"github.com/jest-cloud/rocket/internal/registry"
	"github.com/jest-cloud/rocket/internal/types"
)

// ResolveParams contains all the information for resolving a field
// Re-exported from internal/types for public API
type ResolveParams = types.ResolveParams

// ResolveInfo contains metadata about the field being resolved
// Re-exported from internal/types for public API
type ResolveInfo = types.ResolveInfo

// FieldResolveFn is a function that resolves a field value
// Re-exported from internal/types for public API
type FieldResolveFn = types.FieldResolveFn

// SubscriptionResolveFn is a function that resolves a subscription field
// It returns a channel that emits values over time
// Re-exported from internal/types for public API
type SubscriptionResolveFn = types.SubscriptionResolveFn

// ModuleResolvers is the interface that all module resolvers must implement
// Re-exported from internal/types for public API
type ModuleResolvers = types.ModuleResolvers

// Config holds configuration for building a GraphQL schema
type Config struct {
	SchemaPath       string // Path to the compiled schema.graphql file
	EnablePlayground bool   // Enable GraphQL playground (default: false)
}

// Config defaults
const (
	DefaultPreserveOrder = true
)

// ResolverRegistry holds all resolvers stitched together
// Re-exported from internal/registry for public API
type ResolverRegistry = registry.ResolverRegistry

// NewResolverRegistry creates a new registry by merging module resolvers
// Re-exported from internal/registry for public API
var NewResolverRegistry = registry.NewResolverRegistry

// SchemaCompiler compiles multiple .graphql files into a single schema
// Re-exported from internal/compiler for public API
type SchemaCompiler = compiler.SchemaCompiler

// NewSchemaCompiler creates a new schema compiler
// Re-exported from internal/compiler for public API
var NewSchemaCompiler = compiler.NewSchemaCompiler

