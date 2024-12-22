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
}

func (c controller) handleUpdatePlayerState(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		IsPlaying    bool    `json:"is_playing"`
		CurrentTime  int     `json:"current_time"`
		PlaybackRate float64 `json:"playback_rate"`
		UpdatedAt    int     `json:"updated_at"`
	}
	if err := c.unmarshalJSONorError(ctx, conn, payload, &data); err != nil {
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
		c.logger.InfoContext(ctx, "failed to update player state", "error", err)
		c.writeError(ctx, conn, err)
		return
	}

	c.broadcastPlayerUpdated(ctx, updatePlayerStateResp.Conns, &updatePlayerStateResp.Player)
}

func (c controller) handleUpdatePlayerVideo(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		VideoId   string `json:"video_id"`
		UpdatedAt int    `json:"updated_at"`
	}
	if err := c.unmarshalJSONorError(ctx, conn, payload, &data); err != nil {
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
		c.writeError(ctx, conn, err)
		return
	}

	c.broadcast(ctx, updatePlayerVideoResp.Conns, &Output{
		Type:    "PLAYER_VIDEO_UPDATED",
		Payload: updatePlayerVideoResp.Player,
	})
}

func (c controller) handleAddVideo(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		VideoURL string `json:"video_url"`
	}
	if err := c.unmarshalJSONorError(ctx, conn, payload, &data); err != nil {
		return
	}

	// todo: add validation

	addVideoResponse, err := c.roomService.AddVideo(ctx, &room.AddVideoParams{
		SenderId: memberId,
		RoomId:   roomId,
		VideoURL: data.VideoURL,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to add video", "error", err)
		c.writeError(ctx, conn, err)
		return
	}

	c.broadcast(ctx, addVideoResponse.Conns, &Output{
		Type: "VIDEO_ADDED",
		Payload: map[string]any{
			"added_video": addVideoResponse.AddedVideo,
			"playlist":    addVideoResponse.Playlist,
		},
	})
}

func (c controller) handleRemoveMember(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		MemberId uuid.UUID `json:"member_id"`
	}
	if err := c.unmarshalJSONorError(ctx, conn, payload, &data); err != nil {
		return
	}

	if data.MemberId == uuid.Nil {
		err := ErrValidationError
		c.logger.DebugContext(ctx, "validation error", "error", err)
		c.writeError(ctx, conn, err)
		return
	}

	removeMemberResp, err := c.roomService.RemoveMember(ctx, &room.RemoveMemberParams{
		RemovedMemberId: data.MemberId.String(),
		SenderId:        memberId,
		RoomId:          roomId,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to remove member", "error", err)
		c.writeError(ctx, conn, err)

		return
	}

	//? send message to removed member that he have been removed
	removeMemberResp.Conn.Close()
}

func (c controller) handlePromoteMember(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		MemberId uuid.UUID `json:"member_id"`
	}
	if err := c.unmarshalJSONorError(ctx, conn, payload, &data); err != nil {
		return
	}

	if data.MemberId == uuid.Nil {
		err := ErrValidationError
		c.logger.DebugContext(ctx, "validation error", "error", err)
		c.writeError(ctx, conn, err)
		return
	}

	promoteMemberResp, err := c.roomService.PromoteMember(ctx, &room.PromoteMemberParams{
		PromotedMemberId: data.MemberId.String(),
		SenderId:         memberId,
		RoomId:           roomId,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to promote member", "error", err)
		c.writeError(ctx, conn, err)

		return
	}

	if err := c.broadcastMemberUpdated(ctx, promoteMemberResp.Conns, &promoteMemberResp.PromotedMember, promoteMemberResp.Members); err != nil {
		return
	}

	c.writeToConn(ctx, promoteMemberResp.PromotedMemberConn, &Output{
		Type: "IS_ADMIN_CHANGED",
		Payload: map[string]any{
			"is_admin": promoteMemberResp.PromotedMember.IsAdmin,
		},
	})
}

func (c controller) handleRemoveVideo(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		VideoId string `json:"video_id"`
	}
	if err := c.unmarshalJSONorError(ctx, conn, payload, &data); err != nil {
		return
	}

	removeVideoResponse, err := c.roomService.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoId:  data.VideoId,
		SenderId: memberId,
		RoomId:   roomId,
	})
	if err != nil {
		c.logger.DebugContext(ctx, "failed to remove video", "error", err)
		c.writeError(ctx, conn, err)
		return
	}

	c.broadcast(ctx, removeVideoResponse.Conns, &Output{
		Type: "VIDEO_REMOVED",
		Payload: map[string]any{
			"removed_video_id": data.VideoId,
			"playlist":         removeVideoResponse.Playlist,
		},
	})
}

func (c controller) handleUpdateProfile(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		Username  *string         `json:"username"`
		Color     *string         `json:"color"`
		AvatarURL o.Field[string] `json:"avatar_url"`
	}
	if err := c.unmarshalJSONorError(ctx, conn, payload, &data); err != nil {
		return
	}

	if data.Username == nil && data.Color == nil && !data.AvatarURL.Defined {
		err := ErrValidationError
		c.logger.DebugContext(ctx, "validation error", "error", err)
		c.writeError(ctx, conn, err)
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
		c.writeError(ctx, conn, err)
		return
	}

	c.broadcastMemberUpdated(ctx, updateProfileResp.Conns, &updateProfileResp.UpdatedMember, updateProfileResp.Members)
}

func (c controller) handleUpdateIsReady(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	var data struct {
		IsReady bool `json:"is_ready"`
	}
	if err := c.unmarshalJSONorError(ctx, conn, payload, &data); err != nil {
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
		c.writeError(ctx, conn, err)
		return
	}

	if err := c.broadcastMemberUpdated(ctx, updatePlayerVideoResp.Conns, &updatePlayerVideoResp.UpdatedMember, updatePlayerVideoResp.Members); err != nil {
		return
	}

	if updatePlayerVideoResp.Player != nil {
		c.broadcastPlayerUpdated(ctx, updatePlayerVideoResp.Conns, updatePlayerVideoResp.Player)
	}
}
