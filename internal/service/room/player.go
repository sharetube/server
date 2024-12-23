package room

import (
	"context"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

// func (s service) getPlayer(ctx context.Context, roomId string) (Player, error) {
// 	player, err := s.roomRepo.GetPlayer(ctx, roomId)
// 	if err != nil {
// 		return Player{}, fmt.Errorf("failed to get player: %w", err)
// 	}

type UpdatePlayerStateParams struct {
	IsPlaying    bool
	CurrentTime  int
	PlaybackRate float64
	UpdatedAt    int
	SenderId     string
	RoomId       string
}

type UpdatePlayerStateResponse struct {
	Player Player
	Conns  []*websocket.Conn
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

	videoURL, err := s.roomRepo.GetPlayerVideoURL(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get player video url: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get conns by room id: %w", err)
	}

	return UpdatePlayerStateResponse{
		Player: Player{
			VideoURL:     videoURL,
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
	Player   Player
	Conns    []*websocket.Conn
	Playlist Playlist
}

func (s service) UpdatePlayerVideo(ctx context.Context, params *UpdatePlayerVideoParams) (UpdatePlayerVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to check if member is admin: %w", err)
	}

	if err := s.roomRepo.RemoveVideoFromList(ctx, &room.RemoveVideoFromListParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	}); err != nil {
		if err != room.ErrVideoNotFound {
			return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to remove video from list: %w", err)
		}
		// maybe video is last

		lastVideoId, err := s.roomRepo.GetLastVideoId(ctx, params.RoomId)
		if err != nil {
			return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to get last video id: %w", err)
		}

		if lastVideoId != nil && *lastVideoId == params.VideoId {
			if err := s.roomRepo.SetLastVideo(ctx, &room.SetLastVideoParams{
				VideoId: params.VideoId,
				RoomId:  params.RoomId,
			}); err != nil {
				return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to set last video: %w", err)
			}
		} else {
			return UpdatePlayerVideoResponse{}, errors.New("video is not found")
		}
	} else {
		lastVideoId, err := s.roomRepo.GetLastVideoId(ctx, params.RoomId)
		if err != nil {
			return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to get last video id: %w", err)
		}

		if lastVideoId != nil {
			if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
				VideoId: *lastVideoId,
				RoomId:  params.RoomId,
			}); err != nil {
				return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to remove video: %w", err)
			}
		}
	}

	player, err := s.roomRepo.GetPlayer(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to get player: %w", err)
	}

	if err := s.roomRepo.SetLastVideo(ctx, &room.SetLastVideoParams{
		VideoId: player.VideoId,
		RoomId:  params.RoomId,
	}); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to set last video: %w", err)
	}

	video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	})
	if err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to get video:%w", err)
	}

	updatePlayerParams := room.UpdatePlayerParams{
		VideoId:      params.VideoId,
		IsPlaying:    s.getDefaultPlayerIsPlaying(),
		CurrentTime:  s.getDefaultPlayerCurrentTime(),
		PlaybackRate: s.getDefaultPlayerPlaybackRate(),
		UpdatedAt:    params.UpdatedAt,
		RoomId:       params.RoomId,
	}
	if err := s.roomRepo.UpdatePlayer(ctx, &updatePlayerParams); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update player: %w", err)
	}

	playlist, err := s.getPlaylist(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to get playlist: %w", err)
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
			VideoURL:     video.URL,
			UpdatedAt:    updatePlayerParams.UpdatedAt,
		},
		Playlist: playlist,
		Conns:    conns,
	}, nil
}
