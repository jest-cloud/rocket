package main

import (
	"fmt"
	"testing"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

// TestParseSchema tests if we can parse a schema.graphql file
func TestParseSchema(t *testing.T) {
	schema := `
type Query {
  hello: String!
  users: [User!]!
}

type User {
  id: ID!
  name: String!
  email: String!
}
`

	doc, report := astparser.ParseGraphqlDocumentString(schema)
	if report.HasErrors() {
		for _, err := range report.InternalErrors {
			t.Fatalf("Parse error: %v", err)
		}
		if len(report.ExternalErrors) > 0 {
			for _, err := range report.ExternalErrors {
				t.Errorf("Parse error: %v", err)
			}
		}
	}

	fmt.Printf("✓ Successfully parsed schema\n")
	fmt.Printf("  Document parsed successfully\n")
	fmt.Printf("  Object type definitions: %d\n", len(doc.ObjectTypeDefinitions))
}

// TestExecuteQuery is a placeholder - we need to figure out how to execute queries
func TestExecuteQuery(t *testing.T) {
	// This is where we'll test execution once we understand the API
	t.Skip("Need to understand execution engine first")
}

