package federation

// FederationDirectives contains all Apollo Federation directive definitions
const FederationDirectives = `
# Federation directives (Apollo Federation v1 & v2)
directive @key(fields: _FieldSet!, resolvable: Boolean = true) repeatable on OBJECT | INTERFACE
directive @extends on OBJECT | INTERFACE
directive @external on OBJECT | FIELD_DEFINITION
directive @requires(fields: _FieldSet!) on FIELD_DEFINITION
directive @provides(fields: _FieldSet!) on FIELD_DEFINITION
directive @shareable on OBJECT | FIELD_DEFINITION
directive @inaccessible on FIELD_DEFINITION | OBJECT | INTERFACE | UNION | ARGUMENT_DEFINITION | SCALAR | ENUM | ENUM_VALUE | INPUT_OBJECT | INPUT_FIELD_DEFINITION
directive @tag(name: String!) repeatable on FIELD_DEFINITION | INTERFACE | OBJECT | UNION | ARGUMENT_DEFINITION | SCALAR | ENUM | ENUM_VALUE | INPUT_OBJECT | INPUT_FIELD_DEFINITION
directive @override(from: String!) on FIELD_DEFINITION

# Federation scalar
scalar _FieldSet
`

// FederationTypes contains the special types required by federation
const FederationTypes = `
# Federation query types
scalar _Any

type _Service {
  sdl: String!
}
`

// GetFederationSchema returns the complete federation schema additions
func GetFederationSchema() string {
	return FederationDirectives + "\n" + FederationTypes
}

