package room

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getVideos(ctx context.Context, roomId string) ([]Video, error) {
	videosIds, err := s.roomRepo.GetVideoIds(ctx, roomId)
	if err != nil {
		return []Video{}, err
	}

	playlist := make([]Video, 0, len(videosIds))
	for _, videoId := range videosIds {
		video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
			RoomId:  roomId,
			VideoId: videoId,
		})
		if err != nil {
			return []Video{}, err
		}

		playlist = append(playlist, Video{
			Id:  videoId,
			URL: video.URL,
		})
	}

	return playlist, nil
}

func (s service) getPlaylist(ctx context.Context, roomId string) (Playlist, error) {
	videos, err := s.getVideos(ctx, roomId)
	if err != nil {
		return Playlist{}, err
	}

	lastVideo, err := s.getLastVideo(ctx, roomId)
	if err != nil {
		return Playlist{}, err
	}

	return Playlist{
		Videos:    videos,
		LastVideo: lastVideo,
	}, nil
}

func (s service) getLastVideo(ctx context.Context, roomId string) (*Video, error) {
	lastVideoId, err := s.roomRepo.GetLastVideoId(ctx, roomId)
	if err != nil {
		return nil, err
	}

	if lastVideoId == nil {
		return nil, nil
	}

	video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		RoomId:  roomId,
		VideoId: *lastVideoId,
	})
	if err != nil {
		if err == room.ErrVideoNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &Video{
		Id:  *lastVideoId,
		URL: video.URL,
	}, nil
}

type AddVideoParams struct {
	SenderId string
	RoomId   string
	VideoURL string
}

type AddVideoResponse struct {
	AddedVideo Video
	Conns      []*websocket.Conn
	Playlist   Playlist
}

func (s service) AddVideo(ctx context.Context, params *AddVideoParams) (AddVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to check if member is admin: %w", err)
	}

	videosLength, err := s.roomRepo.GetVideosLength(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to get videos length: %w", err)
	}

	if videosLength >= s.playlistLimit {
		return AddVideoResponse{}, ErrPlaylistLimitReached
	}

	videoId := uuid.NewString()
	if err := s.roomRepo.SetVideo(ctx, &room.SetVideoParams{
		VideoId: videoId,
		RoomId:  params.RoomId,
		URL:     params.VideoURL,
	}); err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to set video: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to get conns by room id: %w", err)
	}

	playlist, err := s.getPlaylist(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to get playlist: %w", err)
	}

	return AddVideoResponse{
		AddedVideo: Video{
			Id:  videoId,
			URL: params.VideoURL,
		},
		Conns:    conns,
		Playlist: playlist,
	}, nil
}

type RemoveVideoParams struct {
	SenderId string
	VideoId  string
	RoomId   string
}

type RemoveVideoResponse struct {
	Conns          []*websocket.Conn
	RemovedVideoId string
	Playlist       Playlist
}

func (s service) RemoveVideo(ctx context.Context, params *RemoveVideoParams) (RemoveVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return RemoveVideoResponse{}, err
	}

	if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	}); err != nil {
		return RemoveVideoResponse{}, fmt.Errorf("failed to remove video: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return RemoveVideoResponse{}, fmt.Errorf("failed to get conns by room id: %w", err)
	}

	playlist, err := s.getPlaylist(ctx, params.RoomId)
	if err != nil {
		return RemoveVideoResponse{}, fmt.Errorf("failed to get playlist: %w", err)
	}

	return RemoveVideoResponse{
		Conns:          conns,
		Playlist:       playlist,
		RemovedVideoId: params.VideoId,
	}, nil
}
