package controller

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/wsrouter"
)

type Output struct {
	Action string `json:"action"`
	Data   any    `json:"data"`
}

func (c controller) handleGetState(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	roomState, err := c.roomService.GetRoomState(ctx, roomID)
	if err != nil {
		slog.Warn("failed to get room state", "error", err)
		if err := c.writeError(conn, err); err != nil {
			slog.Warn("failed to write error", "error", err)
			return
		}
	}

	if err := conn.WriteJSON(&Output{
		Action: "room_state",
		Data:   roomState,
	}); err != nil {
		slog.Warn("failed to write message", "error", err)
		return
	}
}

func (c controller) handleAddVideo(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	memberID := c.getMemberIDFromCtx(ctx)

	var data struct {
		VideoURL string `json:"video_url"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	addVideoResponse, err := c.roomService.AddVideo(ctx, &room.AddVideoParams{
		MemberID: memberID,
		RoomID:   roomID,
		VideoURL: data.VideoURL,
	})
	if err != nil {
		slog.Warn("failed to add video", "error", err)
		if err := c.writeError(conn, err); err != nil {
			slog.Warn("failed to write error", "error", err)
			return
		}
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
}

func (c controller) handleRemoveMember(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	memberID := c.getMemberIDFromCtx(ctx)

	var data struct {
		MemberID string `json:"member_id"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	removeVideoResponse, err := c.roomService.RemoveMember(ctx, &room.RemoveMemberParams{
		RemovedMemberID: data.MemberID,
		SenderID:        memberID,
		RoomID:          roomID,
	})
	if err != nil {
		slog.Warn("failed to remove member", "error", err)
		if err := c.writeError(conn, err); err != nil {
			slog.Warn("failed to write error", "error", err)
			return
		}
	}

	if err := c.broadcast(removeVideoResponse.Conns, &Output{
		Action: "member_removed",
		Data: map[string]any{
			"removed_member_id": data.MemberID,
			"memberlist":        removeVideoResponse.Memberlist,
		},
	}); err != nil {
		slog.Warn("failed to broadcast", "error", err)
		return
	}
}

func (c controller) handlePromoteMember(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	memberID := c.getMemberIDFromCtx(ctx)

	var data struct {
		MemberID string `json:"member_id"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	promoteMemberResp, err := c.roomService.PromoteMember(ctx, &room.PromoteMemberParams{
		PromotedMemberID: data.MemberID,
		SenderID:         memberID,
		RoomID:           roomID,
	})
	if err != nil {
		slog.Warn("failed to promote member", "error", err)
		if err := c.writeError(conn, err); err != nil {
			slog.Warn("failed to write error", "error", err)
			return
		}
	}

	if err := c.broadcast(promoteMemberResp.Conns, &Output{
		Action: "member_promoted",
		Data: map[string]any{
			"promoted_member_id": data.MemberID,
		},
	}); err != nil {

		slog.Warn("failed to broadcast", "error", err)
		return
	}
}

func (c controller) handleRemoveVideo(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	memberID := c.getMemberIDFromCtx(ctx)

	var data struct {
		VideoID string `json:"video_id"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		slog.Warn("failed to unmarshal data", "error", err)
		if err := c.writeError(conn, err); err != nil {
			slog.Warn("failed to write error", "error", err)
			return
		}
	}

	addVideoResponse, err := c.roomService.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoID:  data.VideoID,
		SenderID: memberID,
		RoomID:   roomID,
	})
	if err != nil {
		slog.Warn("failed to remove video", "error", err)
		if err := c.writeError(conn, err); err != nil {
			slog.Warn("failed to write error", "error", err)
			return
		}
	}

	if err := c.broadcast(addVideoResponse.Conns, &Output{
		Action: "video_removed",
		Data: map[string]any{
			"removed_video": data.VideoID,
			"playlist":      addVideoResponse.Playlist,
		},
	}); err != nil {

		slog.Warn("failed to broadcast", "error", err)
		return
	}
}

func (c controller) getWSRouter() *wsrouter.WSRouter {
	mux := wsrouter.New()
	mux.Handle("GET_STATE", c.handleGetState)
	// video
	mux.Handle("ADD_VIDEO", c.handleAddVideo)
	mux.Handle("REMOVE_VIDEO", c.handleRemoveVideo)
	// member
	mux.Handle("PROMOTE_MEMBER", c.handlePromoteMember)
	mux.Handle("REMOVE_MEMBER", c.handleRemoveMember)

	return mux
}
