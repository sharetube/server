package controller

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

func (c controller) getQueryParam(r *http.Request, key string) (string, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return "", fmt.Errorf("param %s was not provided", key)
	}

	return value, nil
}
func (c controller) writeError(conn *websocket.Conn, err error) error {
	return conn.WriteJSON(Output{
		Action: "error",
		Data:   err.Error(),
	})
}

func (c controller) writeOutput(conn *websocket.Conn, output *Output) error {
	return conn.WriteJSON(output)
}

func (c controller) broadcast(conns []*websocket.Conn, output *Output) error {
	for _, conn := range conns {
		if err := c.writeOutput(conn, output); err != nil {
			slog.Warn("failed to broadcast", "error", err)
			// return err
		}
	}

	return nil
}
