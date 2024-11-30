package controller

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/rest"
)

type contextKey int

const (
	roomIDCtxKey contextKey = iota
	memberIDCtxKey
)

func (c controller) createRoom(w http.ResponseWriter, r *http.Request) {
	user, err := c.getUser(r)
	if err != nil {
		slog.Info("CreateRoom", "error", err)
		return
	}

	initialVideoURL, err := c.getQueryParam(r, "video-url")
	if err != nil {
		slog.Info("CreateRoom", "error", err)
		return
	}

	createRoomResponse, err := c.roomService.CreateRoom(r.Context(), &room.CreateRoomParams{
		Username:        user.username,
		Color:           user.color,
		AvatarURL:       user.avatarURL,
		InitialVideoURL: initialVideoURL,
	})
	if err != nil {
		slog.Info("CreateRoom", "error", err)
		return
	}

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := c.upgrader.Upgrade(w, r, headers)
	if err != nil {
		slog.Warn("CreateRoom failed to upgrade connection", "error", err)
		return
	}

	if err := c.roomService.ConnectMember(&room.ConnectMemberParams{
		Conn:     conn,
		MemberID: createRoomResponse.MemberID,
	}); err != nil {
		slog.Warn("CreateRoom failed to connect member", "error", err)
		return
	}

	roomState, err := c.roomService.GetRoomState(r.Context(), createRoomResponse.RoomID)
	if err != nil {
		slog.Warn("CreateRoom failed to get room state", "error", err)
		return
	}

	if err := conn.WriteJSON(&Output{
		Action: "room_created",
		Data:   roomState,
	}); err != nil {
		slog.Warn("CreateRoom failed to write output", "error", err)
		return
	}

	ctx := context.WithValue(r.Context(), roomIDCtxKey, createRoomResponse.RoomID)
	ctx = context.WithValue(ctx, memberIDCtxKey, createRoomResponse.MemberID)

	c.wsmux.ServeWebSocket(ctx, conn)
}
func (c controller) joinRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "room-id")
	if roomID == "" {
		rest.WriteJSON(w, http.StatusNotFound, rest.Envelope{"error": "room not found"})
		return
	}

	user, err := c.getUser(r)
	if err != nil {
		slog.Info("JoinRoom", "error", err)
		return
	}

	joinRoomResponse, err := c.roomService.JoinRoom(r.Context(), &room.JoinRoomParams{
		Username:  user.username,
		Color:     user.color,
		AvatarURL: user.avatarURL,
		RoomID:    roomID,
	})
	if err != nil {
		slog.Info("JoinRoom", "error", err)
		return
	}

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := c.upgrader.Upgrade(w, r, headers)
	if err != nil {
		slog.Warn("JoinRoom failed to upgrade connection", "error", err)
		return
	}

	if err := c.roomService.ConnectMember(&room.ConnectMemberParams{
		Conn:     conn,
		MemberID: joinRoomResponse.JoinedMember.ID,
	}); err != nil {
		slog.Warn("JoinRoom failed to connect member", "error", err)
		return
	}

	roomState, err := c.roomService.GetRoomState(r.Context(), roomID)
	if err != nil {
		slog.Warn("JoinRoom failed to get room state", "error", err)
		return
	}

	if err := conn.WriteJSON(&Output{
		Action: "room_joined",
		Data:   roomState,
	}); err != nil {
		slog.Warn("JoinRoom failed to write output", "error", err)
		return
	}

	if err := c.broadcast(joinRoomResponse.Conns, &Output{
		Action: "member_joined",
		Data: map[string]any{
			"joined_member": joinRoomResponse.JoinedMember,
			"member_list":   joinRoomResponse.MemberList,
		},
	}); err != nil {
		slog.Warn("JoinRoom failed to broadcast", "error", err)
		return
	}

	ctx := context.WithValue(r.Context(), roomIDCtxKey, roomID)
	ctx = context.WithValue(ctx, memberIDCtxKey, joinRoomResponse.JoinedMember.ID)

	c.wsmux.ServeWebSocket(ctx, conn)
}
