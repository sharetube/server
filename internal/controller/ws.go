package controller

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sharetube/server/internal/service"
)

func (c Controller) CreateRoom(w http.ResponseWriter, r *http.Request) {
	user, err := c.getUser(r)
	if err != nil {
		slog.Info("/ws/create-room:", "error", err)
		fmt.Fprint(w, err)
		return
	}

	user.IsAdmin = true
	slog.Debug("/ws/create-room: user recieved", "user", user)

	videoURL, err := c.getHeader(r, "Video-Url")
	if err != nil {
		slog.Info("/ws/create-room:", "error", err)
		fmt.Fprint(w, err)
		return
	}

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := c.upgrader.Upgrade(w, r, headers)
	if err != nil {
		slog.Warn("/ws/create-room: failed to upgrade connection", "error", err)
		return
	}
	slog.Debug("/ws/create-room: connection established", "user", user)

	user.Conn = conn

	roomID, room := c.roomService.CreateRoom(user, videoURL)

	room.SendMessageToAllMembers(&service.Message{
		Action: "room_created",
		Data: map[string]any{
			"room_id": roomID,
		},
	})

	slog.Info("/ws/create-room: room created", "room_id", roomID, "room", room.GetState(), "user", user)

	go room.ReadMessages(conn)
}

func (c Controller) JoinRoom(w http.ResponseWriter, r *http.Request) {
	// for _, c := range r.Cookies() {
	// 	fmt.Printf("Cookie: %#v\n", c)
	// }

	user, err := c.getUser(r)
	if err != nil {
		slog.Info("join-room handler", "error", err)
		fmt.Fprint(w, err)
		return
	}

	user.IsAdmin = false
	slog.Debug("join-room: user recieved", "user", user)

	roomID := chi.URLParam(r, "room-id")
	room, err := c.roomService.GetRoom(roomID)
	if err != nil {
		slog.Info("/ws/join-room: failed to get room", "error", err)
		fmt.Fprint(w, err)
		return
	}

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := c.upgrader.Upgrade(w, r, headers)
	if err != nil {
		slog.Warn("/ws/join-room: failed to upgrade connection", "error", err)
		return
	}
	slog.Debug("/ws/join-room: connection established", "user", user)

	user.Conn = conn

	room.AddMember(user)

	slog.Info("/ws/join-room: user joined", "room_id", roomID, "room", room.GetState(), "user", user)

	go room.ReadMessages(conn)
}
