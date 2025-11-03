package rocket

import (
	httphandler "net/http"

	rockethttp "github.com/jest-cloud/rocket/internal/http"
	rocketws "github.com/jest-cloud/rocket/internal/websocket"
)

// Handler creates an HTTP handler for GraphQL requests
// Works with both net/http and Gin (via gin.WrapH)
func Handler(schema *Schema) httphandler.HandlerFunc {
	return rockethttp.Handler(schema)
}

// WebSocketHandler creates a WebSocket handler for GraphQL subscriptions
// This implements the graphql-ws protocol for real-time subscriptions
func WebSocketHandler(schema *Schema) httphandler.HandlerFunc {
	return rocketws.Handler(schema)
}

// PlaygroundHandler creates an HTTP handler that serves a GraphQL playground
// Uses Apollo Sandbox by default (most modern and stable)
func PlaygroundHandler(endpoint string) httphandler.HandlerFunc {
	return rockethttp.PlaygroundHandler(endpoint)
}

// PlaygroundHandlerWithType creates a playground handler with a specific type
func PlaygroundHandlerWithType(endpoint string, playgroundType PlaygroundType) httphandler.HandlerFunc {
	return rockethttp.PlaygroundHandlerWithType(endpoint, rockethttp.PlaygroundType(playgroundType))
}

// PlaygroundType determines which playground interface to use
type PlaygroundType string

const (
	PlaygroundTypeApolloSandbox PlaygroundType = PlaygroundType(rockethttp.PlaygroundTypeApolloSandbox)
	PlaygroundTypeGraphiQL      PlaygroundType = PlaygroundType(rockethttp.PlaygroundTypeGraphiQL)
	PlaygroundTypePlayground    PlaygroundType = PlaygroundType(rockethttp.PlaygroundTypePlayground)
)

