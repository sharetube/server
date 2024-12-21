package room

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

type UpdatePlayerStateParams struct {
	IsPlaying    bool
	CurrentTime  int
	PlaybackRate float64
	UpdatedAt    int
	SenderId     string
	RoomId       string
}

type UpdatePlayerStateResponse struct {
	PlayerState PlayerState
	Conns       []*websocket.Conn
}

func (s service) UpdatePlayerState(ctx context.Context, params *UpdatePlayerStateParams) (UpdatePlayerStateResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to check if member is admin: %w", err)
	}

	if err := s.roomRepo.UpdatePlayerState(ctx, &room.UpdatePlayerStateParams{
		IsPlaying:    params.IsPlaying,
		CurrentTime:  params.CurrentTime,
		PlaybackRate: params.PlaybackRate,
		UpdatedAt:    params.UpdatedAt,
		RoomId:       params.RoomId,
	}); err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player state: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get conns by room id: %w", err)
	}

	return UpdatePlayerStateResponse{
		PlayerState: PlayerState{
			IsPlaying:    params.IsPlaying,
			CurrentTime:  params.CurrentTime,
			PlaybackRate: params.PlaybackRate,
			UpdatedAt:    params.UpdatedAt,
		},
		Conns: conns,
	}, nil
}

type UpdatePlayerVideoParams struct {
	VideoId   string
	UpdatedAt int
	SenderId  string
	RoomId    string
}

type UpdatePlayerVideoResponse struct {
	Player Player
	Conns  []*websocket.Conn
}

func (s service) UpdatePlayerVideo(ctx context.Context, params *UpdatePlayerVideoParams) (UpdatePlayerVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to check if member is admin: %w", err)
	}

	video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	})
	if err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to get video:%w", err)
	}

	updatePlayerParams := room.UpdatePlayerParams{
		VideoURL:     video.URL,
		IsPlaying:    s.getDefaultPlayerIsPlaying(),
		CurrentTime:  s.getDefaultPlayerCurrentTime(),
		PlaybackRate: s.getDefaultPlayerPlaybackRate(),
		UpdatedAt:    params.UpdatedAt,
		RoomId:       params.RoomId,
	}
	if err := s.roomRepo.UpdatePlayer(ctx, &updatePlayerParams); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update player: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to get conns by room id: %w", err)
	}

	return UpdatePlayerVideoResponse{
		Player: Player{
			IsPlaying:    updatePlayerParams.IsPlaying,
			CurrentTime:  updatePlayerParams.CurrentTime,
			PlaybackRate: updatePlayerParams.PlaybackRate,
			VideoURL:     updatePlayerParams.VideoURL,
			UpdatedAt:    updatePlayerParams.UpdatedAt,
		},
		Conns: conns,
	}, nil
}
