# Rocket Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.4.0] - 2025-11-03

### Added
- **🎯 Apollo-style Context Builder** - `ContextBuilder` config option for per-request context building
  - Similar to Apollo GraphQL's `context` function
  - Alternative to middleware pattern
  - Available in `Config.ContextBuilder`
  - Full TypeScript/Apollo pattern compatibility
- Complete context documentation with both patterns (middleware vs Apollo-style)
- Context example with working code for both approaches
- Comprehensive resolver documentation covering all operation types

### Documentation
- Added `CONTEXT.md` - Complete guide for authentication and context patterns
- Added `RESOLVERS.md` - Complete resolver guide (queries, mutations, subscriptions, types)
- Added `QUERIES.md` - Comprehensive query patterns and best practices
- Added `MUTATIONS.md` - Complete mutation guide with transactions
- Added `FEDERATION.md` - Federation readiness and integration guide
- Updated all examples to demonstrate both context patterns
- Reorganized README documentation into clear categories
- Improved context access documentation in resolvers

## [0.3.0] - 2025-11-03

### Added
- **🚀 GraphQL Subscriptions** - Full WebSocket support with `graphql-ws` protocol
  - `SubscriptionResolveFn` type for channel-based resolvers
  - `WebSocketHandler()` for real-time subscriptions
  - `ExecuteSubscription()` method on Schema
  - Context cancellation support
  - Complete example with chat, countdown, and status updates
- Updated `ModuleResolvers` interface to include `SubscriptionResolvers()`
- Comprehensive subscription tests (4 new tests, all passing)

### Changed
- **Breaking**: `ModuleResolvers` interface now requires `SubscriptionResolvers()` method
- Migrated to `wundergraph/graphql-go-tools/v2` exclusively (removed all other GraphQL deps)
- Hybrid execution strategy: queries use DataSource pattern, mutations execute directly
- Improved parser quirk handling for scalar fields with arguments

### Fixed
- Fixed mutation execution with variables (Input template evaluation)
- Fixed nested field resolution in subscription payloads
- Fixed parser quirk where scalar fields with args appeared to have selection sets

### Documentation
- Added comprehensive subscriptions documentation
- Removed outdated migration and research docs
- Cleaned up all references to old GraphQL libraries

## [0.2.0] - 2024-11-02

### Added
- GraphQL Playground support via `PlaygroundHandler`
- Introspection query support (`__schema`, `__type`, `__typename`)
- Auto-skipping of reserved `__` fields in schema builder
- Mutation support with direct execution
- Complete migration to `wundergraph/graphql-go-tools/v2`

### Fixed
- Fixed introspection error where reserved `__` fields were being manually added
- Schema builder now properly skips introspection fields, letting graphql-go add them automatically

## [0.1.0] - 2024-10-30

### Added
- Initial Rocket release
- Developer-friendly resolver patterns for Go
- Schema-first development with `.graphql` files
- Modular architecture with `ModuleResolvers` interface
- Auto-field resolution for struct fields
- Field order preservation in query responses
- Schema compiler tool
- HTTP handler for Gin and net/http
- Default field resolver
- Support for Query, Mutation, and Type resolvers

