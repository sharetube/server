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
	Rid           string  `json:"rid"`
	VideoId       int     `json:"video_id"`
	IsPlaying     bool    `json:"is_playing"`
	CurrentTime   int     `json:"current_time"`
	PlaybackRate  float64 `json:"playback_rate"`
	UpdatedAt     int     `json:"updated_at"`
	PlayerVersion int     `json:"player_version"`
}

func (c controller) handleUpdatePlayerState(ctx context.Context, conn *websocket.Conn, input UpdatePlayerStateInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	updatePlayerStateResp, err := c.roomService.UpdatePlayerState(ctx, &service.UpdatePlayerStateParams{
		SenderConn:    conn,
		VideoId:       input.VideoId,
		IsPlaying:     input.IsPlaying,
		CurrentTime:   input.CurrentTime,
		PlaybackRate:  input.PlaybackRate,
		UpdatedAt:     input.UpdatedAt,
		PlayerVersion: input.PlayerVersion,
		SenderId:      memberId,
		RoomId:        roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to update player state: %w", err)
	}

	switch {
	case updatePlayerStateResp.PlayerVersionMismatchResponse != nil:
		if err := c.writeToConn(ctx, conn, &Output{
			Type: "PLAYER_STATE_UPDATED",
			Payload: map[string]any{
				"rid":    input.Rid,
				"player": updatePlayerStateResp.PlayerVersionMismatchResponse.Player,
			},
		}); err != nil {
			return fmt.Errorf("failed to write to sender conn: %w", err)
		}

		if err := c.broadcast(ctx, updatePlayerStateResp.Conns, &Output{
			Type: "PLAYER_STATE_UPDATED",
			Payload: map[string]any{
				"player": updatePlayerStateResp.PlayerVersionMismatchResponse.Player,
			},
		}); err != nil {
			return fmt.Errorf("failed to broadcast player state updated: %w", err)
		}
	case updatePlayerStateResp.PlayerStateUpdatedResponse != nil:
		if err := c.broadcastPlayerStateUpdated(ctx, updatePlayerStateResp.Conns, &updatePlayerStateResp.PlayerStateUpdatedResponse.Player); err != nil {
			return fmt.Errorf("failed to broadcast player updated: %w", err)
		}
	}

	return nil
}

type UpdatePlayerVideoInput struct {
	VideoId         int `json:"video_id"`
	UpdatedAt       int `json:"updated_at"`
	PlayerVersion   int `json:"player_version"`
	PlaylistVersion int `json:"playlist_version"`
}

func (c controller) handleUpdatePlayerVideo(ctx context.Context, conn *websocket.Conn, input UpdatePlayerVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	updatePlayerVideoResp, err := c.roomService.UpdatePlayerVideo(ctx, &service.UpdatePlayerVideoParams{
		SenderConn:      conn,
		PlaylistVersion: input.PlaylistVersion,
		PlayerVersion:   input.PlayerVersion,
		VideoId:         input.VideoId,
		UpdatedAt:       input.UpdatedAt,
		SenderId:        memberId,
		RoomId:          roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to update player video: %w", err)
	}

	switch {
	case updatePlayerVideoResp.PlayerVersionMismatchResponse != nil:
		if err := c.broadcastPlayerStateUpdated(ctx, updatePlayerVideoResp.Conns, &updatePlayerVideoResp.PlayerVersionMismatchResponse.Player); err != nil {
			return fmt.Errorf("failed to broadcast player state updated: %w", err)
		}
	case updatePlayerVideoResp.PlayerVideoUpdatedResponse != nil:
		if err := c.broadcastPlayerVideoUpdated(ctx,
			updatePlayerVideoResp.Conns,
			&updatePlayerVideoResp.PlayerVideoUpdatedResponse.Player,
			&updatePlayerVideoResp.PlayerVideoUpdatedResponse.Playlist,
			updatePlayerVideoResp.PlayerVideoUpdatedResponse.Members,
		); err != nil {
			return fmt.Errorf("failed to broadcast player updated: %w", err)
		}
	}

	return nil
}

type AddVideoInput struct {
	VideoUrl        string `json:"video_url"`
	UpdatedAt       int    `json:"updated_at"`
	PlaylsitVersion int    `json:"playlist_version"`
	PlayerVersion   int    `json:"player_version"`
}

func (c controller) handleAddVideo(ctx context.Context, conn *websocket.Conn, input AddVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	addVideoResponse, err := c.roomService.AddVideo(ctx, &service.AddVideoParams{
		SenderConn:      conn,
		PlaylistVersion: input.PlaylsitVersion,
		PlayerVersion:   input.PlayerVersion,
		SenderId:        memberId,
		RoomId:          roomId,
		VideoUrl:        input.VideoUrl,
		UpdatedAt:       input.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to add video: %w", err)
	}

	switch {
	case addVideoResponse.PlaylistVersionMismatchResponse != nil:
		// todo: replace with some other response
		if err := c.broadcastPlaylistReordered(ctx, addVideoResponse.Conns, &addVideoResponse.PlaylistVersionMismatchResponse.Playlist); err != nil {
			return fmt.Errorf("failed to broadcast playlist reordered: %w", err)
		}

	case addVideoResponse.PlayerVersionMismatchResponse != nil:
		if err := c.broadcastPlayerStateUpdated(ctx, addVideoResponse.Conns, &addVideoResponse.PlayerVideoUpdatedResponse.Player); err != nil {
			return fmt.Errorf("failed to broadcast player state updated: %w", err)
		}

	case addVideoResponse.VideoAddedResponse != nil:
		if err := c.broadcast(ctx, addVideoResponse.Conns, &Output{
			Type: "VIDEO_ADDED",
			Payload: map[string]any{
				"added_video": addVideoResponse.VideoAddedResponse.AddedVideo,
				"playlist":    addVideoResponse.VideoAddedResponse.Playlist,
			},
		}); err != nil {
			return fmt.Errorf("failed to broadcast video added: %w", err)
		}

	case addVideoResponse.PlayerVideoUpdatedResponse != nil:
		if err := c.broadcastPlayerVideoUpdated(ctx,
			addVideoResponse.Conns,
			&addVideoResponse.PlayerVideoUpdatedResponse.Player,
			&addVideoResponse.PlayerVideoUpdatedResponse.Playlist,
			addVideoResponse.PlayerVideoUpdatedResponse.Members,
		); err != nil {
			return fmt.Errorf("failed to broadcast player updated: %w", err)
		}

	}
	return nil
}

type EndVideoInput struct {
	PlayerVersion int `json:"player_version"`
}

func (c controller) handleEndVideo(ctx context.Context, conn *websocket.Conn, input EndVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	endVideoResponse, err := c.roomService.EndVideo(ctx, &service.EndVideoParams{
		SenderConn:    conn,
		PlayerVersion: input.PlayerVersion,
		SenderId:      memberId,
		RoomId:        roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to end video: %w", err)
	}

	switch {
	case endVideoResponse.PlayerVersionMismatchResponse != nil:
		if err := c.broadcastPlayerStateUpdated(ctx, endVideoResponse.Conns, &endVideoResponse.PlayerVersionMismatchResponse.Player); err != nil {
			return fmt.Errorf("failed to broadcast player state updated: %w", err)
		}
	case endVideoResponse.PlayerStateUpdatedResponse != nil:
		if err := c.broadcastPlayerStateUpdated(ctx, endVideoResponse.Conns, &endVideoResponse.PlayerStateUpdatedResponse.Player); err != nil {
			return fmt.Errorf("failed to broadcast player state updated: %w", err)
		}
	case endVideoResponse.PlayerVideoUpdatedResponse != nil:
		if err := c.broadcastPlayerVideoUpdated(ctx,
			endVideoResponse.Conns,
			&endVideoResponse.PlayerVideoUpdatedResponse.Player,
			&endVideoResponse.PlayerVideoUpdatedResponse.Playlist,
			endVideoResponse.PlayerVideoUpdatedResponse.Members,
		); err != nil {
			return fmt.Errorf("failed to broadcast player updated: %w", err)
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
	VideoId         int `json:"video_id"`
	PlaylistVersion int `json:"playlist_version"`
}

func (c controller) handleRemoveVideo(ctx context.Context, conn *websocket.Conn, input RemoveVideoInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	removeVideoResponse, err := c.roomService.RemoveVideo(ctx, &service.RemoveVideoParams{
		SenderConn:      conn,
		PlaylistVersion: input.PlaylistVersion,
		VideoId:         input.VideoId,
		SenderId:        memberId,
		RoomId:          roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to remove video: %w", err)
	}

	switch {
	case removeVideoResponse.PlaylistVersionMismatchResponse != nil:
		// todo: replace with some other response
		if err := c.broadcastPlaylistReordered(ctx, removeVideoResponse.Conns, &removeVideoResponse.PlaylistVersionMismatchResponse.Playlist); err != nil {
			return fmt.Errorf("failed to broadcast playlist reordered: %w", err)
		}
	case removeVideoResponse.VideoRemovedResponse != nil:
		if err := c.broadcast(ctx, removeVideoResponse.Conns, &Output{
			Type: "VIDEO_REMOVED",
			Payload: map[string]any{
				"removed_video_id": input.VideoId,
				"playlist":         removeVideoResponse.VideoRemovedResponse.Playlist,
			},
		}); err != nil {
			return fmt.Errorf("failed to broadcast video removed: %w", err)
		}
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
	VideoIds        []int `json:"video_ids"`
	PlaylistVersion int   `json:"playlist_version"`
}

func (c controller) handleReorderPlaylist(ctx context.Context, conn *websocket.Conn, input ReorderPlaylistInput) error {
	roomId := c.getRoomIdFromCtx(ctx)
	memberId := c.getMemberIdFromCtx(ctx)

	reorderVideoResponse, err := c.roomService.ReorderPlaylist(ctx, &service.ReorderPlaylistParams{
		SenderConn:      conn,
		PlaylistVersion: input.PlaylistVersion,
		VideoIds:        input.VideoIds,
		SenderId:        memberId,
		RoomId:          roomId,
	})
	if err != nil {
		return fmt.Errorf("failed to reorder playlist: %w", err)
	}

	switch {
	case reorderVideoResponse.PlaylistVersionMismatchResponse != nil:
		// todo: replace with some other response
		if err := c.broadcastPlaylistReordered(ctx, reorderVideoResponse.Conns, &reorderVideoResponse.PlaylistVersionMismatchResponse.Playlist); err != nil {
			return fmt.Errorf("failed to broadcast playlist reordered: %w", err)
		}
	case reorderVideoResponse.PlaylistReorderedResponse != nil:
		if err := c.broadcastPlaylistReordered(ctx, reorderVideoResponse.Conns, &reorderVideoResponse.PlaylistReorderedResponse.Playlist); err != nil {
			return fmt.Errorf("failed to broadcast playlist reordered: %w", err)
		}
	}

	return nil
}
