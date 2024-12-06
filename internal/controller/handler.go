package controller

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sharetube/server/internal/service/room"
)

func (c controller) createRoom(w http.ResponseWriter, r *http.Request) {
	funcName := "controller.createRoom"
	slog.DebugContext(r.Context(), funcName, "called", "")

	user, err := c.getUser(r)
	if err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}

	initialVideoURL, err := c.getQueryParam(r, "video-url")
	if err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}

	createRoomResponse, err := c.roomService.CreateRoom(r.Context(), &room.CreateRoomParams{
		Username:        user.username,
		Color:           user.color,
		AvatarURL:       user.avatarURL,
		InitialVideoURL: initialVideoURL,
	})
	if err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}
	defer c.disconnect(r.Context(), createRoomResponse.MemberID, createRoomResponse.RoomID)

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := c.upgrader.Upgrade(w, r, headers)
	if err != nil {
		slog.ErrorContext(r.Context(), funcName, "error", err)
		return
	}

	if err := c.roomService.ConnectMember(&room.ConnectMemberParams{
		Conn:     conn,
		MemberID: createRoomResponse.MemberID,
	}); err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}

	roomState, err := c.roomService.GetRoomState(r.Context(), createRoomResponse.RoomID)
	if err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}

	if err := conn.WriteJSON(&Output{
		Action: "room_created",
		Data: map[string]any{
			"auth_token": createRoomResponse.AuthToken,
			"room_state": roomState,
		},
	}); err != nil {
		slog.ErrorContext(r.Context(), funcName, "error", err)
		return
	}

	ctx := context.WithValue(r.Context(), roomIDCtxKey, createRoomResponse.RoomID)
	ctx = context.WithValue(ctx, memberIDCtxKey, createRoomResponse.MemberID)

	if err := c.wsmux.ServeConn(ctx, conn); err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}
}

func (c controller) joinRoom(w http.ResponseWriter, r *http.Request) {
	funcName := "controller.joinRoom"
	slog.DebugContext(r.Context(), funcName, "called", "")

	roomID := chi.URLParam(r, "room-id")
	if roomID == "" {
		slog.InfoContext(r.Context(), funcName, "error", "room-id is empty")
		return
	}

	user, err := c.getUser(r)
	if err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}

	authToken := r.URL.Query().Get("auth-token")

	joinRoomResponse, err := c.roomService.JoinRoom(r.Context(), &room.JoinRoomParams{
		Username:  user.username,
		Color:     user.color,
		AvatarURL: user.avatarURL,
		AuthToken: authToken,
		RoomID:    roomID,
	})
	if err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}
	defer c.disconnect(r.Context(), joinRoomResponse.JoinedMember.ID, roomID)

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := c.upgrader.Upgrade(w, r, headers)
	if err != nil {
		slog.ErrorContext(r.Context(), funcName, "error", err)
		return
	}

	if err := c.roomService.ConnectMember(&room.ConnectMemberParams{
		Conn:     conn,
		MemberID: joinRoomResponse.JoinedMember.ID,
	}); err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}

	roomState, err := c.roomService.GetRoomState(r.Context(), roomID)
	if err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}

	if err := conn.WriteJSON(&Output{
		Action: "room_joined",
		// todo: define output data structs
		Data: map[string]any{
			"auth_token": joinRoomResponse.AuthToken,
			"room_state": roomState,
		},
	}); err != nil {
		slog.ErrorContext(r.Context(), funcName, "error", err)
		return
	}

	if err := c.broadcast(joinRoomResponse.Conns, &Output{
		Action: "member_joined",
		Data: map[string]any{
			"joined_member": joinRoomResponse.JoinedMember,
			"member_list":   joinRoomResponse.MemberList,
		},
	}); err != nil {
		slog.ErrorContext(r.Context(), funcName, "error", err)
		return
	}

	ctx := context.WithValue(r.Context(), roomIDCtxKey, roomID)
	ctx = context.WithValue(ctx, memberIDCtxKey, joinRoomResponse.JoinedMember.ID)

	if err := c.wsmux.ServeConn(ctx, conn); err != nil {
		slog.InfoContext(r.Context(), funcName, "error", err)
		return
	}
}
