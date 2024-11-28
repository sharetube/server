package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/rest"
)

type Input struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type Output struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

// ? pass memberID
func (c controller) readMessages(ctx context.Context, conn *websocket.Conn) {
	for {
		var input Input
		if err := conn.ReadJSON(&input); err != nil {
			slog.Info("error reading message", "error", err)
			conn.Close()
			return
		}
		slog.Info("message recieved", "message", input)

		switch input.Action {
		// case "remove_member":
		// 	removedMember, err := r.handleRemoveMember(&input)

		// 	if err != nil {
		// 		r.sendError(input.Sender.Conn, err)
		// 	} else {
		// 		r.sendMemberLeft(removedMember)
		// 	}
		// case "promote_member":
		// 	promotedMember, err := r.handlePromoteMember(&input)

		// 	if err != nil {
		// 		r.sendError(input.Sender.Conn, err)
		// 	} else {
		// 		r.sendMemberPromoted(promotedMember)
		// 	}

		// case "demote_member":
		// 	demotedMember, err := r.handleDemoteMember(&input)

		// 	if err != nil {
		// 		r.sendError(input.Sender.Conn, err)
		// 	} else {
		// 		r.sendMemberDemoted(demotedMember)
		// 	}
		case "add_video":
			var data struct {
				VideoURL string `json:"video_url"`
			}
			if err := json.Unmarshal(input.Data, &data); err != nil {
				slog.Warn("failed to unmarshal data", "error", err)
				if err := c.writeError(conn, err); err != nil {
					slog.Warn("failed to write error", "error", err)
					return
				}
				continue
			}

			addVideoResponse, err := c.roomService.AddVideo(ctx, &room.AddVideoParams{
				Conn:     conn,
				VideoURL: data.VideoURL,
			})
			if err != nil {
				slog.Warn("failed to add video", "error", err)
				if err := c.writeError(conn, err); err != nil {
					slog.Warn("failed to write error", "error", err)
					return
				}
				continue
			}

			if err := c.broadcast(addVideoResponse.Conns, &Output{
				Action: "video_added",
				Data: map[string]any{
					"added_video": addVideoResponse.AddedVideo,
					"playlist":    addVideoResponse.Playlist,
				},
			}); err != nil {
				slog.Warn("failed to broadcast", "error", err)
				return
			}
		// case "remove_video":
		// 	video, err := r.handleRemoveVideo(&input)

		// 	if err != nil {
		// 		r.sendError(input.Sender.Conn, err)
		// 	} else {
		// 		r.sendVideoRemoved(video)
		// 	}
		// case "player_updated":
		// 	player, err := r.handlePlayerUpdated(&input)
		// 	if err != nil {
		// 		r.sendError(input.Sender.Conn, err)
		// 	} else {
		// 		r.sendPlayerUpdated(player)
		// 	}
		default:
			slog.Warn("unknown action", "action", input.Action)
			if err := c.writeError(conn, errors.New("unknown action")); err != nil {
				slog.Warn("failed to write error", "error", err)
				return
			}
		}
	}
}

func (c controller) createRoom(w http.ResponseWriter, r *http.Request) {
	connectToken, err := c.getQueryParam(r, "connect-token")
	if err != nil {
		slog.Info("CreateRoom:", "error", err)
		fmt.Fprint(w, err)
		return
	}

	slog.Debug("CreateRoom: connect-token recieved", "connect_token", connectToken)

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := c.upgrader.Upgrade(w, r, headers)
	if err != nil {
		slog.Warn("CreateRoom: failed to upgrade connection", "error", err)
		return
	}
	slog.Debug("CreateRoom: connection established", "user", connectToken)

	if err := c.roomService.CreateRoom(r.Context(), &room.CreateRoomParams{
		ConnectToken: connectToken,
		Conn:         conn,
	}); err != nil {
		slog.Info("CreateRoom:", "error", err)
		conn.Close()
		fmt.Fprint(w, err)
		return
	}

	c.readMessages(r.Context(), conn)
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
			return err
		}
	}

	return nil
}

func (c controller) joinRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "room-id")
	if roomID == "" {
		rest.WriteJSON(w, http.StatusNotFound, rest.Envelope{"error": "room not found"})
		return
	}

	connectToken, err := c.getQueryParam(r, "connect-token")
	if err != nil {
		slog.Info("JoinRoom", "error", err)
		fmt.Fprint(w, err)
		return
	}

	slog.Debug("JoinRoom connect-token recieved", "connect_token", connectToken)

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := c.upgrader.Upgrade(w, r, headers)
	if err != nil {
		slog.Warn("JoinRoom failed to upgrade connection", "error", err)
		return
	}
	slog.Debug("JoinRoom connection established", "user", connectToken)

	if err := c.roomService.JoinRoom(r.Context(), &room.JoinRoomParams{
		ConnectToken: connectToken,
		Conn:         conn,
		RoomID:       roomID,
	}); err != nil {
		slog.Info("JoinRoom", "error", err)
		conn.Close()
		fmt.Fprint(w, err)
		return
	}

	c.readMessages(r.Context(), conn)
}
