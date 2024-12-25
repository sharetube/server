package wsrouter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type OutputMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type (
	Middleware         func(HandlerFunc[any]) HandlerFunc[any]
	HandlerFunc[T any] func(context.Context, *websocket.Conn, T) error
	ErrorHandlerFunc   func(context.Context, *websocket.Conn, error) error
)

type WSRouter struct {
	handlers     map[string]handler
	middlewares  []Middleware
	errorHandler ErrorHandlerFunc
}

type handler interface {
	handle(context.Context, *websocket.Conn, json.RawMessage) error
}

type typedHandler[T any] struct {
	fn HandlerFunc[T]
}

func (h typedHandler[T]) handle(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) error {
	var data T
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return h.fn(ctx, conn, data)
}

func New() *WSRouter {
	return &WSRouter{
		handlers:    make(map[string]handler),
		middlewares: make([]Middleware, 0),
		errorHandler: func(ctx context.Context, conn *websocket.Conn, err error) error {
			return conn.WriteJSON(&OutputMessage{
				Type:    "ERROR",
				Payload: err.Error(),
			})
		},
	}
}

func (r *WSRouter) SetErrorHandler(f ErrorHandlerFunc) {
	r.errorHandler = f
}

// Use adds middleware to the router
func (r *WSRouter) Use(middlewares ...Middleware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// Handle registers a handler with middleware support
func Handle[T any](r *WSRouter, msgType string, handler HandlerFunc[T]) {
	// Convert the typed handler to a generic one to apply middleware
	genericHandler := func(ctx context.Context, conn *websocket.Conn, data any) error {
		return handler(ctx, conn, data.(T))
	}

	// Apply all middlewares in reverse order
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		genericHandler = r.middlewares[i](genericHandler)
	}

	// Convert back to typed handler
	finalHandler := func(ctx context.Context, conn *websocket.Conn, data T) error {
		return genericHandler(ctx, conn, data)
	}

	r.handlers[msgType] = typedHandler[T]{finalHandler}
}

func (r *WSRouter) ServeConn(ctx context.Context, conn *websocket.Conn) error {
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return err
			}

			if err := r.errorHandler(ctx, conn, err); err != nil {
				return err
			}
			continue
		}

		ctx = context.WithValue(ctx, messageTypeKey, msg.Type)
		handler, exists := r.handlers[msg.Type]

		if !exists {
			if err := r.errorHandler(ctx, conn, fmt.Errorf("handler for type %s not found", msg.Type)); err != nil {
				return err
			}
			continue
		}

		if err := handler.handle(ctx, conn, msg.Payload); err != nil {
			if err := r.errorHandler(ctx, conn, err); err != nil {
				return err
			}
		}
	}
}
