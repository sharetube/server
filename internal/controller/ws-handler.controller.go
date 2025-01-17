package controller

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service"
	"github.com/skewb1k/goutils/optional"
)

type Output struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

func (c controller) handleAlive(_ context.Context, _ *websocket.Conn, _ EmptyInput) error {
	return nil
}

type UpdatePlayerStateInput struct {
	VideoId      int     `json:"video_id"`
	IsPlaying    bool    `json:"is_playing"`
	CurrentTime  int     `json:"current_time"`
	PlaybackRate float64 `json:"playback_rate"`
	UpdatedAt    int     `json:"updated_at"`
}

func (c controller) handleUpdatePlayerState(ctx context.Context, conn *websocket.Conn, input UpdatePlayerStateInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	updatePlayerStateResp, err := c.roomService.UpdatePlayerState(ctx, &service.UpdatePlayerStateParams{
		SenderConn:   conn,
		VideoId:      input.VideoId,
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

	if err := c.broadcastPlayerStateUpdated(ctx, updatePlayerStateResp.Conns, &updatePlayerStateResp.Player); err != nil {
		return fmt.Errorf("failed to broadcast player updated: %w", err)
	}

	return nil
}

type UpdatePlayerVideoInput struct {
	VideoId   int `json:"video_id"`
	UpdatedAt int `json:"updated_at"`
}

func (c controller) handleUpdatePlayerVideo(ctx context.Context, _ *websocket.Conn, input UpdatePlayerVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	updatePlayerVideoResp, err := c.roomService.UpdatePlayerVideo(ctx, &service.UpdatePlayerVideoParams{
		VideoId:   input.VideoId,
		UpdatedAt: input.UpdatedAt,
		SenderId:  memberId,
		RoomId:    roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to update player video: %w", err)
	}

	if err := c.broadcastPlayerVideoUpdated(ctx,
		updatePlayerVideoResp.Conns,
		&updatePlayerVideoResp.Player,
		&updatePlayerVideoResp.Playlist,
		updatePlayerVideoResp.Members,
	); err != nil {
		return fmt.Errorf("failed to broadcast player updated: %w", err)
	}

	return nil
}

type AddVideoInput struct {
	VideoUrl  string `json:"video_url"`
	UpdatedAt int    `json:"updated_at"`
}

func (c controller) handleAddVideo(ctx context.Context, _ *websocket.Conn, input AddVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	addVideoResponse, err := c.roomService.AddVideo(ctx, &service.AddVideoParams{
		SenderId:  memberId,
		RoomId:    roomId,
		VideoUrl:  input.VideoUrl,
		UpdatedAt: input.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to add video: %w", err)
	}

	if addVideoResponse.AddedVideo != nil {
		if err := c.broadcast(ctx, addVideoResponse.Conns, &Output{
			Type: "VIDEO_ADDED",
			Payload: map[string]any{
				"added_video": addVideoResponse.AddedVideo,
				"playlist":    addVideoResponse.Playlist,
			},
		}); err != nil {
			return fmt.Errorf("failed to broadcast video added: %w", err)
		}
	} else {
		if err := c.broadcastPlayerVideoUpdated(ctx,
			addVideoResponse.Conns,
			addVideoResponse.Player,
			&addVideoResponse.Playlist,
			addVideoResponse.Members,
		); err != nil {
			return fmt.Errorf("failed to broadcast player updated: %w", err)
		}
	}

	return nil
}

func (c controller) handleEndVideo(ctx context.Context, _ *websocket.Conn, _ EmptyInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	addVideoResponse, err := c.roomService.EndVideo(ctx, &service.EndVideoParams{
		SenderId: memberId,
		RoomId:   roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to add video: %w", err)
	}

	if addVideoResponse.Player != nil {
		if err := c.broadcastPlayerVideoUpdated(ctx,
			addVideoResponse.Conns,
			addVideoResponse.Player,
			addVideoResponse.Playlist,
			addVideoResponse.Members,
		); err != nil {
			return fmt.Errorf("failed to broadcast player updated: %w", err)
		}
	} else {
		if err := c.broadcast(ctx, addVideoResponse.Conns, &Output{
			Type:    "VIDEO_ENDED",
			Payload: nil,
		}); err != nil {
			return fmt.Errorf("failed to broadcast video ended: %w", err)
		}
	}

	return nil
}

type RemoveMemberInput struct {
	MemberId uuid.UUID `json:"member_id"`
}

func (c controller) handleRemoveMember(ctx context.Context, _ *websocket.Conn, input RemoveMemberInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	removeMemberResp, err := c.roomService.RemoveMember(ctx, &service.RemoveMemberParams{
		RemovedMemberId: input.MemberId.String(),
		SenderId:        memberId,
		RoomId:          roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	// close with specific code
	removeMemberResp.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "kicked"))

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

func (c controller) handlePromoteMember(ctx context.Context, _ *websocket.Conn, input PromotedMemberInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	promoteMemberResp, err := c.roomService.PromoteMember(ctx, &service.PromoteMemberParams{
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
	VideoId int `json:"video_id"`
}

func (c controller) handleRemoveVideo(ctx context.Context, _ *websocket.Conn, input RemoveVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	removeVideoResponse, err := c.roomService.RemoveVideo(ctx, &service.RemoveVideoParams{
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
	Username  *string                `json:"username"`
	Color     *string                `json:"color"`
	AvatarUrl optional.Field[string] `json:"avatar_url"`
}

func (c controller) handleUpdateProfile(ctx context.Context, _ *websocket.Conn, input UpdateProfileInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	updateProfileResp, err := c.roomService.UpdateProfile(ctx, &service.UpdateProfileParams{
		Username:  input.Username,
		Color:     input.Color,
		AvatarUrl: input.AvatarUrl,
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

	updatePlayerVideoResp, err := c.roomService.UpdateIsReady(ctx, &service.UpdateIsReadyParams{
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
		if err := c.broadcastPlayerStateUpdated(ctx, updatePlayerVideoResp.Conns, updatePlayerVideoResp.Player); err != nil {
			return fmt.Errorf("failed to broadcast player updated: %w", err)
		}
	}

	return nil
}

type UpdateIsMutedInput struct {
	IsMuted bool `json:"is_muted"`
}

func (c controller) handleUpdateIsMuted(ctx context.Context, conn *websocket.Conn, input UpdateIsMutedInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	updatePlayerVideoResp, err := c.roomService.UpdateIsMuted(ctx, &service.UpdateIsMutedParams{
		IsMuted:    input.IsMuted,
		SenderId:   memberId,
		RoomId:     roomId,
		SenderConn: conn,
	})
	if err != nil {
		return fmt.Errorf("failed to update is muted: %w", err)
	}

	if err := c.broadcastMemberUpdated(ctx, updatePlayerVideoResp.Conns, &updatePlayerVideoResp.UpdatedMember, updatePlayerVideoResp.Members); err != nil {
		return fmt.Errorf("failed to broadcast member updated: %w", err)
	}

	return nil
}

type ReorderPlaylistInput struct {
	VideoIds []int `json:"video_ids"`
}

func (c controller) handleReorderPlaylist(ctx context.Context, _ *websocket.Conn, input ReorderPlaylistInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	removeVideoResponse, err := c.roomService.ReorderPlaylist(ctx, &service.ReorderPlaylistParams{
		VideoIds: input.VideoIds,
		SenderId: memberId,
		RoomId:   roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to reorder playlist: %w", err)
	}

	if err := c.broadcast(ctx, removeVideoResponse.Conns, &Output{
		Type: "PLAYLIST_REORDERED",
		Payload: map[string]any{
			"playlist": removeVideoResponse.Playlist,
		},
	}); err != nil {
		return fmt.Errorf("failed to broadcast playlist reordered: %w", err)
	}

	return nil
}
