package controller

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sharetube/server/internal/service/room"
)

func (c controller) createRoom(w http.ResponseWriter, r *http.Request) {
	user, err := c.getUser(r)
	if err != nil {
		c.logger.DebugContext(r.Context(), "failed to get user", "error", err)
		return
	}

	initialVideoURL, err := c.getQueryParam(r, "video-url")
	if err != nil {
		c.logger.DebugContext(r.Context(), "failed to get query param", "error", err)
		return
	}

	createRoomResponse, err := c.roomService.CreateRoom(r.Context(), &room.CreateRoomParams{
		Username:        user.username,
		Color:           user.color,
		AvatarURL:       user.avatarURL,
		InitialVideoURL: initialVideoURL,
	})
	if err != nil {
		c.logger.DebugContext(r.Context(), "failed to create room", "error", err)
		return
	}
	defer c.disconnect(r.Context(), createRoomResponse.RoomId, createRoomResponse.MemberId)

	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to upgrade to websocket", "error", err)
		return
	}

	if err := c.roomService.ConnectMember(r.Context(), &room.ConnectMemberParams{
		Conn:     conn,
		MemberId: createRoomResponse.MemberId,
	}); err != nil {
		c.logger.ErrorContext(r.Context(), "failed to connect member", "error", err)
		return
	}

	roomState, err := c.roomService.GetRoom(r.Context(), createRoomResponse.RoomId)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to get room state", "error", err)
		return
	}

	if err := conn.WriteJSON(&Output{
		Type: "JOINED_ROOM",
		Payload: map[string]any{
			"jwt":  createRoomResponse.JWT,
			"room": roomState,
		},
	}); err != nil {
		c.logger.ErrorContext(r.Context(), "failed to write json", "error", err)
		return
	}

	ctx := context.WithValue(r.Context(), roomIdCtxKey, createRoomResponse.RoomId)
	ctx = context.WithValue(ctx, memberIdCtxKey, createRoomResponse.MemberId)

	if err := c.wsmux.ServeConn(ctx, conn); err != nil {
		c.logger.InfoContext(r.Context(), "failed to serve conn", "error", err)
		return
	}
}

func (c controller) joinRoom(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "room-id")
	if roomId == "" {
		c.logger.DebugContext(r.Context(), "empty room id")
		return
	}

	user, err := c.getUser(r)
	if err != nil {
		c.logger.DebugContext(r.Context(), "failed to get user", "error", err)
		return
	}

	userJWT, _ := c.getQueryParam(r, "jwt")

	joinRoomResponse, err := c.roomService.JoinRoom(r.Context(), &room.JoinRoomParams{
		JWT:       userJWT,
		Username:  user.username,
		Color:     user.color,
		AvatarURL: user.avatarURL,
		RoomId:    roomId,
	})
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to join room", "error", err)
		return
	}
	defer c.disconnect(r.Context(), roomId, joinRoomResponse.JoinedMember.Id)

	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to upgrade to websocket", "error", err)
		return
	}
	defer conn.Close()

	if err := c.roomService.ConnectMember(r.Context(), &room.ConnectMemberParams{
		Conn:     conn,
		MemberId: joinRoomResponse.JoinedMember.Id,
	}); err != nil {
		c.logger.ErrorContext(r.Context(), "failed to connect member", "error", err)
		return
	}

	roomState, err := c.roomService.GetRoom(r.Context(), roomId)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to get room state", "error", err)
		return
	}

	if err := conn.WriteJSON(&Output{
		Type: "JOINED_ROOM",
		// todo: define output data structs
		Payload: map[string]any{
			"jwt":  joinRoomResponse.JWT,
			"room": roomState,
		},
	}); err != nil {
		c.logger.ErrorContext(r.Context(), "failed to write json", "error", err)
		return
	}

	if err := c.broadcast(joinRoomResponse.Conns, &Output{
		Type: "MEMBER_JOINED",
		Payload: map[string]any{
			"joined_member": joinRoomResponse.JoinedMember,
			"member_list":   joinRoomResponse.MemberList,
		},
	}); err != nil {
		c.logger.WarnContext(r.Context(), "failed to broadcast", "error", err)
		return
	}

	ctx := context.WithValue(r.Context(), roomIdCtxKey, roomId)
	ctx = context.WithValue(ctx, memberIdCtxKey, joinRoomResponse.JoinedMember.Id)

	if err := c.wsmux.ServeConn(ctx, conn); err != nil {
		c.logger.InfoContext(r.Context(), "failed to serve conn", "error", err)
		return
	}
}
