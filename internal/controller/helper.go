package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

func (c controller) writeError(conn *websocket.Conn, err error) error {
	return conn.WriteJSON(Output{
		Type:    "error",
		Payload: err.Error(),
	})
}

func (c controller) broadcast(conns []*websocket.Conn, output *Output) error {
	for _, conn := range conns {
		if err := conn.WriteJSON(output); err != nil {
			return err
		}
	}

	return nil
}

func (c controller) unmarshalJSONorError(conn *websocket.Conn, data json.RawMessage, v any) error {
	if err := json.Unmarshal(data, &v); err != nil {
		if err := c.writeError(conn, err); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (c controller) disconnect(ctx context.Context, roomId, memberId string) {
	disconnectMemberResp, err := c.roomService.DisconnectMember(ctx, &room.DisconnectMemberParams{
		MemberId: memberId,
		RoomId:   roomId,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to disconnect member", "error", err)
	}

	if !disconnectMemberResp.IsRoomDeleted {
		if err := c.broadcast(disconnectMemberResp.Conns, &Output{
			Type: "MEMBER_DISCONNECTED",
			Payload: map[string]any{
				"disconnected_member_id": memberId,
				"members":                disconnectMemberResp.Members,
			},
		}); err != nil {
			c.logger.WarnContext(ctx, "failed to broadcast", "error", err)
		}
	}
}
