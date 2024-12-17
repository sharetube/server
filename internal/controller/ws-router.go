package controller

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/wsrouter"
	o "github.com/skewb1k/optional"
)

func (c controller) getWSRouter() *wsrouter.WSRouter {
	mux := wsrouter.New()

	// video
	mux.Handle("ADD_VIDEO", c.handleAddVideo)
	mux.Handle("REMOVE_VIDEO", c.handleRemoveVideo)
	// mux.Handle("REORDER_PLAYLIST", c.handleRemoveVideo)

	// member
	mux.Handle("PROMOTE_MEMBER", c.handlePromoteMember)
	mux.Handle("REMOVE_MEMBER", c.handleRemoveMember)

	// player
	mux.Handle("UPDATE_PLAYER_STATE", c.handleUpdatePlayerState)
	mux.Handle("UPDATE_PLAYER_VIDEO", c.handleUpdatePlayerVideo)

	// profile
	mux.Handle("UPDATE_PROFILE", c.handleUpdateProfile)
	// mux.Handle("UPDATE_MUTED", c.handleUpdateMuted)
	// mux.Handle("UPDATE_READY", c.handleUpdateReady)

	return mux
}

type Output struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

func (c controller) handleUpdatePlayerState(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	c.logger.InfoContext(ctx, "update player state")
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

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
		SenderId:     memberId,
		RoomId:       roomId,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to update player state", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
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
	c.logger.InfoContext(ctx, "update player video")
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		VideoId   string `json:"video_id"`
		UpdatedAt int    `json:"updated_at"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	updatePlayerVideoResp, err := c.roomService.UpdatePlayerVideo(ctx, &room.UpdatePlayerVideoParams{
		VideoId:   data.VideoId,
		UpdatedAt: data.UpdatedAt,
		SenderId:  memberId,
		RoomId:    roomId,
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
	c.logger.InfoContext(ctx, "add video")
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		VideoURL string `json:"video_url"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	addVideoResponse, err := c.roomService.AddVideo(ctx, &room.AddVideoParams{
		SenderId: memberId,
		RoomId:   roomId,
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
			"playlist":    addVideoResponse.Playlist,
		},
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
		return
	}
}

func (c controller) handleRemoveMember(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	c.logger.InfoContext(ctx, "remove member")
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		MemberId uuid.UUID `json:"member_id"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		c.logger.DebugContext(ctx, "failed to join room", "error", err)
		return
	}

	if data.MemberId == uuid.Nil {
		err := ErrValidationError
		c.logger.DebugContext(ctx, "validation error", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	removeMemberResp, err := c.roomService.RemoveMember(ctx, &room.RemoveMemberParams{
		RemovedMemberId: data.MemberId.String(),
		SenderId:        memberId,
		RoomId:          roomId,
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
	c.logger.InfoContext(ctx, "promote member")
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		MemberId uuid.UUID `json:"member_id"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		c.logger.DebugContext(ctx, "failed to unmarshal json", "error", err)
		return
	}

	if data.MemberId == uuid.Nil {
		err := ErrValidationError
		c.logger.DebugContext(ctx, "validation error", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	promoteMemberResp, err := c.roomService.PromoteMember(ctx, &room.PromoteMemberParams{
		PromotedMemberId: data.MemberId.String(),
		SenderId:         memberId,
		RoomId:           roomId,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to promote member", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	if err := c.broadcast(promoteMemberResp.Conns, &Output{
		Type: "MEMBER_UPDATED",
		Payload: map[string]any{
			"updated_member": promoteMemberResp.PromotedMember,
			"members":        promoteMemberResp.Members,
		},
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
		return
	}
}

func (c controller) handleRemoveVideo(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	c.logger.InfoContext(ctx, "remove video")
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		VideoId string `json:"video_id"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		c.logger.DebugContext(ctx, "failed to unmarshal json", "error", err)
		return
	}

	addVideoResponse, err := c.roomService.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoId:  data.VideoId,
		SenderId: memberId,
		RoomId:   roomId,
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
			"removed_video_id": data.VideoId,
			"playlist":         addVideoResponse.Playlist,
		},
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
		return
	}
}

func (c controller) handleUpdateProfile(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	c.logger.InfoContext(ctx, "update profile")
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		Username  *string         `json:"username"`
		Color     *string         `json:"color"`
		AvatarURL o.Field[string] `json:"avatar_url"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	if data.Username == nil && data.Color == nil && !data.AvatarURL.Defined {
		err := ErrValidationError
		c.logger.DebugContext(ctx, "validation error", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}
	// todo: add validation

	updateProfileResp, err := c.roomService.UpdateProfile(ctx, &room.UpdateProfileParams{
		Username:  data.Username,
		Color:     data.Color,
		AvatarURL: data.AvatarURL,
		SenderId:  memberId,
		RoomId:    roomId,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to update member", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	if err := c.broadcast(updateProfileResp.Conns, &Output{
		Type: "MEMBER_UPDATED",
		Payload: map[string]any{
			"updated_member": updateProfileResp.UpdatedMember,
			"members":        updateProfileResp.Members,
		},
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
		return
	}
}
