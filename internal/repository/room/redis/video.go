package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getVideoKey(roomId, videoId string) string {
	return "room:" + roomId + ":video:" + videoId
}

func (r repo) getPlaylistKey(roomId string) string {
	return "room:" + roomId + ":playlist"
}

func (r repo) getLastVideoKey(roomId string) string {
	return "room:" + roomId + ":last-video"
}

// func (r repo) getPlaylistVersionKey(roomId string) string {
// 	return "room:" + roomId + ":playlist-version"
// }

func (r repo) GetVideosLength(ctx context.Context, roomId string) (int, error) {
	playlistKey := r.getPlaylistKey(roomId)
	cmd := r.rc.ZCard(ctx, playlistKey)
	if err := cmd.Err(); err != nil {
		return 0, err
	}

	r.rc.Expire(ctx, playlistKey, r.maxExpireDuration)

	videosLength := int(cmd.Val())
	return videosLength, nil
}

func (r repo) AddVideoToList(ctx context.Context, params *room.AddVideoToListParams) error {
	pipe := r.rc.TxPipeline()

	playlistKey := r.getPlaylistKey(params.RoomId)
	r.addWithIncrement(ctx, pipe, playlistKey, params.VideoId)
	pipe.Expire(ctx, playlistKey, r.maxExpireDuration)

	return r.executePipe(ctx, pipe)
}

func (r repo) SetVideo(ctx context.Context, params *room.SetVideoParams) error {
	pipe := r.rc.TxPipeline()

	// playlistVersionKey := r.getPlaylistVersionKey(params.RoomID)
	// playlistVersion, err := r.rc.Get(ctx, playlistVersionKey).Int()
	// fmt.Printf("playlistVersion: %d\n", playlistVersion)
	// if err != nil {
	// 	fmt.Printf("set video error: %v\n", err)
	// 	return err
	// }

	video := room.Video{
		URL: params.URL,
	}
	videoKey := r.getVideoKey(params.RoomId, params.VideoId)
	r.hSetIfNotExists(ctx, pipe, videoKey, video)
	pipe.Expire(ctx, videoKey, r.maxExpireDuration)

	return r.executePipe(ctx, pipe)
}

func (r repo) getVideo(ctx context.Context, params *room.GetVideoParams) (room.Video, error) {
	var video room.Video
	videoKey := r.getVideoKey(params.RoomId, params.VideoId)
	if err := r.rc.HGetAll(ctx, videoKey).Scan(&video); err != nil {
		return room.Video{}, err
	}

	if video.URL == "" {
		return room.Video{}, room.ErrVideoNotFound
	}

	r.rc.Expire(ctx, videoKey, r.maxExpireDuration)

	return video, nil
}

func (r repo) GetVideo(ctx context.Context, params *room.GetVideoParams) (room.Video, error) {
	video, err := r.getVideo(ctx, params)
	if err != nil {
		return room.Video{}, err
	}

	return video, nil
}

func (r repo) GetVideoIds(ctx context.Context, roomId string) ([]string, error) {
	playlistKey := r.getPlaylistKey(roomId)
	videoIds, err := r.rc.ZRange(ctx, playlistKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	r.rc.Expire(ctx, playlistKey, r.maxExpireDuration)

	return videoIds, nil
}

func (r repo) removeVideoFromList(ctx context.Context, roomId, videoId string) error {
	res, err := r.rc.ZRem(ctx, r.getPlaylistKey(roomId), videoId).Result()
	if err != nil {
		return err
	}

	if res == 0 {
		return room.ErrVideoNotFound
	}

	return nil
}

func (r repo) RemoveVideoFromList(ctx context.Context, params *room.RemoveVideoFromListParams) error {
	return r.removeVideoFromList(ctx, params.RoomId, params.VideoId)
}

func (r repo) RemoveVideo(ctx context.Context, params *room.RemoveVideoParams) error {
	return r.rc.Del(ctx, r.getVideoKey(params.RoomId, params.VideoId)).Err()
}

func (r repo) ExpireVideo(ctx context.Context, params *room.ExpireVideoParams) error {
	res, err := r.rc.Expire(ctx, r.getVideoKey(params.RoomId, params.VideoId), r.roomExpireDuration).Result()
	if err != nil {
		return fmt.Errorf("failed to expire video: %w", err)
	}

	fmt.Printf("ExpireVideo res: %v\n", res)

	if !res {
		return room.ErrVideoNotFound
	}

	return nil
}

func (r repo) GetLastVideoId(ctx context.Context, roomId string) (*string, error) {
	lastVideoKey := r.getLastVideoKey(roomId)
	lastVideoId, err := r.rc.Get(ctx, lastVideoKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if lastVideoId == "" {
		return nil, nil
	}

	r.rc.Expire(ctx, lastVideoKey, r.maxExpireDuration)

	return &lastVideoId, nil
}

func (r repo) SetLastVideo(ctx context.Context, params *room.SetLastVideoParams) error {
	lastVideoKey := r.getLastVideoKey(params.RoomId)

	if err := r.rc.Set(ctx, lastVideoKey, params.VideoId, r.maxExpireDuration).Err(); err != nil {
		return err
	}

	r.rc.Expire(ctx, lastVideoKey, r.maxExpireDuration)
	return nil
}

func (r repo) RemoveLastVideo(ctx context.Context, roomId string) error {
	return r.rc.Del(ctx, r.getLastVideoKey(roomId)).Err()
}
