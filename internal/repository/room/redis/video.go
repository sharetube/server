package redis

import (
	"context"
	"errors"

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

	videoKey := r.getVideoKey(params.RoomId, params.VideoId)
	pipe.SetNX(ctx, videoKey, params.Url, r.maxExpireDuration)
	pipe.Expire(ctx, videoKey, r.maxExpireDuration)

	return r.executePipe(ctx, pipe)
}

func (r repo) GetVideo(ctx context.Context, params *room.GetVideoParams) (room.Video, error) {
	videoKey := r.getVideoKey(params.RoomId, params.VideoId)

	videoUrl, err := r.rc.Get(ctx, videoKey).Result()
	if err != nil {
		return room.Video{}, err
	}

	if videoUrl == "" {
		return room.Video{}, room.ErrVideoNotFound
	}

	r.rc.Expire(ctx, videoKey, r.maxExpireDuration)

	return room.Video{
		Url: videoUrl,
	}, nil
}

func (r repo) getVideoIds(ctx context.Context, roomId string) ([]string, error) {
	playlistKey := r.getPlaylistKey(roomId)
	videoIds, err := r.rc.ZRangeByScore(ctx, playlistKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, err
	}

	r.rc.Expire(ctx, playlistKey, r.maxExpireDuration)

	return videoIds, nil
}

func (r repo) GetVideoIds(ctx context.Context, roomId string) ([]string, error) {
	return r.getVideoIds(ctx, roomId)
}

func (r repo) ReorderList(ctx context.Context, params *room.ReorderListParams) error {
	videoIds, err := r.getVideoIds(ctx, params.RoomId)
	if err != nil {
		return err
	}

	// todo: check that all params.VideoIds are in videoIds

	if len(videoIds) != len(params.VideoIds) {
		return errors.New("invalid video ids")
	}

	pipe := r.rc.TxPipeline()
	playlistKey := r.getPlaylistKey(params.RoomId)
	for i := 1; i <= len(videoIds); i++ {
		if videoIds[i-1] != params.VideoIds[i-1] {
			pipe.ZAdd(ctx, playlistKey, redis.Z{
				Score:  float64(i),
				Member: params.VideoIds[i-1],
			})
		}
	}

	pipe.Expire(ctx, playlistKey, r.maxExpireDuration)

	return r.executePipe(ctx, pipe)
}

func (r repo) RemoveVideoFromList(ctx context.Context, params *room.RemoveVideoFromListParams) error {
	res, err := r.rc.ZRem(ctx, r.getPlaylistKey(params.RoomId), params.VideoId).Result()
	if err != nil {
		return err
	}

	if res == 0 {
		return room.ErrVideoNotFound
	}

	return nil
}

func (r repo) RemoveVideo(ctx context.Context, params *room.RemoveVideoParams) error {
	return r.rc.Del(ctx, r.getVideoKey(params.RoomId, params.VideoId)).Err()
}

func (r repo) ExpireVideo(ctx context.Context, params *room.ExpireVideoParams) error {
	res, err := r.rc.ExpireAt(ctx, r.getVideoKey(params.RoomId, params.VideoId), params.ExpireAt).Result()
	if err != nil {
		return err
	}

	if !res {
		return room.ErrVideoNotFound
	}

	return nil
}

func (r repo) ExpirePlaylist(ctx context.Context, params *room.ExpirePlaylistParams) error {
	res, err := r.rc.ExpireAt(ctx, r.getPlaylistKey(params.RoomId), params.ExpireAt).Result()
	if err != nil {
		return err
	}

	if !res {
		return room.ErrPlaylistNotFound
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

func (r repo) ExpireLastVideo(ctx context.Context, params *room.ExpireLastVideoParams) error {
	res, err := r.rc.ExpireAt(ctx, r.getLastVideoKey(params.RoomId), params.ExpireAt).Result()
	if err != nil {
		return err
	}

	if !res {
		return room.ErrLastVideoNotFound
	}
	return nil
}
