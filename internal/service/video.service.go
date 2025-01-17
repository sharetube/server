package service

import (
	"context"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
	"github.com/sharetube/server/pkg/ytvideodata"
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

	if currentVideoId == nil {
		return nil, nil
	}

	video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		RoomId:  roomId,
		VideoId: *currentVideoId,
	})
	if err != nil {
		if err == room.ErrVideoNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	return &Video{
		Id:           *currentVideoId,
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
	SenderId  string `json:"sender_id"`
	RoomId    string `json:"room_id"`
	VideoUrl  string `json:"video_url"`
	UpdatedAt int    `json:"updated_at"`
}

// todo: two optional responses
type AddVideoResponse struct {
	AddedVideo *Video
	Conns      []*websocket.Conn
	Playlist   Playlist
	Player     *Player
	Members    []Member
}

func (s service) AddVideo(ctx context.Context, params *AddVideoParams) (AddVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return AddVideoResponse{}, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoUrl, VideoUrlRule...),
	); err != nil {
		return AddVideoResponse{}, err
	}

	videoData, err := ytvideodata.Get(params.VideoUrl)
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to get video data: %w", err)
	}

	videosLength, err := s.roomRepo.GetVideosLength(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to get videos length: %w", err)
	}

	if videosLength >= s.playlistLimit {
		return AddVideoResponse{}, ErrPlaylistLimitReached
	}

	player, err := s.roomRepo.GetPlayer(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to get player: %w", err)
	}

	videoId, err := s.roomRepo.SetVideo(ctx, &room.SetVideoParams{
		RoomId:       params.RoomId,
		Url:          params.VideoUrl,
		Title:        videoData.Title,
		ThumbnailUrl: videoData.ThumbnailUrl,
		AuthorName:   videoData.AuthorName,
	})
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to set video: %w", err)
	}

	if videosLength == 0 && player.IsEnded {
		// todo: move duplicated with UpdatePlayerVideo logic to helper
		lastVideoId, err := s.roomRepo.GetLastVideoId(ctx, params.RoomId)
		if err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to get last video id: %w", err)
		}

		if lastVideoId != nil {
			if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
				VideoId: *lastVideoId,
				RoomId:  params.RoomId,
			}); err != nil {
				return AddVideoResponse{}, fmt.Errorf("failed to remove video: %w", err)
			}
		}

		currentVideoId, err := s.roomRepo.GetCurrentVideoId(ctx, params.RoomId)
		if err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to get current video id: %w", err)
		}

		if err := s.roomRepo.SetLastVideo(ctx, &room.SetLastVideoParams{
			VideoId: *currentVideoId,
			RoomId:  params.RoomId,
		}); err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to set last video: %w", err)
		}

		if err := s.roomRepo.SetCurrentVideoId(ctx, &room.SetCurrentVideoParams{
			VideoId: videoId,
			RoomId:  params.RoomId,
		}); err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to update current video id: %w", err)
		}

		if err := s.roomRepo.UpdatePlayerUpdatedAt(ctx, params.RoomId, params.UpdatedAt); err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to update player updated at: %w", err)
		}

		isPlaying := s.getDefaultPlayerIsPlaying()
		if err := s.roomRepo.UpdatePlayerIsPlaying(ctx, params.RoomId, isPlaying); err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to update player is playing: %w", err)
		}

		isEnded := s.getDefaultPlayerIsEnded()
		if err := s.roomRepo.UpdatePlayerIsEnded(ctx, params.RoomId, isEnded); err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to update player is ended: %w", err)
		}

		currentTime := s.getDefaultPlayerCurrentTime()
		if err := s.roomRepo.UpdatePlayerCurrentTime(ctx, params.RoomId, currentTime); err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to update player current time: %w", err)
		}

		playbackRate := s.getDefaultPlayerPlaybackRate()
		if err := s.roomRepo.UpdatePlayerPlaybackRate(ctx, params.RoomId, playbackRate); err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to update player playback rate: %w", err)
		}

		if err := s.roomRepo.UpdatePlayerWaitingForReady(ctx, params.RoomId, true); err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to update player waiting for ready: %w", err)
		}

		memberIds, err := s.roomRepo.GetMemberIds(ctx, params.RoomId)
		if err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to get member ids: %w", err)
		}

		for _, memberId := range memberIds {
			if err := s.roomRepo.UpdateMemberIsReady(ctx, params.RoomId, memberId, false); err != nil {
				return AddVideoResponse{}, fmt.Errorf("failed to update member is ready: %w", err)
			}
		}

		members, err := s.mapMembers(ctx, params.RoomId, memberIds)
		if err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to map members: %w", err)
		}

		conns, err := s.getConns(ctx, params.RoomId)
		if err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to get conns: %w", err)
		}

		playlist, err := s.getPlaylistWithIncrVersion(ctx, params.RoomId)
		if err != nil {
			return AddVideoResponse{}, fmt.Errorf("failed to get playlist with incr version: %w", err)
		}

		return AddVideoResponse{
			Conns:    conns,
			Playlist: *playlist,
			Player: &Player{
				CurrentTime:  currentTime,
				IsPlaying:    isPlaying,
				IsEnded:      isEnded,
				PlaybackRate: playbackRate,
				UpdatedAt:    params.UpdatedAt,
			},
			Members: members,
		}, nil
	}

	if err := s.roomRepo.AddVideoToList(ctx, &room.AddVideoToListParams{
		RoomId:  params.RoomId,
		VideoId: videoId,
	}); err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to add video to list: %w", err)
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to get conns: %w", err)
	}

	playlist, err := s.getPlaylistWithIncrVersion(ctx, params.RoomId)
	if err != nil {
		return AddVideoResponse{}, fmt.Errorf("failed to get playlist with incr version: %w", err)
	}

	return AddVideoResponse{
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

type RemoveVideoParams struct {
	SenderId string `json:"sender_id"`
	VideoId  int    `json:"video_id"`
	RoomId   string `json:"room_id"`
}

type RemoveVideoResponse struct {
	Conns          []*websocket.Conn
	RemovedVideoId int
	Playlist       Playlist
}

func (s service) RemoveVideo(ctx context.Context, params *RemoveVideoParams) (RemoveVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return RemoveVideoResponse{}, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoId, VideoIdRule...),
	); err != nil {
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

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return RemoveVideoResponse{}, fmt.Errorf("failed to get conns: %w", err)
	}

	playlist, err := s.getPlaylistWithIncrVersion(ctx, params.RoomId)
	if err != nil {
		return RemoveVideoResponse{}, fmt.Errorf("failed to get playlist: %w", err)
	}

	return RemoveVideoResponse{
		Conns:          conns,
		RemovedVideoId: params.VideoId,
		Playlist:       *playlist,
	}, nil
}

