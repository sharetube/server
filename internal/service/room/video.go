package room

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository"
)

type AddVideoParams struct {
	Conn     *websocket.Conn
	VideoURL string
}

type AddVideoResponse struct {
	AddedVideo Video
	Conns      []*websocket.Conn
	Playlist   []Video
}

func (s service) AddVideo(ctx context.Context, params *AddVideoParams) (AddVideoResponse, error) {
	memberID, err := s.connRepo.GetMemberID(params.Conn)
	if err != nil {
		slog.Info("failed to get member id", "err", err)
		return AddVideoResponse{}, err
	}

	isAdmin, err := s.roomRepo.IsMemberAdmin(ctx, memberID)
	if err != nil {
		slog.Info("failed to check if member is admin", "err", err)
		return AddVideoResponse{}, err
	}
	if !isAdmin {
		return AddVideoResponse{}, ErrPermissionDenied
	}

	roomID, err := s.roomRepo.GetMemberRoomId(ctx, memberID)
	if err != nil {
		slog.Info("failed to get room id", "err", err)
		return AddVideoResponse{}, err
	}

	playlistLength, err := s.roomRepo.GetPlaylistLength(ctx, roomID)
	if err != nil {
		slog.Info("failed to get playlist length", "err", err)
		return AddVideoResponse{}, err
	}

	if playlistLength >= s.playlistLimit {
		return AddVideoResponse{}, ErrPlaylistLimitReached
	}

	videoID := uuid.NewString()
	if err := s.roomRepo.SetVideo(ctx, &repository.SetVideoParams{
		VideoID:   videoID,
		RoomID:    roomID,
		URL:       params.VideoURL,
		AddedByID: memberID,
	}); err != nil {
		slog.Info("failed to create video", "err", err)
		return AddVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, roomID)
	if err != nil {
		slog.Info("failed to get conns", "err", err)
		return AddVideoResponse{}, err
	}

	playlistIDs, err := s.roomRepo.GetPlaylist(ctx, roomID)
	if err != nil {
		slog.Info("failed to get playlist", "err", err)
		return AddVideoResponse{}, err
	}

	playlist := make([]Video, 0, len(playlistIDs))
	for _, videoID := range playlistIDs {
		video, err := s.roomRepo.GetVideo(ctx, videoID)
		if err != nil {
			slog.Info("failed to get video", "err", err)
			return AddVideoResponse{}, err
		}

		playlist = append(playlist, Video{
			ID:        videoID,
			URL:       video.URL,
			AddedByID: video.AddedByID,
		})
	}

	return AddVideoResponse{
		AddedVideo: Video{
			ID:        videoID,
			URL:       params.VideoURL,
			AddedByID: memberID,
		},
		Conns:    conns,
		Playlist: playlist,
	}, nil
}
