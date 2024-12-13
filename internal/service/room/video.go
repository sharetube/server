package room

import (
	"context"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getVideos(ctx context.Context, roomID string) ([]Video, error) {
	videosIDs, err := s.roomRepo.GetVideoIDs(ctx, roomID)
	if err != nil {
		return []Video{}, err
	}

	playlist := make([]Video, 0, len(videosIDs))
	for _, videoID := range videosIDs {
		video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
			RoomID:  roomID,
			VideoID: videoID,
		})
		if err != nil {
			return []Video{}, err
		}

		playlist = append(playlist, Video{
			ID:        videoID,
			URL:       video.URL,
			AddedByID: video.AddedByID,
		})
	}

	return playlist, nil
}

func (s service) getPreviousVideo(ctx context.Context, roomID string) (*Video, error) {
	previousVideoID, err := s.roomRepo.GetPreviousVideoID(ctx, roomID)
	if err != nil {
		switch err {
		case room.ErrNoPreviousVideo:
			return nil, nil
		default:
			return nil, err
		}
	}

	video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		RoomID:  roomID,
		VideoID: previousVideoID,
	})
	if err != nil {
		return nil, err
	}

	return &Video{
		ID:        previousVideoID,
		URL:       video.URL,
		AddedByID: video.AddedByID,
	}, nil
}

type AddVideoParams struct {
	SenderID string
	RoomID   string
	VideoURL string
}

type AddVideoResponse struct {
	AddedVideo Video
	Conns      []*websocket.Conn
	Videos     []Video
}

func (s service) AddVideo(ctx context.Context, params *AddVideoParams) (AddVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomID, params.SenderID); err != nil {
		return AddVideoResponse{}, err
	}

	videosLength, err := s.roomRepo.GetVideosLength(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get videos length", "error", err)
		return AddVideoResponse{}, err
	}

	if videosLength >= s.playlistLimit {
		s.logger.InfoContext(ctx, "playlist limit reached", "limit", s.playlistLimit)
		return AddVideoResponse{}, ErrPlaylistLimitReached
	}

	videoID := uuid.NewString()
	if err := s.roomRepo.SetVideo(ctx, &room.SetVideoParams{
		VideoID:   videoID,
		RoomID:    params.RoomID,
		URL:       params.VideoURL,
		AddedByID: params.SenderID,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to set video", "error", err)
		return AddVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return AddVideoResponse{}, err
	}

	playlist, err := s.getVideos(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get videos", "error", err)
		return AddVideoResponse{}, err
	}

	return AddVideoResponse{
		AddedVideo: Video{
			ID:        videoID,
			URL:       params.VideoURL,
			AddedByID: params.SenderID,
		},
		Conns:  conns,
		Videos: playlist,
	}, nil
}

type RemoveVideoParams struct {
	SenderID string
	VideoID  string
	RoomID   string
}

type RemoveVideoResponse struct {
	Conns          []*websocket.Conn
	Playlist       Playlist
	RemovedVideoID string
}

func (s service) RemoveVideo(ctx context.Context, params *RemoveVideoParams) (RemoveVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomID, params.SenderID); err != nil {
		return RemoveVideoResponse{}, err
	}

	if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoID: params.VideoID,
		RoomID:  params.RoomID,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to remove video", "error", err)
		return RemoveVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return RemoveVideoResponse{}, err
	}

	videos, err := s.getVideos(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get videos", "error", err)
		return RemoveVideoResponse{}, err
	}

	previousVideo, err := s.getPreviousVideo(ctx, params.RoomID)
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
		RemovedVideoID: params.VideoID,
	}, nil
}
