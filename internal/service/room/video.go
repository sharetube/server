package room

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository"
)

func (s service) getPlaylist(ctx context.Context, roomID string) ([]Video, error) {
	videosIDs, err := s.roomRepo.GetVideosIDs(ctx, roomID)
	if err != nil {
		slog.Info("failed to get memberlist", "err", err)
		return []Video{}, err
	}

	playlist := make([]Video, 0, len(videosIDs))
	for _, videoID := range videosIDs {
		video, err := s.roomRepo.GetVideo(ctx, videoID)
		if err != nil {
			slog.Info("failed to get member", "err", err)
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
	// Conn     *websocket.Conn
	MemberID string
	VideoURL string
}

type AddVideoResponse struct {
	AddedVideo Video
	Conns      []*websocket.Conn
	Playlist   []Video
}

func (s service) AddVideo(ctx context.Context, params *AddVideoParams) (AddVideoResponse, error) {
	// memberID, err := s.connRepo.GetMemberID(params.Conn)
	// if err != nil {
	// 	slog.Info("failed to get member id", "err", err)
	// 	return AddVideoResponse{}, err
	// }

	isAdmin, err := s.roomRepo.IsMemberAdmin(ctx, params.MemberID)
	if err != nil {
		slog.Info("failed to check if member is admin", "err", err)
		return AddVideoResponse{}, err
	}
	if !isAdmin {
		return AddVideoResponse{}, ErrPermissionDenied
	}

	roomID, err := s.roomRepo.GetMemberRoomId(ctx, params.MemberID)
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
		AddedByID: params.MemberID,
	}); err != nil {
		slog.Info("failed to create video", "err", err)
		return AddVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, roomID)
	if err != nil {
		slog.Info("failed to get conns", "err", err)
		return AddVideoResponse{}, err
	}

	playlist, err := s.getPlaylist(ctx, roomID)
	if err != nil {
		slog.Info("failed to get playlist", "err", err)
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
