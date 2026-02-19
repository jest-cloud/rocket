package websocket

import (
	"encoding/json"
)

// graphql-ws protocol message types (modern)
// https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md
const (
	// Client -> Server
	MessageTypeConnectionInit = "connection_init"
	MessageTypeSubscribe      = "subscribe"
	MessageTypeComplete       = "complete"
	MessageTypePing           = "ping"

	// Server -> Client
	MessageTypeConnectionAck = "connection_ack"
	MessageTypeNext          = "next"
	MessageTypeError         = "error"
	// MessageTypeComplete is used bidirectionally (already defined above)
	MessageTypePong = "pong"
)

// Legacy subscriptions-transport-ws protocol message types.
// These are accepted for backward compatibility with older clients and
// gateways (e.g., Cosmo Router in AUTO sub-protocol mode).
// https://github.com/apollographql/subscriptions-transport-ws/blob/master/PROTOCOL.md
const (
	LegacyMessageTypeStart               = "start"                // equivalent to "subscribe"
	LegacyMessageTypeStop                = "stop"                 // equivalent to "complete"
	LegacyMessageTypeConnectionTerminate = "connection_terminate" // client wants to close
)

// Message represents a graphql-ws protocol message
type Message struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// SubscribePayload is the payload for a subscribe message
type SubscribePayload struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// NextPayload is the payload for a next message (subscription data)
type NextPayload struct {
	Data   interface{}            `json:"data,omitempty"`
	Errors []GraphQLError          `json:"errors,omitempty"`
}

// ErrorPayload is the payload for an error message
type ErrorPayload struct {
	Message string `json:"message"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message string        `json:"message"`
	Path    []interface{} `json:"path,omitempty"`
}

// ConnectionInitMessage creates a connection_init message
func ConnectionInitMessage() Message {
	return Message{Type: MessageTypeConnectionInit}
}

// ConnectionAckMessage creates a connection_ack message
func ConnectionAckMessage() Message {
	return Message{Type: MessageTypeConnectionAck}
}

// NextMessage creates a next message with data
func NextMessage(id string, data interface{}, errors []GraphQLError) (Message, error) {
	payload := NextPayload{
		Data:   data,
		Errors: errors,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{
		ID:      id,
		Type:    MessageTypeNext,
		Payload: payloadBytes,
	}, nil
}

// ErrorMessage creates an error message
func ErrorMessage(id string, errorMsg string) (Message, error) {
	payload := ErrorPayload{Message: errorMsg}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{
		ID:      id,
		Type:    MessageTypeError,
		Payload: payloadBytes,
	}, nil
}

// CompleteMessage creates a complete message
func CompleteMessage(id string) Message {
	return Message{
		ID:   id,
		Type: MessageTypeComplete,
	}
}

// PongMessage creates a pong message
func PongMessage() Message {
	return Message{Type: MessageTypePong}
}

