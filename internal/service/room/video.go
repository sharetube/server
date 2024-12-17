package room

import (
	"context"

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
			Id:        videoId,
			URL:       video.URL,
			AddedById: video.AddedById,
		})
	}

	return playlist, nil
}

func (s service) getPreviousVideo(ctx context.Context, roomId string) (*Video, error) {
	previousVideoId, err := s.roomRepo.GetPreviousVideoId(ctx, roomId)
	if err != nil {
		switch err {
		case room.ErrNoPreviousVideo:
			return nil, nil
		default:
			return nil, err
		}
	}

	video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		RoomId:  roomId,
		VideoId: previousVideoId,
	})
	if err != nil {
		return nil, err
	}

	return &Video{
		Id:        previousVideoId,
		URL:       video.URL,
		AddedById: video.AddedById,
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
	Videos     []Video
}

func (s service) AddVideo(ctx context.Context, params *AddVideoParams) (AddVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return AddVideoResponse{}, err
	}

	videosLength, err := s.roomRepo.GetVideosLength(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get videos length", "error", err)
		return AddVideoResponse{}, err
	}

	if videosLength >= s.playlistLimit {
		s.logger.InfoContext(ctx, "playlist limit reached", "limit", s.playlistLimit)
		return AddVideoResponse{}, ErrPlaylistLimitReached
	}

	videoId := uuid.NewString()
	if err := s.roomRepo.SetVideo(ctx, &room.SetVideoParams{
		VideoId:   videoId,
		RoomId:    params.RoomId,
		URL:       params.VideoURL,
		AddedById: params.SenderId,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to set video", "error", err)
		return AddVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return AddVideoResponse{}, err
	}

	playlist, err := s.getVideos(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get videos", "error", err)
		return AddVideoResponse{}, err
	}

	return AddVideoResponse{
		AddedVideo: Video{
			Id:        videoId,
			URL:       params.VideoURL,
			AddedById: params.SenderId,
		},
		Conns:  conns,
		Videos: playlist,
	}, nil
}

type RemoveVideoParams struct {
	SenderId string
	VideoId  string
	RoomId   string
}

type RemoveVideoResponse struct {
	Conns          []*websocket.Conn
	Playlist       Playlist
	RemovedVideoId string
}

func (s service) RemoveVideo(ctx context.Context, params *RemoveVideoParams) (RemoveVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return RemoveVideoResponse{}, err
	}

	if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to remove video", "error", err)
		return RemoveVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return RemoveVideoResponse{}, err
	}

	videos, err := s.getVideos(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get videos", "error", err)
		return RemoveVideoResponse{}, err
	}

	previousVideo, err := s.getPreviousVideo(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get previous video", "error", err)
		return RemoveVideoResponse{}, err
	}

	return RemoveVideoResponse{
		Conns: conns,
		Playlist: Playlist{
			Videos:        videos,
			PreviousVideo: previousVideo,
		},
		RemovedVideoId: params.VideoId,
	}, nil
}
