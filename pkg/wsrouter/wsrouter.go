package wsrouter

import (
	"context"
	"encoding/json"

	"github.com/gorilla/websocket"
)

type message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type HandlerFunc func(ctx context.Context, conn *websocket.Conn, payload json.RawMessage)

type WSRouter struct {
	routes map[string]HandlerFunc
}

func New() *WSRouter {
	return &WSRouter{routes: make(map[string]HandlerFunc)}
}

func (r *WSRouter) Handle(messageType string, handler HandlerFunc) {
	r.routes[messageType] = handler
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
			handler(ctx, conn, msg.Payload)
		} else {
			conn.WriteJSON(map[string]string{"error": "Unknown message type"})
		}
	}
}
