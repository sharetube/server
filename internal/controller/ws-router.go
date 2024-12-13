package controller

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/wsrouter"
)

func (c controller) getWSRouter() *wsrouter.WSRouter {
	mux := wsrouter.New()
	mux.Handle("GET_ROOM", c.handleGetRoom)
	// video
	mux.Handle("ADD_VIDEO", c.handleAddVideo)
	mux.Handle("REMOVE_VIDEO", c.handleRemoveVideo)
	// member
	mux.Handle("PROMOTE_MEMBER", c.handlePromoteMember)
	mux.Handle("REMOVE_MEMBER", c.handleRemoveMember)

	// player
	mux.Handle("UPDATE_PLAYER_STATE", c.handleUpdatePlayerState)
	mux.Handle("UPDATE_PLAYER_VIDEO", c.handleUpdatePlayerVideo)

	return mux
}

type Output struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

func (c controller) handleGetRoom(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	roomState, err := c.roomService.GetRoom(ctx, roomID)
	if err != nil {
		c.logger.DebugContext(ctx, "failed to get room state", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	if err := conn.WriteJSON(&Output{
		Type:    "ROOM",
		Payload: roomState,
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to write json", "error", err)
		return
	}
}

func (c controller) handleUpdatePlayerState(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	memberID := c.getMemberIDFromCtx(ctx)

	var data struct {
		IsPlaying    bool    `json:"is_playing"`
		CurrentTime  int     `json:"current_time"`
		PlaybackRate float64 `json:"playback_rate"`
		UpdatedAt    int     `json:"updated_at"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	updatePlayerStateResp, err := c.roomService.UpdatePlayerState(ctx, &room.UpdatePlayerStateParams{
		IsPlaying:    data.IsPlaying,
		CurrentTime:  data.CurrentTime,
		PlaybackRate: data.PlaybackRate,
		UpdatedAt:    data.UpdatedAt,
		SenderID:     memberID,
		RoomID:       roomID,
	})
	if err != nil {
		if err := c.writeError(conn, err); err != nil {
		}

		return
	}

	if err := c.broadcast(updatePlayerStateResp.Conns, &Output{
		Type:    "PLAYER_UPDATED",
		Payload: updatePlayerStateResp.PlayerState,
	}); err != nil {
		return
	}
}

func (c controller) handleUpdatePlayerVideo(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	memberID := c.getMemberIDFromCtx(ctx)

	var data struct {
		VideoID   string `json:"video_id"`
		UpdatedAt int    `json:"updated_at"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	updatePlayerVideoResp, err := c.roomService.UpdatePlayerVideo(ctx, &room.UpdatePlayerVideoParams{
		VideoID:   data.VideoID,
		UpdatedAt: data.UpdatedAt,
		SenderID:  memberID,
		RoomID:    roomID,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to update player video", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	if err := c.broadcast(updatePlayerVideoResp.Conns, &Output{
		Type:    "PLAYER_VIDEO_UPDATED",
		Payload: updatePlayerVideoResp.Player,
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
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
		SenderID: memberID,
		RoomID:   roomID,
		VideoURL: data.VideoURL,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to add video", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	if err := c.broadcast(addVideoResponse.Conns, &Output{
		Type: "VIDEO_ADDED",
		Payload: map[string]any{
			"added_video": addVideoResponse.AddedVideo,
			"playlist":    addVideoResponse.Videos,
		},
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
		return
	}
}

func (c controller) handleRemoveMember(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	memberID := c.getMemberIDFromCtx(ctx)

	var data struct {
		MemberID uuid.UUID `json:"member_id"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		c.logger.DebugContext(ctx, "failed to join room", "error", err)
		return
	}

	if data.MemberID == uuid.Nil {
		err := ErrValidationError
		c.logger.DebugContext(ctx, "validation error", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	removeMemberResp, err := c.roomService.RemoveMember(ctx, &room.RemoveMemberParams{
		RemovedMemberID: data.MemberID.String(),
		SenderID:        memberID,
		RoomID:          roomID,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to remove member", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	//? send message to removed member that he have been removed
	removeMemberResp.Conn.Close()
}

func (c controller) handlePromoteMember(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	memberID := c.getMemberIDFromCtx(ctx)

	var data struct {
		MemberID uuid.UUID `json:"member_id"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		c.logger.DebugContext(ctx, "failed to unmarshal json", "error", err)
		return
	}

	if data.MemberID == uuid.Nil {
		err := ErrValidationError
		c.logger.DebugContext(ctx, "validation error", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	promoteMemberResp, err := c.roomService.PromoteMember(ctx, &room.PromoteMemberParams{
		PromotedMemberID: data.MemberID.String(),
		SenderID:         memberID,
		RoomID:           roomID,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to promote member", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	if err := c.broadcast(promoteMemberResp.Conns, &Output{
		Type: "MEMBER_PROMOTED",
		Payload: map[string]any{
			"promoted_member_id": data.MemberID,
		},
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
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
		c.logger.DebugContext(ctx, "failed to unmarshal json", "error", err)
		return
	}

	addVideoResponse, err := c.roomService.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoID:  data.VideoID,
		SenderID: memberID,
		RoomID:   roomID,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to remove video", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	if err := c.broadcast(addVideoResponse.Conns, &Output{
		Type: "VIDEO_REMOVED",
		Payload: map[string]any{
			"removed_video": data.VideoID,
			"playlist":      addVideoResponse.Playlist,
		},
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
		return
	}
}
