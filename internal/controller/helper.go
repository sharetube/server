package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
)

func (c controller) getOptQueryParam(r *http.Request, key string) *string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil
	}

	return &value
}

func (c controller) getQueryParam(r *http.Request, key string) (string, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return "", fmt.Errorf("param %s was not provided", key)
	}

	return value, nil
}

type user struct {
	username  string
	color     string
	avatarURL *string
}

func (c controller) getUser(r *http.Request) (user, error) {
	username, err := c.getQueryParam(r, "username")
	if err != nil {
		return user{}, err
	}

	color, err := c.getQueryParam(r, "color")
	if err != nil {
		return user{}, err
	}

	avatarURL := c.getOptQueryParam(r, "avatar-url")

	return user{
		username:  username,
		color:     color,
		avatarURL: avatarURL,
	}, nil
}

func (c controller) writeToConn(ctx context.Context, conn *websocket.Conn, output *Output) error {
	c.logger.DebugContext(ctx, "writing to conn", "output", output)
	if err := conn.WriteJSON(output); err != nil {
		c.logger.ErrorContext(ctx, "failed to write to conn", "error", err)
		return err
	}

	return nil
}

func (c controller) writeError(ctx context.Context, conn *websocket.Conn, err error) error {
	return c.writeToConn(ctx, conn, &Output{
		Type:    "error",
		Payload: err.Error(),
	})
}

func (c controller) broadcast(ctx context.Context, conns []*websocket.Conn, output *Output) error {
	c.logger.DebugContext(ctx, "broadcasting", "output", output)
	var err error
	for _, conn := range conns {
		err = conn.WriteJSON(output)
	}

	return err
}

func (c controller) generateTimeBasedId() string {
	return fmt.Sprintf("%d-%s", time.Now().Unix(), uuid.NewString())
}

func (c controller) broadcastMemberUpdated(ctx context.Context, conns []*websocket.Conn, updatedMember *room.Member, members []room.Member) error {
	return c.broadcast(ctx, conns, &Output{
		Type: "MEMBER_UPDATED",
		Payload: map[string]any{
			"updated_member": updatedMember,
			"members":        members,
		},
	})
}

func (c controller) broadcastPlayerUpdated(ctx context.Context, conns []*websocket.Conn, player *room.Player) error {
	return c.broadcast(ctx, conns, &Output{
		Type: "PLAYER_UPDATED",
		Payload: map[string]any{
			"player": player,
		},
	})
}
