package room

import (
	"context"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getPlaylist(ctx context.Context, roomID string) ([]Video, error) {
	videosIDs, err := s.roomRepo.GetVideoIDs(ctx, roomID)
	if err != nil {
		return []Video{}, err
	}

	playlist := make([]Video, 0, len(videosIDs))
	for _, videoID := range videosIDs {
		video, err := s.roomRepo.GetVideo(ctx, videoID)
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

type AddVideoParams struct {
	MemberID string
	RoomID   string
	VideoURL string
}

type AddVideoResponse struct {
	AddedVideo Video
	Conns      []*websocket.Conn
	Playlist   []Video
}

func (s service) AddVideo(ctx context.Context, params *AddVideoParams) (AddVideoResponse, error) {
	isAdmin, err := s.roomRepo.IsMemberAdmin(ctx, params.MemberID)
	if err != nil {
		return AddVideoResponse{}, err
	}
	if !isAdmin {
		return AddVideoResponse{}, ErrPermissionDenied
	}

	playlistLength, err := s.roomRepo.GetPlaylistLength(ctx, params.RoomID)
	if err != nil {
		return AddVideoResponse{}, err
	}

	if playlistLength >= s.playlistLimit {
		return AddVideoResponse{}, ErrPlaylistLimitReached
	}

	videoID := uuid.NewString()
	if err := s.roomRepo.SetVideo(ctx, &room.SetVideoParams{
		VideoID:   videoID,
		RoomID:    params.RoomID,
		URL:       params.VideoURL,
		AddedByID: params.MemberID,
	}); err != nil {
		return AddVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		return AddVideoResponse{}, err
	}

	playlist, err := s.getPlaylist(ctx, params.RoomID)
	if err != nil {
		return AddVideoResponse{}, err
	}

	return AddVideoResponse{
		AddedVideo: Video{
			ID:        videoID,
			URL:       params.VideoURL,
			AddedByID: params.MemberID,
		},
		Conns:    conns,
		Playlist: playlist,
	}, nil
}

type RemoveVideoParams struct {
	SenderID string
	VideoID  string
	RoomID   string
}

type RemoveVideoResponse struct {
	Conns          []*websocket.Conn
	Playlist       []Video
	RemovedVideoID string
}

func (s service) RemoveVideo(ctx context.Context, params *RemoveVideoParams) (RemoveVideoResponse, error) {
	isAdmin, err := s.roomRepo.IsMemberAdmin(ctx, params.SenderID)
	if err != nil {
		return RemoveVideoResponse{}, err
	}
	if !isAdmin {
		return RemoveVideoResponse{}, ErrPermissionDenied
	}

	if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoID: params.VideoID,
		RoomID:  params.RoomID,
	}); err != nil {
		return RemoveVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		return RemoveVideoResponse{}, err
	}

	playlist, err := s.getPlaylist(ctx, params.RoomID)
	if err != nil {
		return RemoveVideoResponse{}, err
	}

	return RemoveVideoResponse{
		Conns:          conns,
		Playlist:       playlist,
		RemovedVideoID: params.VideoID,
	}, nil
}
