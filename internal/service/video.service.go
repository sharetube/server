package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
	"github.com/sharetube/server/pkg/ytvideodata"
)

var (
	ErrPlayerVersionMismatch   = errors.New("player version mismatch")
	ErrPlaylistVersionMismatch = errors.New("playlist version mismatch")
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
			Id:           videoId,
			Url:          video.Url,
			Title:        video.Title,
			AuthorName:   video.AuthorName,
			ThumbnailUrl: video.ThumbnailUrl,
		})
	}

	return playlist, nil
}

func (s service) getPlaylist(ctx context.Context, roomId string) (*Playlist, error) {
	videos, err := s.getVideos(ctx, roomId)
	if err != nil {
		return nil, err
	}

	lastVideo, err := s.getLastVideo(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get last video: %w", err)
	}

	currentVideo, err := s.getCurrentVideo(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get current video: %w", err)
	}

	version, err := s.roomRepo.GetPlaylistVersion(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist version: %w", err)
	}

	return &Playlist{
		Videos:       videos,
		LastVideo:    lastVideo,
		CurrentVideo: *currentVideo,
		Version:      version,
	}, nil
}

func (s service) getPlayer(ctx context.Context, roomId string) (*Player, error) {
	player, err := s.roomRepo.GetPlayer(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	videoEnded, err := s.roomRepo.GetVideoEnded(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get video ended: %w", err)
	}

	playerVersion, err := s.roomRepo.GetPlayerVersion(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player version: %w", err)
	}

	return &Player{
		State: PlayerState{
			CurrentTime:  player.CurrentTime,
			IsPlaying:    player.IsPlaying,
			PlaybackRate: player.PlaybackRate,
			UpdatedAt:    player.UpdatedAt,
		},
		IsEnded: videoEnded,
		Version: playerVersion,
	}, nil
}

func (s service) getPlaylistWithIncrVersion(ctx context.Context, roomId string) (*Playlist, error) {
	playlistVersion, err := s.roomRepo.IncrPlaylistVersion(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to incr playlist version: %w", err)
	}

	videos, err := s.getVideos(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	lastVideo, err := s.getLastVideo(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get last video: %w", err)
	}

	currentVideo, err := s.getCurrentVideo(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get current video: %w", err)
	}

	return &Playlist{
		Videos:       videos,
		LastVideo:    lastVideo,
		CurrentVideo: *currentVideo,
		Version:      playlistVersion,
	}, nil
}

func (s service) getCurrentVideo(ctx context.Context, roomId string) (*Video, error) {
	currentVideoId, err := s.roomRepo.GetCurrentVideoId(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get current video id: %w", err)
	}

	video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		RoomId:  roomId,
		VideoId: currentVideoId,
	})
	if err != nil {
		if err == room.ErrVideoNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	return &Video{
		Id:           currentVideoId,
		Url:          video.Url,
		Title:        video.Title,
		AuthorName:   video.AuthorName,
		ThumbnailUrl: video.ThumbnailUrl,
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
		Id:           *lastVideoId,
		Url:          video.Url,
		Title:        video.Title,
		AuthorName:   video.AuthorName,
		ThumbnailUrl: video.ThumbnailUrl,
	}, nil
}

type AddVideoParams struct {
	SenderId        string `json:"sender_id"`
	RoomId          string `json:"room_id"`
	VideoUrl        string `json:"video_url"`
	UpdatedAt       int    `json:"updated_at"`
	PlaylistVersion int    `json:"playlist_version"`
	PlayerVersion   int    `json:"player_version"`
}

// todo: two optional responses
type AddVideoResponse struct {
	AddedVideo *Video
	Conns      []*websocket.Conn
	Playlist   Playlist
	Player     *Player
	Members    []Member
}

func (s service) AddVideo(ctx context.Context, params *AddVideoParams) (*AddVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return nil, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoUrl, VideoUrlRule...),
	); err != nil {
		return nil, err
	}

	playerVersion, err := s.roomRepo.GetPlayerVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player version: %w", err)
	}

	if params.PlayerVersion != playerVersion {
		return nil, ErrPlayerVersionMismatch
	}

	playlistVersion, err := s.roomRepo.GetPlaylistVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist version: %w", err)
	}

	if params.PlaylistVersion != playlistVersion {
		return nil, ErrPlaylistVersionMismatch
	}

	videoData, err := ytvideodata.Get(params.VideoUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get video data: %w", err)
	}

	videosLength, err := s.roomRepo.GetVideosLength(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos length: %w", err)
	}

	if videosLength >= s.playlistLimit {
		return nil, ErrPlaylistLimitReached
	}

	videoEnded, err := s.roomRepo.GetVideoEnded(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get video ended: %w", err)
	}

	videoId, err := s.roomRepo.SetVideo(ctx, &room.SetVideoParams{
		RoomId:       params.RoomId,
		Url:          params.VideoUrl,
		Title:        videoData.Title,
		ThumbnailUrl: videoData.ThumbnailUrl,
		AuthorName:   videoData.AuthorName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set video: %w", err)
	}

	if videosLength == 0 && videoEnded {
		updatePlayerVideoRes, err := s.updatePlayerVideo(ctx, params.RoomId, videoId, params.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to update player video: %w", err)
		}

		return &AddVideoResponse{
			Conns:    updatePlayerVideoRes.Conns,
			Playlist: updatePlayerVideoRes.Playlist,
			Player:   &updatePlayerVideoRes.Player,
			Members:  updatePlayerVideoRes.Members,
		}, nil
	}

	if err := s.roomRepo.AddVideoToList(ctx, &room.AddVideoToListParams{
		RoomId:  params.RoomId,
		VideoId: videoId,
	}); err != nil {
		return nil, fmt.Errorf("failed to add video to list: %w", err)
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	playlist, err := s.getPlaylistWithIncrVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist with incr version: %w", err)
	}

	return &AddVideoResponse{
		AddedVideo: &Video{
			Id:           videoId,
			Url:          params.VideoUrl,
			Title:        videoData.Title,
			ThumbnailUrl: videoData.ThumbnailUrl,
			AuthorName:   videoData.AuthorName,
		},
		Conns:    conns,
		Playlist: *playlist,
	}, nil
}

