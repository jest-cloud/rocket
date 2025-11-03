package types

import (
	"context"
)

// FieldResolveFn is a function that resolves a field value
type FieldResolveFn func(p ResolveParams) (interface{}, error)

// SubscriptionResolveFn is a function that resolves a subscription field
// It returns a channel that emits values over time
type SubscriptionResolveFn func(p ResolveParams) (<-chan interface{}, error)

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

// ModuleResolvers is the interface that all module resolvers must implement
// This provides a clean, modular approach to organizing resolvers
type ModuleResolvers interface {
	QueryResolvers() map[string]FieldResolveFn
	MutationResolvers() map[string]FieldResolveFn
	SubscriptionResolvers() map[string]SubscriptionResolveFn
	TypeResolvers() map[string]map[string]FieldResolveFn
}

