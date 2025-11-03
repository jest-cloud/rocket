package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jest-cloud/rocket"
)

// Message represents a chat message
type Message struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	UserID    string `json:"userId"`
	Timestamp string `json:"timestamp"`
}

// User represents a user
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// In-memory message storage for the example
var (
	messages      []Message
	messageID     = 1
	messageChan   = make(chan Message, 10)
	statusChanges = make(chan map[string]interface{}, 10)
)

// Resolvers implements rocket.ModuleResolvers
type Resolvers struct{}

func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
	return map[string]rocket.FieldResolveFn{
		"messages": func(p rocket.ResolveParams) (interface{}, error) {
			return messages, nil
		},
		"hello": func(p rocket.ResolveParams) (interface{}, error) {
			return "Hello from Rocket Subscriptions! 🚀", nil
		},
	}
}

func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
	return map[string]rocket.FieldResolveFn{
		"sendMessage": func(p rocket.ResolveParams) (interface{}, error) {
			text := p.Args["text"].(string)
			userID := p.Args["userId"].(string)

			msg := Message{
				ID:        fmt.Sprintf("msg-%d", messageID),
				Text:      text,
				UserID:    userID,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			messageID++

			messages = append(messages, msg)

			// Broadcast to subscribers
			select {
			case messageChan <- msg:
			default:
				// Channel full, skip
			}

			return msg, nil
		},
	}
}

func (r *Resolvers) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
	return map[string]rocket.SubscriptionResolveFn{
		"messageAdded": func(p rocket.ResolveParams) (<-chan interface{}, error) {
			// Create a channel for this subscription
			ch := make(chan interface{})

			// Forward messages from the global channel to this subscription
			go func() {
				defer close(ch)

				// Create a local buffer to avoid missing messages
				for {
					select {
					case <-p.Context.Done():
						log.Println("messageAdded subscription: client disconnected")
						return
					case msg := <-messageChan:
						// Try to send, but don't block
						select {
						case ch <- msg:
						case <-p.Context.Done():
							return
						case <-time.After(1 * time.Second):
							log.Println("messageAdded subscription: timeout sending message")
						}
					}
				}
			}()

			log.Println("messageAdded subscription: new subscriber")
			return ch, nil
		},
		"countdown": func(p rocket.ResolveParams) (<-chan interface{}, error) {
			// Get 'from' argument (defaults to 10)
			from := 10
			if f, ok := p.Args["from"].(float64); ok {
				from = int(f)
			} else if i, ok := p.Args["from"].(int); ok {
				from = i
			}

			ch := make(chan interface{})

			go func() {
				defer close(ch)
				log.Printf("countdown subscription: starting from %d", from)

				for i := from; i >= 0; i-- {
					select {
					case <-p.Context.Done():
						log.Println("countdown subscription: cancelled")
						return
					case ch <- i:
						time.Sleep(1 * time.Second)
					}
				}

				log.Println("countdown subscription: completed")
			}()

			return ch, nil
		},
		"userStatus": func(p rocket.ResolveParams) (<-chan interface{}, error) {
			userID := ""
			if id, ok := p.Args["userId"].(string); ok {
				userID = id
			}

			ch := make(chan interface{})

			go func() {
				defer close(ch)
				log.Printf("userStatus subscription: watching user %s", userID)

				// Simulate status changes
				statuses := []string{"online", "away", "busy", "offline"}
				for i, status := range statuses {
					select {
					case <-p.Context.Done():
						log.Println("userStatus subscription: cancelled")
						return
					default:
						statusUpdate := map[string]interface{}{
							"userId":    userID,
							"status":    status,
							"timestamp": time.Now().Format(time.RFC3339),
						}
						ch <- statusUpdate
						if i < len(statuses)-1 {
							time.Sleep(3 * time.Second)
						}
					}
				}

				log.Println("userStatus subscription: completed")
			}()

			return ch, nil
		},
	}
}

