package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// SubprotocolGraphQLTransportWS is the WebSocket sub-protocol for the modern
// graphql-ws protocol (https://github.com/enisdenjo/graphql-ws).
const SubprotocolGraphQLTransportWS = "graphql-transport-ws"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Make this configurable for production
	},
	Subprotocols: []string{SubprotocolGraphQLTransportWS},
}

// SchemaExecutor defines the interface for executing GraphQL subscriptions
// This avoids import cycles with the main rocket package
type SchemaExecutor interface {
	ExecuteSubscription(ctx context.Context, query string, variables map[string]interface{}, operationName string) (<-chan interface{}, error)
}

// ContextBuilderProvider is an optional interface that SchemaExecutor can implement
// to provide per-request context building (e.g., for authentication).
// The WebSocket handler calls GetContextBuilder with the HTTP upgrade request
// so subscriptions receive the same auth context as queries/mutations.
type ContextBuilderProvider interface {
	GetContextBuilder() func(r *http.Request) context.Context
}

// Handler creates a WebSocket handler for GraphQL subscriptions
func Handler(schema SchemaExecutor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// Build base context from HTTP upgrade request using ContextBuilder if available.
		// This allows subscriptions to receive auth claims from the upgrade request headers.
		var baseCtx context.Context
		if provider, ok := schema.(ContextBuilderProvider); ok {
			if builder := provider.GetContextBuilder(); builder != nil {
				baseCtx = builder(r)
			}
		}
		if baseCtx == nil {
			baseCtx = context.Background()
		}

		// Create a new connection handler
		handler := &connectionHandler{
			conn:          conn,
			schema:        schema,
			baseCtx:       baseCtx,
			subscriptions: make(map[string]context.CancelFunc),
		}

		// Handle the connection
		handler.handle()
	}
}

// connectionHandler manages a single WebSocket connection
type connectionHandler struct {
	conn          *websocket.Conn
	schema        SchemaExecutor
	baseCtx       context.Context
	subscriptions map[string]context.CancelFunc
	mu            sync.Mutex
	initialized   bool
}

// handle processes WebSocket messages for this connection
func (h *connectionHandler) handle() {
	defer h.cleanup()

	for {
		var msg Message
		err := h.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if err := h.handleMessage(msg); err != nil {
			log.Printf("Error handling message: %v", err)
			// Send error to client
			errMsg, _ := ErrorMessage(msg.ID, err.Error())
			h.conn.WriteJSON(errMsg)
		}
	}
}

// handleMessage processes a single WebSocket message.
// Supports both the modern graphql-ws protocol and the legacy
// subscriptions-transport-ws protocol for backward compatibility.
func (h *connectionHandler) handleMessage(msg Message) error {
	switch msg.Type {
	case MessageTypeConnectionInit:
		return h.handleConnectionInit(msg)
	case MessageTypeSubscribe, LegacyMessageTypeStart:
		return h.handleSubscribe(msg)
	case MessageTypeComplete, LegacyMessageTypeStop:
		return h.handleComplete(msg)
	case MessageTypePing:
		return h.handlePing(msg)
	case LegacyMessageTypeConnectionTerminate:
		return fmt.Errorf("connection terminated by client")
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleConnectionInit handles connection_init message
func (h *connectionHandler) handleConnectionInit(msg Message) error {
	if h.initialized {
		return fmt.Errorf("connection already initialized")
	}

	h.initialized = true
	
	// Send connection_ack
	ackMsg := ConnectionAckMessage()
	return h.conn.WriteJSON(ackMsg)
}

// handleSubscribe handles subscribe message
func (h *connectionHandler) handleSubscribe(msg Message) error {
	if !h.initialized {
		return fmt.Errorf("connection not initialized")
	}

	// Parse payload
	var payload SubscribePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("invalid subscribe payload: %w", err)
	}

	// Create cancellable context for this subscription, inheriting auth from the upgrade request
	ctx, cancel := context.WithCancel(h.baseCtx)
	
	h.mu.Lock()
	h.subscriptions[msg.ID] = cancel
	h.mu.Unlock()

	// Execute the subscription
	resultChan, err := h.schema.ExecuteSubscription(ctx, payload.Query, payload.Variables, payload.OperationName)
	if err != nil {
		h.mu.Lock()
		delete(h.subscriptions, msg.ID)
		h.mu.Unlock()
		return fmt.Errorf("subscription error: %w", err)
	}

	// Start listening for results in a goroutine
	go h.streamResults(msg.ID, resultChan)

	return nil
}

// streamResults sends subscription results to the client
func (h *connectionHandler) streamResults(id string, resultChan <-chan interface{}) {
	defer func() {
		// Send complete message when done
		completeMsg := CompleteMessage(id)
		h.conn.WriteJSON(completeMsg)
		
		// Clean up subscription
		h.mu.Lock()
		delete(h.subscriptions, id)
		h.mu.Unlock()
	}()

	for result := range resultChan {
		// Send next message with data
		nextMsg, err := NextMessage(id, result, nil)
		if err != nil {
			log.Printf("Error creating next message: %v", err)
			continue
		}

		if err := h.conn.WriteJSON(nextMsg); err != nil {
			log.Printf("Error sending next message: %v", err)
			return
		}
	}
}

// handleComplete handles complete message (client wants to stop subscription)
func (h *connectionHandler) handleComplete(msg Message) error {
	h.mu.Lock()
	cancel, exists := h.subscriptions[msg.ID]
	if exists {
		delete(h.subscriptions, msg.ID)
	}
	h.mu.Unlock()

	if exists {
		cancel() // Cancel the subscription context
	}

	return nil
}

// handlePing handles ping message
func (h *connectionHandler) handlePing(msg Message) error {
	pongMsg := PongMessage()
	return h.conn.WriteJSON(pongMsg)
}

// cleanup closes the connection and cancels all subscriptions
func (h *connectionHandler) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Cancel all active subscriptions
	for _, cancel := range h.subscriptions {
		cancel()
	}
	h.subscriptions = make(map[string]context.CancelFunc)

	// Close WebSocket connection
	h.conn.Close()
}

