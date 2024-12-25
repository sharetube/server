package controller

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/ctxlogger"
)

func (c controller) createRoom(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "create room")
	start := time.Now()

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
		c.logger.InfoContext(r.Context(), "failed to create room", "error", err)
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to upgrade to websocket", "error", err)
		return
	}
	defer conn.Close()

	if err := c.roomService.ConnectMember(r.Context(), &room.ConnectMemberParams{
		Conn:     conn,
		MemberId: createRoomResponse.JoinedMember.Id,
	}); err != nil {
		c.logger.ErrorContext(r.Context(), "failed to connect member", "error", err)
		return
	}

	roomState, err := c.roomService.GetRoom(r.Context(), createRoomResponse.RoomId)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to get room state", "error", err)
		return
	}

	if err := c.writeToConn(r.Context(), conn, &Output{
		Type: "JOINED_ROOM",
		Payload: map[string]any{
			"jwt":           createRoomResponse.JWT,
			"joined_member": createRoomResponse.JoinedMember,
			"room":          roomState,
		},
	}); err != nil {
		return
	}

	c.logger.InfoContext(r.Context(), "room created", "room_id", createRoomResponse.RoomId, "duration", time.Since(start).Microseconds())

	ctx := context.WithValue(r.Context(), roomIdCtxKey, createRoomResponse.RoomId)
	ctx = ctxlogger.AppendCtx(ctx, slog.String("room_id", createRoomResponse.RoomId))
	ctx = context.WithValue(ctx, memberIdCtxKey, createRoomResponse.JoinedMember.Id)
	ctx = ctxlogger.AppendCtx(ctx, slog.String("sender_id", createRoomResponse.JoinedMember.Id))

	if err := c.wsmux.ServeConn(ctx, conn); err != nil {
		c.logger.InfoContext(r.Context(), "serve conn error", "error", err)
		if e, ok := err.(*websocket.CloseError); ok {
			if e.Code == 4001 {
				return
			}
		}

		disconnectMemberResp, err := c.roomService.DisconnectMember(ctx, &room.DisconnectMemberParams{
			MemberId: createRoomResponse.JoinedMember.Id,
			RoomId:   createRoomResponse.RoomId,
		})
		if err != nil {
			c.logger.DebugContext(ctx, "failed to disconnect member", "error", err)
		}

		if !disconnectMemberResp.IsRoomDeleted {
			if err := c.broadcast(ctx, disconnectMemberResp.Conns, &Output{
				Type: "MEMBER_DISCONNECTED",
				Payload: map[string]any{
					"disconnected_member_id": createRoomResponse.JoinedMember.Id,
					"members":                disconnectMemberResp.Members,
				},
			}); err != nil {
				c.logger.DebugContext(ctx, "failed to broadcast member disconnected", "error", err)
			}
		}

		return
	}
}

func (c controller) joinRoom(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "join room")
	start := time.Now()

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

	if err := c.writeToConn(r.Context(), conn, &Output{
		Type: "JOINED_ROOM",
		Payload: map[string]any{
			"jwt":           joinRoomResponse.JWT,
			"joined_member": joinRoomResponse.JoinedMember,
			"room":          roomState,
		},
	}); err != nil {
		return
	}

	if err := c.broadcast(r.Context(), joinRoomResponse.Conns, &Output{
		Type: "MEMBER_JOINED",
		Payload: map[string]any{
			"joined_member": joinRoomResponse.JoinedMember,
			"members":       joinRoomResponse.Members,
		},
	}); err != nil {
		return
	}

	c.logger.InfoContext(r.Context(), "room joined", "room_id", roomId, "duration", time.Since(start).Microseconds())

	ctx := context.WithValue(r.Context(), roomIdCtxKey, roomId)
	ctx = ctxlogger.AppendCtx(ctx, slog.String("room_id", roomId))
	ctx = context.WithValue(ctx, memberIdCtxKey, joinRoomResponse.JoinedMember.Id)
	ctx = ctxlogger.AppendCtx(ctx, slog.String("sender_id", joinRoomResponse.JoinedMember.Id))

	if err := c.wsmux.ServeConn(ctx, conn); err != nil {
		c.logger.InfoContext(r.Context(), "serve conn error", "error", err)
		if e, ok := err.(*websocket.CloseError); ok {
			if e.Code == 4001 {
				return
			}
		}

		disconnectMemberResp, err := c.roomService.DisconnectMember(ctx, &room.DisconnectMemberParams{
			MemberId: joinRoomResponse.JoinedMember.Id,
			RoomId:   roomId,
		})
		if err != nil {
			c.logger.DebugContext(ctx, "failed to disconnect member", "error", err)
		}

		if !disconnectMemberResp.IsRoomDeleted {
			if err := c.broadcast(ctx, disconnectMemberResp.Conns, &Output{
				Type: "MEMBER_DISCONNECTED",
				Payload: map[string]any{
					"disconnected_member_id": joinRoomResponse.JoinedMember.Id,
					"members":                disconnectMemberResp.Members,
				},
			}); err != nil {
				c.logger.DebugContext(ctx, "failed to broadcast member disconnected", "error", err)
			}
		}

		return
	}
}
