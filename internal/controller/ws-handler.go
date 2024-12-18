package controller

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	o "github.com/skewb1k/optional"
)

type Output struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

func (c controller) handleAlive(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)
	c.logger.InfoContext(ctx, "alive", "room_id", roomId, "member_id", memberId)
}

func (c controller) handleUpdatePlayerState(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)
	c.logger.InfoContext(ctx, "update player state", "room_id", roomId, "member_id", memberId)

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
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
		return
	}
}

func (c controller) handleUpdatePlayerVideo(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)
	c.logger.InfoContext(ctx, "update player video", "room_id", roomId, "member_id", memberId)

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
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)
	c.logger.InfoContext(ctx, "add video", "room_id", roomId, "member_id", memberId)

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
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)
	c.logger.InfoContext(ctx, "remove member", "room_id", roomId, "member_id", memberId)

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
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)
	c.logger.InfoContext(ctx, "promote member", "room_id", roomId, "member_id", memberId)

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
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)
	c.logger.InfoContext(ctx, "remove video", "room_id", roomId, "member_id", memberId)

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
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)
	c.logger.InfoContext(ctx, "update profile", "room_id", roomId, "member_id", memberId)

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

func (c controller) handleUpdateIsReady(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)
	c.logger.InfoContext(ctx, "update is_ready", "room_id", roomId, "member_id", memberId)

	var data struct {
		IsReady bool `json:"is_ready"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	updatePlayerVideoResp, err := c.roomService.UpdateIsReady(ctx, &room.UpdateIsReadyParams{
		IsReady:    data.IsReady,
		SenderId:   memberId,
		RoomId:     roomId,
		SenderConn: conn,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to update player video", "error", err)
		if err := c.writeError(conn, err); err != nil {
			c.logger.ErrorContext(ctx, "failed to write error", "error", err)
		}

		return
	}

	if err := c.broadcast(updatePlayerVideoResp.Conns, &Output{
		Type: "MEMBER_UPDATED",
		Payload: map[string]any{
			"updated_member": updatePlayerVideoResp.UpdatedMember,
			"members":        updatePlayerVideoResp.Members,
		},
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
		return
	}

	if updatePlayerVideoResp.PlayerState != nil {
		if err := c.broadcast(updatePlayerVideoResp.Conns, &Output{
			Type:    "PLAYER_UPDATED",
			Payload: updatePlayerVideoResp.PlayerState,
		}); err != nil {
			c.logger.ErrorContext(ctx, "failed to broadcast", "error", err)
			return
		}
	}
}
