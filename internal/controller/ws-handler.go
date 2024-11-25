package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

type Input struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type Output struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

func (c Controller) CreateRoom(w http.ResponseWriter, r *http.Request) {
	connectToken, err := c.getQueryParam(r, "connect-token")
	if err != nil {
		slog.Info("CreateRoom:", "error", err)
		fmt.Fprint(w, err)
		return
	}

	slog.Debug("CreateRoom: connect-token recieved", "connect_token", connectToken)

	memberID, err := c.roomService.GetMemberIDByConnectToken(r.Context(), connectToken)
	if err != nil {
		slog.Info("CreateRoom:", "error", err)
		fmt.Fprint(w, err)
		return
	}
	slog.Debug("CreateRoom: memberID recieved", "member_id", memberID)

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := c.upgrader.Upgrade(w, r, headers)
	if err != nil {
		slog.Warn("CreateRoom: failed to upgrade connection", "error", err)
		return
	}
	slog.Debug("CreateRoom: connection established", "user", connectToken)

	if err := c.roomService.ConnectMember(r.Context(), conn, memberID); err != nil {
		slog.Warn("CreateRoom: failed to connect member", "error", err)
		return
	}

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

			// if err != nil {
			// } else {
			// 	r.sendVideoAdded(video)
			// }
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

func (c Controller) writeError(conn *websocket.Conn, err error) error {
	return conn.WriteJSON(Output{
		Action: "error",
		Data:   err.Error(),
	})
}

// func (c Controller) JoinRoom(w http.ResponseWriter, r *http.Request) {
// 	// for _, c := range r.Cookies() {
// 	// 	fmt.Printf("Cookie: %#v\n", c)
// 	// }

// 	user, err := c.getUser(r)
// 	if err != nil {
// 		slog.Info("JoinRoom:", "error", err)
// 		fmt.Fprint(w, err)
// 		return
// 	}

// 	slog.Debug("JoinRoom: user recieved", "user", user)

// 	roomID := chi.URLParam(r, "room-id")
// 	room, err := c.roomService.GetRoom(roomID)
// 	if err != nil {
// 		slog.Info("JoinRoom: failed to get room", "error", err)
// 		fmt.Fprint(w, err)
// 		return
// 	}

// 	headers := http.Header{}
// 	// headers.Add("Set-Cookie", cookieString)
// 	conn, err := c.upgrader.Upgrade(w, r, headers)
// 	if err != nil {
// 		slog.Warn("JoinRoom: failed to upgrade connection", "error", err)
// 		return
// 	}
// 	slog.Debug("JoinRoom: connection established", "user", user)

// 	user.Conn = conn

// 	room.AddMember(user)

// 	go room.ReadMessages(conn)
// }
