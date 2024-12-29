package service

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

	// todo: wrap in transaction
	if player.CurrentTime != params.CurrentTime {
		if err := s.roomRepo.UpdatePlayerCurrentTime(ctx, params.RoomId, params.CurrentTime); err != nil {
			return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player current time: %w", err)
		}
	}

	isPlaying := params.IsPlaying && !player.WaitingForReady
	if player.IsPlaying != isPlaying {
		if err := s.roomRepo.UpdatePlayerIsPlaying(ctx, params.RoomId, isPlaying); err != nil {
			return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player is playing: %w", err)
		}
	}

	if player.PlaybackRate != params.PlaybackRate {
		if err := s.roomRepo.UpdatePlayerPlaybackRate(ctx, params.RoomId, params.PlaybackRate); err != nil {
			return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player playback rate: %w", err)
		}
	}

	if err := s.roomRepo.UpdatePlayerUpdatedAt(ctx, params.RoomId, params.UpdatedAt); err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player updated at: %w", err)
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

	if err := s.roomRepo.UpdatePlayerVideoId(ctx, params.RoomId, params.VideoId); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update player video id: %w", err)
	}

	if err := s.roomRepo.UpdatePlayerUpdatedAt(ctx, params.RoomId, params.UpdatedAt); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update player updated at: %w", err)
	}

	isPlaying := s.getDefaultPlayerIsPlaying()
	if err := s.roomRepo.UpdatePlayerIsPlaying(ctx, params.RoomId, isPlaying); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update player is playing: %w", err)
	}

	currentTime := s.getDefaultPlayerCurrentTime()
	if err := s.roomRepo.UpdatePlayerCurrentTime(ctx, params.RoomId, currentTime); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update player current time: %w", err)
	}

	playbackRate := s.getDefaultPlayerPlaybackRate()
	if err := s.roomRepo.UpdatePlayerPlaybackRate(ctx, params.RoomId, playbackRate); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update player playback rate: %w", err)
	}

	waitingForReady := true
	if err := s.roomRepo.UpdatePlayerWaitingForReady(ctx, params.RoomId, waitingForReady); err != nil {
		return UpdatePlayerVideoResponse{}, fmt.Errorf("failed to update player waiting for ready: %w", err)
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
			IsPlaying:    isPlaying,
			CurrentTime:  currentTime,
			PlaybackRate: playbackRate,
			VideoUrl:     video.Url,
			UpdatedAt:    params.UpdatedAt,
		},
		Playlist: playlist,
		Conns:    conns,
	}, nil
}
