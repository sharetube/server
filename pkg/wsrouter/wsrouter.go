package wsrouter

import (
	"context"
	"encoding/json"
	"fmt"
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
	HandlerFunc     func(ctx context.Context, conn *websocket.Conn, payload json.RawMessage)
	MiddlewareFunc  func(HandlerFunc) HandlerFunc
	NotFoundHandler func(ctx context.Context, conn *websocket.Conn, messageType string)
)

type WSRouter struct {
	routes          map[string]HandlerFunc
	middlewares     []MiddlewareFunc
	handlerNotFound NotFoundHandler
	logger          *slog.Logger
}

func New(logger *slog.Logger) *WSRouter {
	return &WSRouter{
		routes:      make(map[string]HandlerFunc),
		middlewares: make([]MiddlewareFunc, 0),
		logger:      logger,
		handlerNotFound: func(ctx context.Context, conn *websocket.Conn, messageType string) {
			conn.WriteJSON(map[string]any{
				"type":    "ERROR",
				"payload": fmt.Sprintf("handler for type %s not found", messageType),
			})
		},
	}
}

func (r *WSRouter) SetHandlerNotFound(handler NotFoundHandler) {
	r.handlerNotFound = handler
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
	for {
		// Read JSON message from the connection
		var msg message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				r.logger.InfoContext(ctx, "websocket closed", "error", err)
			}
			return err
		}

		// Route the message to the appropriate handler
		if handler, exists := r.routes[msg.Type]; exists {
			ctx = context.WithValue(ctx, messageTypeKey, msg.Type)
			handler(ctx, conn, msg.Payload)
		} else {
			r.handlerNotFound(ctx, conn, msg.Type)
		}
	}
}

func GetMessageTypeFromCtx(ctx context.Context) string {
	return ctx.Value(messageTypeKey).(string)
}
