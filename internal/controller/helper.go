package controller

import (
	"context"
	"encoding/json"
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
	// errors := make([]error, len(conns))
	// for _, conn := range conns {
	// 	if err := c.writeToConn(ctx, conn, output); err != nil {
	// 		errors = append(errors, err)
	// 	}
	// }
	// if len(errors) > 0 {
	// 	c.logger.ErrorContext(ctx, "failed to broadcast", "errors", errors)
	// 	// todo: return all errors
	// 	return errors[0]
	// }

	for _, conn := range conns {
		if err := c.writeToConn(ctx, conn, output); err != nil {
			return err
		}
	}

	return nil
}

func (c controller) unmarshalJSONorError(ctx context.Context, conn *websocket.Conn, data json.RawMessage, v any) error {
	if err := json.Unmarshal(data, &v); err != nil {
		if err := c.writeError(ctx, conn, err); err != nil {
			return err
		}

		c.logger.InfoContext(ctx, "failed to unmarshal json", "error", err)
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
		if err := c.broadcast(ctx, disconnectMemberResp.Conns, &Output{
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

func (c controller) generateTimeBasedId() string {
	return fmt.Sprintf("%d-%s", time.Now().Unix(), uuid.NewString())
}
