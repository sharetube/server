package room

import (
	"context"
	"errors"
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
	Player Player
	Conns  []*websocket.Conn
}

func (s service) UpdatePlayerState(ctx context.Context, params *UpdatePlayerStateParams) (UpdatePlayerStateResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return UpdatePlayerStateResponse{}, err
	}

	player, err := s.roomRepo.GetPlayer(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get player: %w", err)
	}

	updatePlayerStateParams := room.UpdatePlayerStateParams{
		IsPlaying:       params.IsPlaying,
		CurrentTime:     params.CurrentTime,
		PlaybackRate:    params.PlaybackRate,
		UpdatedAt:       params.UpdatedAt,
		RoomId:          params.RoomId,
		WaitingForReady: player.WaitingForReady,
	}

	if player.WaitingForReady && params.IsPlaying {
		updatePlayerStateParams.IsPlaying = false
	}

	if err := s.roomRepo.UpdatePlayerState(ctx, &updatePlayerStateParams); err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player state: %w", err)
	}

	videoId, err := s.roomRepo.GetPlayerVideoId(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get player video url: %w", err)
	}

	video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		VideoId: videoId,
		RoomId:  params.RoomId,
	})
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get video: %w", err)
	}

	memberIds, err := s.roomRepo.GetMemberIds(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerStateResponse{}, err
	}

	for i, memberId := range memberIds {
		if memberId == params.SenderId {
			memberIds = append(memberIds[:i], memberIds[i+1:]...)
			break
		}
	}

	conns, err := s.getConnsFromMemberIds(ctx, memberIds)
	if err != nil {
		return UpdatePlayerStateResponse{}, err
	}

	return UpdatePlayerStateResponse{
		Player: Player{
			VideoUrl:     video.Url,
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
	Members  []Member
	Conns    []*websocket.Conn
	Playlist Playlist
}

func (s service) UpdatePlayerVideo(ctx context.Context, params *UpdatePlayerVideoParams) (UpdatePlayerVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return UpdatePlayerVideoResponse{}, err
	}

	if err := s.roomRepo.RemoveVideoFromList(ctx, &room.RemoveVideoFromListParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	}); err != nil {
		if err != room.ErrVideoNotFound {
			return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to remove video from list: %w", err)
		}

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
		VideoId:         params.VideoId,
		IsPlaying:       s.getDefaultPlayerIsPlaying(),
		CurrentTime:     s.getDefaultPlayerCurrentTime(),
		WaitingForReady: true,
		PlaybackRate:    s.getDefaultPlayerPlaybackRate(),
		UpdatedAt:       params.UpdatedAt,
		RoomId:          params.RoomId,
	}
	if err := s.roomRepo.UpdatePlayer(ctx, &updatePlayerParams); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update player: %w", err)
	}

	playlist, err := s.getPlaylist(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerVideoResponse{}, err
	}

	memberIds, err := s.roomRepo.GetMemberIds(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to get member ids: %w", err)
	}

	for _, memberId := range memberIds {
		if err := s.roomRepo.UpdateMemberIsReady(ctx, params.RoomId, memberId, false); err != nil {
			return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update member is ready: %w", err)
		}
	}

	members, err := s.mapMembers(ctx, params.RoomId, memberIds)
	if err != nil {
		return UpdatePlayerVideoResponse{}, err
	}

	conns := make([]*websocket.Conn, 0, len(memberIds))
	for _, memberId := range memberIds {
		conn, err := s.connRepo.GetConn(memberId)
		if err != nil {
			return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to get conn: %w", err)
		}

		conns = append(conns, conn)
	}

	return UpdatePlayerVideoResponse{
		Members: members,
		Player: Player{
			IsPlaying:    updatePlayerParams.IsPlaying,
			CurrentTime:  updatePlayerParams.CurrentTime,
			PlaybackRate: updatePlayerParams.PlaybackRate,
			VideoUrl:     video.Url,
			UpdatedAt:    updatePlayerParams.UpdatedAt,
		},
		Playlist: playlist,
		Conns:    conns,
	}, nil
}
