# Rocket Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- GraphQL Playground support via `PlaygroundHandler`
- Introspection query support (`__schema`, `__type`, `__typename`)
- Auto-skipping of reserved `__` fields in schema builder

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