func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
	return map[string]map[string]rocket.FieldResolveFn{
		"Message": {
			"user": func(p rocket.ResolveParams) (interface{}, error) {
				// Resolve user from message.userId
				msg, ok := p.Source.(Message)
				if !ok {
					if msgMap, ok := p.Source.(map[string]interface{}); ok {
						userID, _ := msgMap["userId"].(string)
						return User{
							ID:    userID,
							Name:  fmt.Sprintf("User %s", userID),
							Email: fmt.Sprintf("%s@example.com", userID),
						}, nil
					}
				}
				return User{
					ID:    msg.UserID,
					Name:  fmt.Sprintf("User %s", msg.UserID),
					Email: fmt.Sprintf("%s@example.com", msg.UserID),
				}, nil
			},
		},
	}
}

func main() {
	// Build schema
	schema, err := rocket.BuildSchema(rocket.Config{
		SchemaPath: "examples/subscriptions/schema.graphql",
	}, &Resolvers{})
	if err != nil {
		log.Fatalf("Failed to build schema: %v", err)
	}

	// Setup HTTP handlers
	http.HandleFunc("/graphql", rocket.Handler(schema))
	http.HandleFunc("/graphql/ws", rocket.WebSocketHandler(schema))
	http.HandleFunc("/playground", rocket.PlaygroundHandler("/graphql"))

	// Add a simple home page with instructions
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
	<title>Rocket Subscriptions Example</title>
	<style>
		body { font-family: system-ui, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
		h1 { color: #e535ab; }
		code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
		pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
		.endpoint { margin: 20px 0; }
	</style>
</head>
<body>
	<h1>🚀 Rocket Subscriptions Example</h1>
	<p>This example demonstrates GraphQL subscriptions with Rocket.</p>

	<div class="endpoint">
		<h2>Endpoints</h2>
		<ul>
			<li><strong>GraphQL API:</strong> <a href="/graphql">/graphql</a> (HTTP POST for queries/mutations)</li>
			<li><strong>GraphQL WebSocket:</strong> <code>ws://localhost:8080/graphql/ws</code> (for subscriptions)</li>
			<li><strong>Playground:</strong> <a href="/playground">/playground</a></li>
		</ul>
	</div>

	<div class="endpoint">
		<h2>Example Queries</h2>
		
		<h3>Query Messages</h3>
		<pre>query {
  messages {
    id
    text
    userId
    timestamp
  }
}</pre>

		<h3>Send Message (Mutation)</h3>
		<pre>mutation {
  sendMessage(text: "Hello!", userId: "user-1") {
    id
    text
    timestamp
  }
}</pre>

		<h3>Subscribe to Messages</h3>
		<pre>subscription {
  messageAdded {
    id
    text
    userId
    timestamp
    user {
      id
      name
    }
  }
}</pre>

		<h3>Countdown Subscription</h3>
		<pre>subscription {
  countdown(from: 5)
}</pre>

		<h3>User Status Subscription</h3>
		<pre>subscription {
  userStatus(userId: "user-1") {
    userId
    status
    timestamp
  }
}</pre>
	</div>

	<div class="endpoint">
		<h2>Using with Apollo Client</h2>
		<pre>import { GraphQLWsLink } from '@apollo/client/link/subscriptions';
import { createClient } from 'graphql-ws';

const wsLink = new GraphQLWsLink(
  createClient({
    url: 'ws://localhost:8080/graphql/ws',
  })
);</pre>
	</div>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	log.Println("🚀 Rocket Subscriptions Example")
	log.Println("   HTTP:       http://localhost:8080/graphql")
	log.Println("   WebSocket:  ws://localhost:8080/graphql/ws")
	log.Println("   Playground: http://localhost:8080/playground")
	log.Println("   Home:       http://localhost:8080/")
	log.Println("")
	log.Println("Server running on http://localhost:8080")

	// Start a goroutine to simulate some activity
	go func() {
		time.Sleep(5 * time.Second)
		log.Println("💡 Tip: Open the playground and try subscribing to messageAdded!")
	}()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

