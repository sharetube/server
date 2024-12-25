package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	o "github.com/skewb1k/optional"
)

type Output struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type EmptyStruct struct{}

func (es *EmptyStruct) UnmarshalJSON([]byte) error {
	return nil
}

func (c controller) handleAlive(ctx context.Context, conn *websocket.Conn, input EmptyStruct) error {
	return nil
}

type UpdatePlayerStateInput struct {
	IsPlaying    bool    `json:"is_playing"`
	CurrentTime  int     `json:"current_time"`
	PlaybackRate float64 `json:"playback_rate"`
	UpdatedAt    int     `json:"updated_at"`
}

func (c controller) handleUpdatePlayerState(ctx context.Context, conn *websocket.Conn, input UpdatePlayerStateInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	updatePlayerStateResp, err := c.roomService.UpdatePlayerState(ctx, &room.UpdatePlayerStateParams{
		IsPlaying:    input.IsPlaying,
		CurrentTime:  input.CurrentTime,
		PlaybackRate: input.PlaybackRate,
		UpdatedAt:    input.UpdatedAt,
		SenderId:     memberId,
		RoomId:       roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to update player state: %w", err)
	}

	if err := c.broadcastPlayerUpdated(ctx, updatePlayerStateResp.Conns, &updatePlayerStateResp.Player); err != nil {
		return fmt.Errorf("failed to broadcast player updated: %w", err)
	}

	return nil
}

type UpdatePlayerVideoInput struct {
	VideoId   string `json:"video_id"`
	UpdatedAt int    `json:"updated_at"`
}

func (c controller) handleUpdatePlayerVideo(ctx context.Context, conn *websocket.Conn, input UpdatePlayerVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	updatePlayerVideoResp, err := c.roomService.UpdatePlayerVideo(ctx, &room.UpdatePlayerVideoParams{
		VideoId:   input.VideoId,
		UpdatedAt: input.UpdatedAt,
		SenderId:  memberId,
		RoomId:    roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to update player video: %w", err)
	}

	if err := c.broadcast(ctx, updatePlayerVideoResp.Conns, &Output{
		Type: "PLAYER_VIDEO_UPDATED",
		Payload: map[string]any{
			"player":   updatePlayerVideoResp.Player,
			"playlist": updatePlayerVideoResp.Playlist,
		},
	}); err != nil {
		return fmt.Errorf("failed to broadcast player updated: %w", err)
	}

	return nil
}

type AddVideoInput struct {
	VideoURL string `json:"video_url"`
}

func (c controller) handleAddVideo(ctx context.Context, conn *websocket.Conn, input AddVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	// todo: add validation

	addVideoResponse, err := c.roomService.AddVideo(ctx, &room.AddVideoParams{
		SenderId: memberId,
		RoomId:   roomId,
		VideoURL: input.VideoURL,
	})
	if err != nil {
		return fmt.Errorf("failed to add video: %w", err)
	}

	if err := c.broadcast(ctx, addVideoResponse.Conns, &Output{
		Type: "VIDEO_ADDED",
		Payload: map[string]any{
			"added_video": addVideoResponse.AddedVideo,
			"playlist":    addVideoResponse.Playlist,
		},
	}); err != nil {
		return fmt.Errorf("failed to broadcast video added: %w", err)
	}

	return nil
}

type RemoveMemberInput struct {
	MemberId uuid.UUID `json:"member_id"`
}

func (c controller) handleRemoveMember(ctx context.Context, conn *websocket.Conn, input RemoveMemberInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	// validation
	if input.MemberId == uuid.Nil {
		return fmt.Errorf("validation error: %w", ErrValidationError)
	}

	removeMemberResp, err := c.roomService.RemoveMember(ctx, &room.RemoveMemberParams{
		RemovedMemberId: input.MemberId.String(),
		SenderId:        memberId,
		RoomId:          roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	// close with specific code
	removeMemberResp.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4001, ""), time.Now().Add(time.Second*5))

	if err := c.broadcast(ctx, removeMemberResp.Conns, &Output{
		Type: "MEMBER_DISCONNECTED",
		Payload: map[string]any{
			"disconnected_member_id": input.MemberId,
			"members":                removeMemberResp.Members,
		},
	}); err != nil {
		return fmt.Errorf("failed to broadcast member disconnected: %w", err)
	}

	return nil
}

type PromotedMemberInput struct {
	MemberId uuid.UUID `json:"member_id"`
}

func (c controller) handlePromoteMember(ctx context.Context, conn *websocket.Conn, input PromotedMemberInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	if input.MemberId == uuid.Nil {
		return fmt.Errorf("validation error: %w", ErrValidationError)
	}

	promoteMemberResp, err := c.roomService.PromoteMember(ctx, &room.PromoteMemberParams{
		PromotedMemberId: input.MemberId.String(),
		SenderId:         memberId,
		RoomId:           roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to promote member: %w", err)
	}

	if err := c.broadcastMemberUpdated(ctx, promoteMemberResp.Conns, &promoteMemberResp.PromotedMember, promoteMemberResp.Members); err != nil {
		return err
	}

	if err := c.writeToConn(ctx, promoteMemberResp.PromotedMemberConn, &Output{
		Type: "IS_ADMIN_UPDATED",
		Payload: map[string]any{
			"is_admin": promoteMemberResp.PromotedMember.IsAdmin,
		},
	}); err != nil {
		return fmt.Errorf("failed to write to conn: %w", err)
	}

	return nil
}

type RemoveVideoInput struct {
	VideoId string `json:"video_id"`
}

func (c controller) handleRemoveVideo(ctx context.Context, conn *websocket.Conn, input RemoveVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	removeVideoResponse, err := c.roomService.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoId:  input.VideoId,
		SenderId: memberId,
		RoomId:   roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to remove video: %w", err)
	}

	if err := c.broadcast(ctx, removeVideoResponse.Conns, &Output{
		Type: "VIDEO_REMOVED",
		Payload: map[string]any{
			"removed_video_id": input.VideoId,
			"playlist":         removeVideoResponse.Playlist,
		},
	}); err != nil {
		return fmt.Errorf("failed to broadcast video removed: %w", err)
	}

	return nil
}

type UpdateProfileInput struct {
	Username  *string         `json:"username"`
	Color     *string         `json:"color"`
	AvatarURL o.Field[string] `json:"avatar_url"`
}

func (c controller) handleUpdateProfile(ctx context.Context, conn *websocket.Conn, input UpdateProfileInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	if input.Username == nil && input.Color == nil && !input.AvatarURL.Defined {
		return fmt.Errorf("validation error: %w", ErrValidationError)
	}
	// todo: add validation

	updateProfileResp, err := c.roomService.UpdateProfile(ctx, &room.UpdateProfileParams{
		Username:  input.Username,
		Color:     input.Color,
		AvatarURL: input.AvatarURL,
		SenderId:  memberId,
		RoomId:    roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}

	if err := c.broadcastMemberUpdated(ctx, updateProfileResp.Conns, &updateProfileResp.UpdatedMember, updateProfileResp.Members); err != nil {
		return fmt.Errorf("failed to broadcast member updated: %w", err)
	}

	return nil
}

type UpdateIsReadyInput struct {
	IsReady bool `json:"is_ready"`
}

func (c controller) handleUpdateIsReady(ctx context.Context, conn *websocket.Conn, input UpdateIsReadyInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	updatePlayerVideoResp, err := c.roomService.UpdateIsReady(ctx, &room.UpdateIsReadyParams{
		IsReady:    input.IsReady,
		SenderId:   memberId,
		RoomId:     roomId,
		SenderConn: conn,
	})
	if err != nil {
		return fmt.Errorf("failed to update player video: %w", err)
	}

	if err := c.broadcastMemberUpdated(ctx, updatePlayerVideoResp.Conns, &updatePlayerVideoResp.UpdatedMember, updatePlayerVideoResp.Members); err != nil {
		return fmt.Errorf("failed to broadcast member updated: %w", err)
	}

	if updatePlayerVideoResp.Player != nil {
		if err := c.broadcastPlayerUpdated(ctx, updatePlayerVideoResp.Conns, updatePlayerVideoResp.Player); err != nil {
			return fmt.Errorf("failed to broadcast player updated: %w", err)
		}
	}

	return nil
}
