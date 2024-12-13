package room

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

type UpdatePlayerStateParams struct {
	IsPlaying    bool
	CurrentTime  int
	PlaybackRate float64
	UpdatedAt    int
	SenderID     string
	RoomID       string
}

type UpdatePlayerStateResponse struct {
	PlayerState PlayerState
	Conns       []*websocket.Conn
}

func (s service) UpdatePlayerState(ctx context.Context, params *UpdatePlayerStateParams) (UpdatePlayerStateResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomID, params.SenderID); err != nil {
		return UpdatePlayerStateResponse{}, err
	}

	if err := s.roomRepo.UpdatePlayerState(ctx, &room.UpdatePlayerStateParams{
		IsPlaying:    params.IsPlaying,
		CurrentTime:  params.CurrentTime,
		PlaybackRate: params.PlaybackRate,
		UpdatedAt:    params.UpdatedAt,
		RoomID:       params.RoomID,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to update player state", "error", err)
		return UpdatePlayerStateResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
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
	VideoID   string
	UpdatedAt int
	SenderID  string
	RoomID    string
}

type UpdatePlayerVideoResponse struct {
	Player Player
	Conns  []*websocket.Conn
}

func (s service) UpdatePlayerVideo(ctx context.Context, params *UpdatePlayerVideoParams) (UpdatePlayerVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomID, params.SenderID); err != nil {
		return UpdatePlayerVideoResponse{}, err
	}

	if _, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		VideoID: params.VideoID,
		RoomID:  params.RoomID,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to get video", "error", err)
		return UpdatePlayerVideoResponse{}, err
	}

	updatePlayerParams := room.UpdatePlayerParams{
		VideoURL:     params.VideoID,
		IsPlaying:    s.getDefaultPlayerIsPlaying(),
		CurrentTime:  s.getDefaultPlayerCurrentTime(),
		PlaybackRate: s.getDefaultPlayerPlaybackRate(),
		UpdatedAt:    params.UpdatedAt,
		RoomID:       params.RoomID,
	}
	if err := s.roomRepo.UpdatePlayer(ctx, &updatePlayerParams); err != nil {
		s.logger.InfoContext(ctx, "failed to update player", "error", err)
		return UpdatePlayerVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return UpdatePlayerVideoResponse{}, err
	}

	return UpdatePlayerVideoResponse{
		Player: Player{
			IsPlaying:    updatePlayerParams.IsPlaying,
			CurrentTime:  updatePlayerParams.CurrentTime,
			PlaybackRate: updatePlayerParams.PlaybackRate,
			VideoURL:     params.VideoID,
			UpdatedAt:    params.UpdatedAt,
		},
		Conns: conns,
	}, nil
}
