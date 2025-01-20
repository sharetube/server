package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getDefaultPlayerPlaybackRate() float64 {
	return 1.0
}

func (s service) getDefaultPlayerCurrentTime() int {
	return 0
}

func (s service) getDefaultPlayerIsPlaying() bool {
	return false
}

func (s service) getDefaultPlayerWaitingForReady() bool {
	return false
}

func (s service) getDefaultMemberIsMuted() bool {
	return false
}

func (s service) getDefaultMemberIsReady() bool {
	return false
}

func (s service) checkIfMemberAdmin(ctx context.Context, roomId, memberId string) error {
	isAdmin, err := s.roomRepo.GetMemberIsAdmin(ctx, roomId, memberId)
	if err != nil {
		return fmt.Errorf("failed to get member is admin: %w", err)
	}

	if !isAdmin {
		return ErrPermissionDenied
	}

	return nil
}

type updatePlayerVideoResponse struct {
	Player   Player
	Members  []Member
	Playlist Playlist
	Conns    []*websocket.Conn
}

func (s service) updatePlayerVideo(ctx context.Context, roomId string, videoId int, updatedAt int) (*updatePlayerVideoResponse, error) {
	currentVideoId, err := s.roomRepo.GetCurrentVideoId(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get current video id: %w", err)
	}

	if currentVideoId == videoId {
		return nil, errors.New("video is already playing")
	}

	video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		VideoId: videoId,
		RoomId:  roomId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	lastVideoId, err := s.roomRepo.GetLastVideoId(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get last video id: %w", err)
	}

	if err := s.roomRepo.RemoveVideoFromList(ctx, &room.RemoveVideoFromListParams{
		VideoId: videoId,
		RoomId:  roomId,
	}); err != nil && err != room.ErrVideoNotFound {
		return nil, fmt.Errorf("failed to remove video from list: %w", err)
	}
	if lastVideoId != nil && *lastVideoId != videoId {
		if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
			VideoId: *lastVideoId,
			RoomId:  roomId,
		}); err != nil {
			return nil, fmt.Errorf("failed to remove video: %w", err)
		}
	}

	if err := s.roomRepo.SetVideoEnded(ctx, &room.SetVideoEndedParams{
		RoomId:     roomId,
		VideoEnded: false,
	}); err != nil {
		return nil, fmt.Errorf("failed to set video ended: %w", err)
	}

	// updating last video
	if err := s.roomRepo.SetLastVideo(ctx, &room.SetLastVideoParams{
		VideoId: currentVideoId,
		RoomId:  roomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to set last video: %w", err)
	}

	if err := s.roomRepo.SetCurrentVideoId(ctx, &room.SetCurrentVideoParams{
		VideoId: videoId,
		RoomId:  roomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to update current video id: %w", err)
	}

	if err := s.roomRepo.UpdatePlayerUpdatedAt(ctx, roomId, updatedAt); err != nil {
		return nil, fmt.Errorf("failed to update player updated at: %w", err)
	}

	isPlaying := s.getDefaultPlayerIsPlaying()
	if err := s.roomRepo.UpdatePlayerIsPlaying(ctx, roomId, isPlaying); err != nil {
		return nil, fmt.Errorf("failed to update player is playing: %w", err)
	}

	currentTime := s.getDefaultPlayerCurrentTime()
	if err := s.roomRepo.UpdatePlayerCurrentTime(ctx, roomId, currentTime); err != nil {
		return nil, fmt.Errorf("failed to update player current time: %w", err)
	}

	playbackRate := s.getDefaultPlayerPlaybackRate()
	if err := s.roomRepo.UpdatePlayerPlaybackRate(ctx, roomId, playbackRate); err != nil {
		return nil, fmt.Errorf("failed to update player playback rate: %w", err)
	}

	if err := s.roomRepo.UpdatePlayerWaitingForReady(ctx, roomId, true); err != nil {
		return nil, fmt.Errorf("failed to update player waiting for ready: %w", err)
	}

	playlistVersion, err := s.roomRepo.IncrPlaylistVersion(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to incr playlist version: %w", err)
	}

	videos, err := s.getVideos(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	lastVideo, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		VideoId: currentVideoId,
		RoomId:  roomId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get last video: %w", err)
	}

	memberIds, err := s.roomRepo.GetMemberIds(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get member ids: %w", err)
	}

	for _, memberId := range memberIds {
		if err := s.roomRepo.UpdateMemberIsReady(ctx, roomId, memberId, false); err != nil {
			return nil, fmt.Errorf("failed to update member is ready: %w", err)
		}
	}

	members, err := s.mapMembers(ctx, roomId, memberIds)
	if err != nil {
		return nil, fmt.Errorf("failed to map members: %w", err)
	}

	conns := make([]*websocket.Conn, 0, len(memberIds))
	for _, memberId := range memberIds {
		conn, err := s.connRepo.GetConn(memberId)
		if err != nil {
			return nil, fmt.Errorf("failed to get conn: %w", err)
		}

		conns = append(conns, conn)
	}

	playerVersion, err := s.roomRepo.IncrPlayerVersion(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to incr player version: %w", err)
	}

	isEnded, err := s.roomRepo.GetVideoEnded(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get video ended: %w", err)
	}

	return &updatePlayerVideoResponse{
		Player: Player{
			State: PlayerState{
				CurrentTime:  currentTime,
				IsPlaying:    isPlaying,
				PlaybackRate: playbackRate,
				UpdatedAt:    updatedAt,
			},
			IsEnded: isEnded,
			Version: playerVersion,
		},
		Members: members,
		Playlist: Playlist{
			Videos: videos,
			LastVideo: &Video{
				Id:           currentVideoId,
				Url:          lastVideo.Url,
				Title:        lastVideo.Title,
				AuthorName:   lastVideo.AuthorName,
				ThumbnailUrl: lastVideo.ThumbnailUrl,
			},
			CurrentVideo: Video{
				Id:           videoId,
				Url:          video.Url,
				Title:        video.Title,
				AuthorName:   video.AuthorName,
				ThumbnailUrl: video.ThumbnailUrl,
			},
			Version: playlistVersion,
		},
		Conns: conns,
	}, nil
}