type EndVideoParams struct {
	SenderId      string `json:"sender_id"`
	RoomId        string `json:"room_id"`
	PlayerVersion int    `json:"player_version"`
}

type EndVideoResponse struct {
	Conns    []*websocket.Conn
	Playlist *Playlist
	Player   *Player
	Members  []Member
}

func (s service) EndVideo(ctx context.Context, params *EndVideoParams) (*EndVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return nil, err
	}

	playerVersion, err := s.roomRepo.GetPlayerVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player version: %w", err)
	}

	if playerVersion != params.PlayerVersion {
		return nil, errors.New("player version is not equal")
	}

	videoEnded, err := s.roomRepo.GetVideoEnded(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get video ended: %w", err)
	}

	if videoEnded {
		return nil, errors.New("ended already set")
	}

	videos, err := s.getVideos(ctx, params.RoomId)
	if err != nil {
		return nil, err
	}

	if len(videos) > 0 {
		updatePlayerVideoRes, err := s.updatePlayerVideo(ctx, params.RoomId, videos[0].Id, int(time.Now().UnixMicro()))
		if err != nil {
			return nil, fmt.Errorf("failed to update player video: %w", err)
		}

		return &EndVideoResponse{
			Conns:    updatePlayerVideoRes.Conns,
			Playlist: &updatePlayerVideoRes.Playlist,
			Player:   &updatePlayerVideoRes.Player,
			Members:  updatePlayerVideoRes.Members,
		}, nil
	}

	if err := s.roomRepo.SetVideoEnded(ctx, &room.SetVideoEndedParams{
		RoomId:     params.RoomId,
		VideoEnded: true,
	}); err != nil {
		return nil, fmt.Errorf("failed to set video ended: %w", err)
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	player, err := s.getPlayer(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	return &EndVideoResponse{
		Player: player,
		Conns:  conns,
	}, nil
}

type RemoveVideoParams struct {
	SenderId        string `json:"sender_id"`
	VideoId         int    `json:"video_id"`
	RoomId          string `json:"room_id"`
	PlaylistVersion int    `json:"playlist_version"`
}

type RemoveVideoResponse struct {
	Conns          []*websocket.Conn
	RemovedVideoId int
	Playlist       Playlist
}

func (s service) RemoveVideo(ctx context.Context, params *RemoveVideoParams) (*RemoveVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return nil, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoId, VideoIdRule...),
	); err != nil {
		return nil, err
	}

	playlistVersion, err := s.roomRepo.GetPlaylistVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist version: %w", err)
	}

	if params.PlaylistVersion != playlistVersion {
		return nil, ErrPlaylistVersionMismatch
	}

	if err := s.roomRepo.RemoveVideoFromList(ctx, &room.RemoveVideoFromListParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to remove video from list: %w", err)
	}

	if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
		VideoId: params.VideoId,
		RoomId:  params.RoomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to remove video: %w", err)
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	playlist, err := s.getPlaylistWithIncrVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	return &RemoveVideoResponse{
		Conns:          conns,
		RemovedVideoId: params.VideoId,
		Playlist:       *playlist,
	}, nil
}

type ReorderPlaylistParams struct {
	VideoIds        []int  `json:"video_ids"`
	SenderId        string `json:"sender_id"`
	RoomId          string `json:"room_id"`
	PlaylistVersion int    `json:"playlist_version"`
}

type ReorderPlaylistResponse struct {
	Conns    []*websocket.Conn
	Playlist Playlist
}

func (s service) ReorderPlaylist(ctx context.Context, params *ReorderPlaylistParams) (*ReorderPlaylistResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return nil, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoIds, validation.Each(VideoIdRule...)),
	); err != nil {
		return nil, err
	}

	playlistVersion, err := s.roomRepo.GetPlaylistVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist version: %w", err)
	}

	if params.PlaylistVersion != playlistVersion {
		return nil, ErrPlaylistVersionMismatch
	}

	if err := s.roomRepo.ReorderList(ctx, &room.ReorderListParams{
		VideoIds: params.VideoIds,
		RoomId:   params.RoomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to reorder playlist: %w", err)
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	playlist, err := s.getPlaylistWithIncrVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	return &ReorderPlaylistResponse{
		Conns:    conns,
		Playlist: *playlist,
	}, nil
}
