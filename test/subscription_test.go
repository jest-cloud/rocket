package rocket_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jest-cloud/rocket"
)

// testResolverWithSubscriptions implements rocket.ModuleResolvers with subscription support
type testResolverWithSubscriptions struct {
	queries       map[string]rocket.FieldResolveFn
	subscriptions map[string]rocket.SubscriptionResolveFn
}

func (r *testResolverWithSubscriptions) QueryResolvers() map[string]rocket.FieldResolveFn {
	if r.queries == nil {
		return map[string]rocket.FieldResolveFn{}
	}
	return r.queries
}

func (r *testResolverWithSubscriptions) MutationResolvers() map[string]rocket.FieldResolveFn {
	return map[string]rocket.FieldResolveFn{}
}

func (r *testResolverWithSubscriptions) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
	if r.subscriptions == nil {
		return map[string]rocket.SubscriptionResolveFn{}
	}
	return r.subscriptions
}

func (r *testResolverWithSubscriptions) EntityResolvers() map[string]rocket.EntityResolveFn {
	return map[string]rocket.EntityResolveFn{}
}

func (r *testResolverWithSubscriptions) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
	return map[string]map[string]rocket.FieldResolveFn{}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSubscriptionExecution tests subscription execution
func TestSubscriptionExecution(t *testing.T) {
	// Create a temporary schema file
	schemaContent := `
type Query {
  hello: String!
}

type Subscription {
  countdown(from: Int!): Int!
  messageAdded: Message!
}

type Message {
  id: ID!
  text: String!
  timestamp: String!
}
`

	tmpfile, err := os.CreateTemp("", "test-schema-*.graphql")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(schemaContent); err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Define test types
	type Message struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		Timestamp string `json:"timestamp"`
	}

	// Create resolver implementations
	queryResolvers := map[string]rocket.FieldResolveFn{
		"hello": func(p rocket.ResolveParams) (interface{}, error) {
			return "Hello, World!", nil
		},
	}

	subscriptionResolvers := map[string]rocket.SubscriptionResolveFn{
		"countdown": func(p rocket.ResolveParams) (<-chan interface{}, error) {
			from, ok := p.Args["from"].(int)
			if !ok {
				// Handle JSON numbers which come as float64
				if f, ok := p.Args["from"].(float64); ok {
					from = int(f)
				} else {
					from = 5 // default
				}
			}

			ch := make(chan interface{})
			go func() {
				defer close(ch)
				for i := from; i >= 0; i-- {
					select {
					case <-p.Context.Done():
						return
					case ch <- i:
						time.Sleep(100 * time.Millisecond) // Small delay for testing
					}
				}
			}()
			return ch, nil
		},
		"messageAdded": func(p rocket.ResolveParams) (<-chan interface{}, error) {
			ch := make(chan interface{})
			go func() {
				defer close(ch)
				messages := []Message{
					{ID: "1", Text: "Hello", Timestamp: "2023-01-01T00:00:00Z"},
					{ID: "2", Text: "World", Timestamp: "2023-01-01T00:00:01Z"},
					{ID: "3", Text: "!", Timestamp: "2023-01-01T00:00:02Z"},
				}
				for _, msg := range messages {
					select {
					case <-p.Context.Done():
						return
					case ch <- msg:
						time.Sleep(100 * time.Millisecond)
					}
				}
			}()
			return ch, nil
		},
	}

	// Create a resolver that implements rocket.ModuleResolvers
	resolver := &testResolverWithSubscriptions{
		queries:       queryResolvers,
		subscriptions: subscriptionResolvers,
	}

	// Build schema
	schema, err := rocket.BuildSchema(rocket.Config{
		SchemaPath: tmpfile.Name(),
	}, resolver)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	// Test 1: Subscription via HTTP should return error
	t.Run("subscription via HTTP returns error", func(t *testing.T) {
		query := `subscription { countdown(from: 3) }`
		result := schema.Execute(context.Background(), query, nil, "")

		if result.Errors == nil || len(result.Errors) == 0 {
			t.Error("Expected error when executing subscription over HTTP, got none")
		}

		if len(result.Errors) > 0 {
			expectedMsg := "Subscriptions must be executed over WebSocket"
			if !containsString(result.Errors[0].Message, expectedMsg) {
				t.Errorf("Expected error message containing '%s', got '%s'", expectedMsg, result.Errors[0].Message)
			}
			t.Logf("✓ Subscription over HTTP correctly returns error: %s", result.Errors[0].Message)
		}
	})

	// Test 2: ExecuteSubscription with countdown
	t.Run("countdown subscription", func(t *testing.T) {
		query := `subscription { countdown(from: 3) }`
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		resultChan, err := schema.ExecuteSubscription(ctx, query, map[string]interface{}{"from": 3}, "")
		if err != nil {
			t.Fatalf("Failed to execute subscription: %v", err)
		}

		var results []interface{}
		for result := range resultChan {
			results = append(results, result)
			resultBytes, _ := json.Marshal(result)
			t.Logf("Received: %s", string(resultBytes))
		}

		if len(results) != 4 { // 3, 2, 1, 0
			t.Errorf("Expected 4 results (countdown from 3), got %d", len(results))
		}

		t.Logf("✓ Countdown subscription received %d events", len(results))
	})

	// Test 3: ExecuteSubscription with messageAdded
	t.Run("messageAdded subscription", func(t *testing.T) {
		query := `subscription { messageAdded { id text timestamp } }`
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		resultChan, err := schema.ExecuteSubscription(ctx, query, nil, "")
		if err != nil {
			t.Fatalf("Failed to execute subscription: %v", err)
		}

		var results []interface{}
		for result := range resultChan {
			results = append(results, result)
			resultBytes, _ := json.Marshal(result)
			t.Logf("Received: %s", string(resultBytes))

			// Verify structure
			if resultMap, ok := result.(map[string]interface{}); ok {
				if messageAdded, ok := resultMap["messageAdded"]; ok {
					if msgMap, ok := messageAdded.(map[string]interface{}); ok {
						if _, hasID := msgMap["id"]; !hasID {
							t.Error("Message missing 'id' field")
						}
						if _, hasText := msgMap["text"]; !hasText {
							t.Error("Message missing 'text' field")
						}
						if _, hasTimestamp := msgMap["timestamp"]; !hasTimestamp {
							t.Error("Message missing 'timestamp' field")
						}
					} else {
						t.Error("messageAdded is not a map")
					}
				} else {
					t.Error("Result missing 'messageAdded' field")
				}
			}
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(results))
		}

		t.Logf("✓ MessageAdded subscription received %d events", len(results))
	})

	// Test 4: Context cancellation stops subscription
	t.Run("context cancellation", func(t *testing.T) {
		query := `subscription { countdown(from: 10) }`
		ctx, cancel := context.WithCancel(context.Background())

		resultChan, err := schema.ExecuteSubscription(ctx, query, map[string]interface{}{"from": 10}, "")
		if err != nil {
			t.Fatalf("Failed to execute subscription: %v", err)
		}

		// Receive a few results, then cancel
		count := 0
		for result := range resultChan {
			count++
			resultBytes, _ := json.Marshal(result)
			t.Logf("Received %d: %s", count, string(resultBytes))

			if count == 3 {
				cancel() // Cancel after 3 events
				break
			}
		}

		// Drain remaining events (if any)
		for range resultChan {
			count++
		}

		if count > 5 {
			t.Errorf("Expected subscription to stop after cancellation, but got %d events", count)
		}

		t.Logf("✓ Subscription stopped after context cancellation (received %d events)", count)
	})
}

