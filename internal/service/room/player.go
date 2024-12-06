package room

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository"
)

type UpdatePlayerStateParams struct {
	IsPlaying    bool
	CurrentTime  float64
	PlaybackRate float64
	UpdatedAt    int64
	SenderID     string
	RoomID       string
}

type UpdatePlayerStateResponse struct {
	PlayerState PlayerState
	Conns       []*websocket.Conn
}

func (s service) UpdatePlayerState(ctx context.Context, params *UpdatePlayerStateParams) (UpdatePlayerStateResponse, error) {
	isAdmin, err := s.roomRepo.IsMemberAdmin(ctx, params.SenderID)
	if err != nil {
		return UpdatePlayerStateResponse{}, err
	}
	if !isAdmin {
		return UpdatePlayerStateResponse{}, ErrPermissionDenied
	}

	if err := s.roomRepo.UpdatePlayerState(ctx, &repository.UpdatePlayerStateParams{
		IsPlaying:    params.IsPlaying,
		CurrentTime:  params.CurrentTime,
		PlaybackRate: params.PlaybackRate,
		UpdatedAt:    params.UpdatedAt,
		RoomID:       params.RoomID,
	}); err != nil {
		return UpdatePlayerStateResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		return UpdatePlayerStateResponse{}, err
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
	VideoURL  string
	UpdatedAt int64
	SenderID  string
	RoomID    string
}

type UpdatePlayerVideoResponse struct {
	Player Player
	Conns  []*websocket.Conn
}

func (s service) UpdatePlayerVideo(ctx context.Context, params *UpdatePlayerVideoParams) (UpdatePlayerVideoResponse, error) {
	isAdmin, err := s.roomRepo.IsMemberAdmin(ctx, params.SenderID)
	if err != nil {
		return UpdatePlayerVideoResponse{}, err
	}
	if !isAdmin {
		return UpdatePlayerVideoResponse{}, ErrPermissionDenied
	}

	if err := s.roomRepo.UpdatePlayerVideo(ctx, params.RoomID, params.VideoURL); err != nil {
		return UpdatePlayerVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		return UpdatePlayerVideoResponse{}, err
	}

	return UpdatePlayerVideoResponse{
		Player: Player{
			VideoURL:  params.VideoURL,
			UpdatedAt: params.UpdatedAt,
		},
		Conns: conns,
	}, nil
}
