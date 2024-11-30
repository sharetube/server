package wsrouter

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type HandlerFunc func(ctx context.Context, conn *websocket.Conn, payload json.RawMessage)

type WSRouter struct {
	routes map[string]HandlerFunc
}

func NewWSRouter() *WSRouter {
	return &WSRouter{routes: make(map[string]HandlerFunc)}
}

func (r *WSRouter) Handle(messageType string, handler HandlerFunc) {
	r.routes[messageType] = handler
}

func (r *WSRouter) ServeWebSocket(ctx context.Context, conn *websocket.Conn) {
	defer conn.Close()

	for {
		// Read JSON message from the connection
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Error reading message:", err)
			return
		}

		// Route the message to the appropriate handler
		if handler, exists := r.routes[msg.Type]; exists {
			handler(ctx, conn, msg.Payload)
		} else {
			log.Printf("No handler for message type: %s\n", msg.Type)
		}
	}
}
