package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jest-cloud/rocket"
)

// Context keys
type contextKey string

const (
	UserIDKey contextKey = "userID"
	UserKey   contextKey = "user"
)

// User model
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Resolvers
type Resolvers struct{}

func (r *Resolvers) QueryResolvers() map[string]rocket.FieldResolveFn {
	return map[string]rocket.FieldResolveFn{
		"hello": func(p rocket.ResolveParams) (interface{}, error) {
			return "Hello, World!", nil
		},
		"currentUser": func(p rocket.ResolveParams) (interface{}, error) {
			// Get user from context
			user, ok := p.Context.Value(UserKey).(*User)
			if !ok {
				return nil, fmt.Errorf("unauthorized: user not authenticated")
			}
			return user, nil
		},
		"me": func(p rocket.ResolveParams) (interface{}, error) {
			// Get user ID from context
			userID, ok := p.Context.Value(UserIDKey).(string)
			if !ok {
				return nil, fmt.Errorf("unauthorized")
			}
			// In real app, fetch from database
			return &User{
				ID:    userID,
				Name:  "Alice",
				Email: "alice@example.com",
			}, nil
		},
	}
}

func (r *Resolvers) MutationResolvers() map[string]rocket.FieldResolveFn {
	return map[string]rocket.FieldResolveFn{}
}

func (r *Resolvers) SubscriptionResolvers() map[string]rocket.SubscriptionResolveFn {
	return map[string]rocket.SubscriptionResolveFn{}
}

func (r *Resolvers) TypeResolvers() map[string]map[string]rocket.FieldResolveFn {
	return map[string]map[string]rocket.FieldResolveFn{}
}

// Mock JWT verification
func verifyJWT(token string) (string, error) {
	// In real app, verify JWT signature and expiry
	if token == "" {
		return "", fmt.Errorf("empty token")
	}
	// Mock: extract userID from token
	return "user-123", nil
}

// Mock user lookup
func getUserByID(userID string) (*User, error) {
	// In real app, fetch from database
	return &User{
		ID:    userID,
		Name:  "Alice Johnson",
		Email: "alice@example.com",
	}, nil
}

// OPTION 1: Middleware Pattern (Recommended for most use cases)
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		ctx := r.Context()

		if token != "" {
			// Verify token
			userID, err := verifyJWT(token)
			if err == nil {
				// Add user ID to context
				ctx = context.WithValue(ctx, UserIDKey, userID)

				// Optionally fetch and add full user
				user, err := getUserByID(userID)
				if err == nil {
					ctx = context.WithValue(ctx, UserKey, user)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	// Choose which pattern to use

	// OPTION 1: Middleware Pattern
	fmt.Println("Starting GraphQL server with MIDDLEWARE pattern on :8080")
	runWithMiddleware()

	// OPTION 2: Context Builder Pattern (uncomment to try)
	// fmt.Println("Starting GraphQL server with CONTEXT BUILDER pattern on :8080")
	// runWithContextBuilder()
}

func runWithMiddleware() {
	// Build schema without ContextBuilder
	schema, err := rocket.BuildSchema(
		rocket.Config{SchemaPath: "schema.graphql"},
		&Resolvers{},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Chain middleware
	handler := rocket.Handler(schema)
	handler = AuthMiddleware(handler) // Add auth middleware

	http.Handle("/graphql", handler)
	http.Handle("/", rocket.PlaygroundHandler("/graphql"))

	fmt.Println("🚀 GraphQL endpoint: http://localhost:8080/graphql")
	fmt.Println("🎮 Playground:       http://localhost:8080/")
	fmt.Println("\nTry these queries:")
	fmt.Println(`
Without auth:
  { hello }

With auth (add header: Authorization: Bearer YOUR_TOKEN):
  { currentUser { id name email } }
  { me { id name email } }
`)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func runWithContextBuilder() {
	// Build schema WITH ContextBuilder (Apollo-style)
	schema, err := rocket.BuildSchema(
		rocket.Config{
			SchemaPath: "schema.graphql",
			// Apollo-style context builder
			ContextBuilder: func(r *http.Request) context.Context {
				ctx := r.Context()

				// Extract and verify token
				authHeader := r.Header.Get("Authorization")
				token := strings.TrimPrefix(authHeader, "Bearer ")

				if token != "" {
					userID, err := verifyJWT(token)
					if err == nil {
						// Add user ID
						ctx = context.WithValue(ctx, UserIDKey, userID)

						// Fetch and add full user
						user, err := getUserByID(userID)
						if err == nil {
							ctx = context.WithValue(ctx, UserKey, user)
						}
					}
				}

				// Add request start time
				ctx = context.WithValue(ctx, "requestStartTime", time.Now())

				return ctx
			},
		},
		&Resolvers{},
	)
	if err != nil {
		log.Fatal(err)
	}

	// No middleware needed - context builder handles everything
	http.Handle("/graphql", rocket.Handler(schema))
	http.Handle("/", rocket.PlaygroundHandler("/graphql"))

	fmt.Println("🚀 GraphQL endpoint: http://localhost:8080/graphql")
	fmt.Println("🎮 Playground:       http://localhost:8080/")
	fmt.Println("\nTry these queries:")
	fmt.Println(`
Without auth:
  { hello }

With auth (add header: Authorization: Bearer YOUR_TOKEN):
  { currentUser { id name email } }
  { me { id name email } }
`)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

