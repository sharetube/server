package service

import (
	"context"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getVideos(ctx context.Context, roomId string) ([]Video, error) {
	videosIds, err := s.roomRepo.GetVideoIds(ctx, roomId)
	if err != nil {
		return []Video{}, fmt.Errorf("failed to get videos ids: %w", err)
	}

	playlist := make([]Video, 0, len(videosIds))
	for _, videoId := range videosIds {
		video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
			RoomId:  roomId,
			VideoId: videoId,
		})
		if err != nil {
			return []Video{}, fmt.Errorf("failed to get video: %w", err)
		}

		playlist = append(playlist, Video{
			Id:  videoId,
			Url: video.Url,
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
		return nil, fmt.Errorf("failed to get last video id: %w", err)
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
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	return &Video{
		Id:  *lastVideoId,
		Url: video.Url,
	}, nil
}

type AddVideoParams struct {
	SenderId string `json:"sender_id"`
	RoomId   string `json:"room_id"`
	VideoUrl string `json:"video_url"`
}

type AddVideoResponse struct {
	AddedVideo Video
	Conns      []*websocket.Conn
	Playlist   Playlist
}

func (s service) AddVideo(ctx context.Context, params *AddVideoParams) (AddVideoResponse, error) {
	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoUrl, VideoUrlRule...),
	); err != nil {
		return AddVideoResponse{}, err
	}

	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return AddVideoResponse{}, err
	}

	videosLength, err := s.roomRepo.GetVideosLength(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to get videos length: %w", err)
	}

	if videosLength >= s.playlistLimit {
		return AddVideoResponse{}, ErrPlaylistLimitReached
	}

	videoId, err := s.roomRepo.SetVideo(ctx, &room.SetVideoParams{
		RoomId: params.RoomId,
		Url:    params.VideoUrl,
	})
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to set video: %w", err)
	}

	if err := s.roomRepo.AddVideoToList(ctx, &room.AddVideoToListParams{
		RoomId:  params.RoomId,
		VideoId: videoId,
		Url:     params.VideoUrl,
	}); err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to add video to list: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, err
	}

	playlist, err := s.getPlaylist(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, err
	}

	return AddVideoResponse{
		AddedVideo: Video{
			Id:  videoId,
			Url: params.VideoUrl,
		},
		Conns:    conns,
		Playlist: playlist,
	}, nil
}

type RemoveVideoParams struct {
	SenderId string `json:"sender_id"`
	VideoId  string `json:"video_id"`
	RoomId   string `json:"room_id"`
}

type RemoveVideoResponse struct {
	Conns          []*websocket.Conn
	RemovedVideoId string
	Playlist       Playlist
}

func (s service) RemoveVideo(ctx context.Context, params *RemoveVideoParams) (RemoveVideoResponse, error) {
	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoId, VideoIdRule...),
	); err != nil {
		return RemoveVideoResponse{}, err
	}

	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return RemoveVideoResponse{}, err
	}

	if err := s.roomRepo.RemoveVideoFromList(ctx, &room.RemoveVideoFromListParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	}); err != nil {
		return RemoveVideoResponse{}, fmt.Errorf("failed to remove video from list: %w", err)
	}

	if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	}); err != nil {
		return RemoveVideoResponse{}, fmt.Errorf("failed to remove video: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return RemoveVideoResponse{}, err
	}

	playlist, err := s.getPlaylist(ctx, params.RoomId)
	if err != nil {
		return RemoveVideoResponse{}, err
	}

	return RemoveVideoResponse{
		Conns:          conns,
		Playlist:       playlist,
		RemovedVideoId: params.VideoId,
	}, nil
}

type ReorderPlaylistParams struct {
	VideoIds []string `json:"video_ids"`
	SenderId string   `json:"sender_id"`
	RoomId   string   `json:"room_id"`
}

type ReorderPlaylistResponse struct {
	Conns    []*websocket.Conn
	Playlist Playlist
}

func (s service) ReorderPlaylist(ctx context.Context, params *ReorderPlaylistParams) (ReorderPlaylistResponse, error) {
	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoIds, validation.Each(VideoIdRule...)),
	); err != nil {
		return ReorderPlaylistResponse{}, err
	}

	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return ReorderPlaylistResponse{}, err
	}

	if err := s.roomRepo.ReorderList(ctx, &room.ReorderListParams{
		VideoIds: params.VideoIds,
		RoomId:   params.RoomId,
	}); err != nil {
		return ReorderPlaylistResponse{}, fmt.Errorf("failed to reorder playlist: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return ReorderPlaylistResponse{}, err
	}

	playlist, err := s.getPlaylist(ctx, params.RoomId)
	if err != nil {
		return ReorderPlaylistResponse{}, err
	}

	return ReorderPlaylistResponse{
		Conns:    conns,
		Playlist: playlist,
	}, nil
}
