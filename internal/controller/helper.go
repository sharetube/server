package controller

import (
	"encoding/json"
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

type user struct {
	authToken string
	username  string
	color     string
	avatarURL string
}

func (c controller) getUser(r *http.Request) (user, error) {
	// authToken, err := c.getQueryParam(r, "auth-token")
	// if err != nil {
	// 	return user{}, err
	// }

	username, err := c.getQueryParam(r, "username")
	if err != nil {
		return user{}, err
	}

	color, err := c.getQueryParam(r, "color")
	if err != nil {
		return user{}, err
	}

	avatarURL, err := c.getQueryParam(r, "avatar-url")
	if err != nil {
		return user{}, err
	}

	return user{
		// authToken: authToken,
		username:  username,
		color:     color,
		avatarURL: avatarURL,
	}, nil
}

func (c controller) writeError(conn *websocket.Conn, err error) error {
	return conn.WriteJSON(Output{
		Action: "error",
		Data:   err.Error(),
	})
}

func (c controller) broadcast(conns []*websocket.Conn, output *Output) error {
	for _, conn := range conns {
		if err := conn.WriteJSON(output); err != nil {
			slog.Warn("failed to broadcast", "error", err)
		}
	}

	return nil
}

func (c controller) unmarshalJSONorError(conn *websocket.Conn, data json.RawMessage, v any) error {
	if err := json.Unmarshal(data, &v); err != nil {
		slog.Warn("failed to unmarshal data", "error", err)
		if err := c.writeError(conn, err); err != nil {
			slog.Warn("failed to write error", "error", err)
			return err
		}
		return err
	}

	return nil
}
