package main

import (
	"fmt"
	"testing"

	"github.com/jensneuse/abstractlogger"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
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

// TestBuildExecutionPlan tests if we can build an execution plan from schema + query
func TestBuildExecutionPlan(t *testing.T) {
	schema := `
type Query {
  hello: String!
}

type User {
  id: ID!
  name: String!
}
`
	
	query := `
query {
  hello
}
`

	// Parse schema
	schemaDoc, report := astparser.ParseGraphqlDocumentString(schema)
	if report.HasErrors() {
		t.Fatalf("Failed to parse schema: %v", report.Error())
	}

	// Parse query
	queryDoc, queryReport := astparser.ParseGraphqlDocumentString(query)
	if queryReport.HasErrors() {
		t.Fatalf("Failed to parse query: %v", queryReport.Error())
	}

	fmt.Printf("✓ Schema parsed: %d object types\n", len(schemaDoc.ObjectTypeDefinitions))
	fmt.Printf("✓ Query parsed: %d operations\n", len(queryDoc.OperationDefinitions))
	
	// Create planner configuration
	// For standalone execution, we'll need to create a custom DataSource
	// For now, let's see if we can at least create a planner
	plannerConfig := plan.Configuration{
		DataSources: []plan.DataSource{}, // Empty for now - will need custom DataSource
		Fields:      plan.FieldConfigurations{},
		Logger:      abstractlogger.Noop{},
	}
	
	planner, err := plan.NewPlanner(plannerConfig)
	if err != nil {
		t.Fatalf("Failed to create planner: %v", err)
	}
	
	// Try to create a plan (this will likely fail without DataSources, but let's see what happens)
	var planReport operationreport.Report
	execPlan := planner.Plan(&queryDoc, &schemaDoc, "", &planReport)
	
	if planReport.HasErrors() {
		fmt.Printf("⚠ Planner returned errors (expected without DataSources):\n")
		for _, err := range planReport.InternalErrors {
			fmt.Printf("  - %v\n", err)
		}
		if len(planReport.ExternalErrors) > 0 {
			for _, err := range planReport.ExternalErrors {
				fmt.Printf("  - %v\n", err)
			}
		}
	} else {
		fmt.Printf("✓ Execution plan created: %T\n", execPlan)
	}
}