type ReorderPlaylistParams struct {
	VideoIds []int  `json:"video_ids"`
	SenderId string `json:"sender_id"`
	RoomId   string `json:"room_id"`
}

type ReorderPlaylistResponse struct {
	Conns    []*websocket.Conn
	Playlist Playlist
}

func (s service) ReorderPlaylist(ctx context.Context, params *ReorderPlaylistParams) (ReorderPlaylistResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return ReorderPlaylistResponse{}, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoIds, validation.Each(VideoIdRule...)),
	); err != nil {
		return ReorderPlaylistResponse{}, err
	}

	if err := s.roomRepo.ReorderList(ctx, &room.ReorderListParams{
		VideoIds: params.VideoIds,
		RoomId:   params.RoomId,
	}); err != nil {
		return ReorderPlaylistResponse{}, fmt.Errorf("failed to reorder playlist: %w", err)
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return ReorderPlaylistResponse{}, fmt.Errorf("failed to get conns: %w", err)
	}

	playlist, err := s.getPlaylistWithIncrVersion(ctx, params.RoomId)
	if err != nil {
		return ReorderPlaylistResponse{}, fmt.Errorf("failed to get playlist: %w", err)
	}

	return ReorderPlaylistResponse{
		Conns:    conns,
		Playlist: *playlist,
	}, nil
}
