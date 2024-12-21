package wsrouter

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/gorilla/websocket"
)

type ctxKey string

const (
	messageTypeKey ctxKey = "message_type"
)

type message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type (
	HandlerFunc    func(ctx context.Context, conn *websocket.Conn, payload json.RawMessage)
	MiddlewareFunc func(HandlerFunc) HandlerFunc
)

type WSRouter struct {
	routes      map[string]HandlerFunc
	middlewares []MiddlewareFunc
}

func New() *WSRouter {
	return &WSRouter{
		routes:      make(map[string]HandlerFunc),
		middlewares: make([]MiddlewareFunc, 0),
	}
}

func (r *WSRouter) Use(middleware MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middleware)
}

func (r *WSRouter) Handle(messageType string, handler HandlerFunc) {
	finalHandler := handler
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		finalHandler = r.middlewares[i](finalHandler)
	}
	r.routes[messageType] = finalHandler
}

func (r *WSRouter) ServeConn(ctx context.Context, conn *websocket.Conn) error {
	defer conn.Close()

	for {
		// Read JSON message from the connection
		var msg message
		err := conn.ReadJSON(&msg)
		if err != nil {
			return err
		}

		// Route the message to the appropriate handler
		if handler, exists := r.routes[msg.Type]; exists {
			ctx = context.WithValue(ctx, messageTypeKey, msg.Type)
			handler(ctx, conn, msg.Payload)
		} else {
			slog.ErrorContext(ctx, "unknown message type", "type", msg.Type)
			conn.WriteJSON(map[string]string{"error": "Unknown message type"})
		}
	}
}

func GetMessageTypeFromCtx(ctx context.Context) string {
	return ctx.Value(messageTypeKey).(string)
}
