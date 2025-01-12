package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getVideoKey(roomId string, videoId int) string {
	return fmt.Sprintf("room:%s:video:%d", roomId, videoId)
}

func (r repo) getLastIdKey(roomId string) string {
	return fmt.Sprintf("room:%s:video-last-id", roomId)
}

func (r repo) getPlaylistKey(roomId string) string {
	return fmt.Sprintf("room:%s:playlist", roomId)
}

func (r repo) getLastVideoKey(roomId string) string {
	return fmt.Sprintf("room:%s:last-video", roomId)
}

func (r repo) getPlaylistVersionKey(roomId string) string {
	return fmt.Sprintf("room:%s:playlist-version", roomId)
}

func (r repo) incrLastId(ctx context.Context, roomId string) (int, error) {
	lastIdKey := r.getLastIdKey(roomId)
	lastId, err := r.rc.Incr(ctx, lastIdKey).Result()
	if err != nil {
		return 0, err
	}

	r.rc.Expire(ctx, lastIdKey, r.maxExpireDuration)

	return int(lastId), nil
}

func (r repo) IncrPlaylistVersion(ctx context.Context, roomId string) (int, error) {
	pipe := r.rc.TxPipeline()
	playlistVersionKey := r.getPlaylistVersionKey(roomId)
	incrCmd := pipe.Incr(ctx, playlistVersionKey)
	pipe.Expire(ctx, playlistVersionKey, r.maxExpireDuration)

	if err := r.executePipe(ctx, pipe); err != nil {
		return 0, err
	}

	if incrCmd.Val() == 0 {
		return 0, room.ErrPlaylistVersionNotFound
	}

	return int(incrCmd.Val()), nil
}

func (r repo) GetPlaylistVersion(ctx context.Context, roomId string) (int, error) {
	playlistVersionKey := r.getPlaylistVersionKey(roomId)
	playlistVersion, err := r.rc.Get(ctx, playlistVersionKey).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}

	r.rc.Expire(ctx, playlistVersionKey, r.maxExpireDuration)
	return playlistVersion, nil
}

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

func (r repo) SetVideo(ctx context.Context, params *room.SetVideoParams) (int, error) {
	pipe := r.rc.TxPipeline()

	// todo: use pipeline
	videoId, err := r.incrLastId(ctx, params.RoomId)
	if err != nil {
		return 0, err
	}

	videoKey := r.getVideoKey(params.RoomId, videoId)
	pipe.SetNX(ctx, videoKey, params.Url, r.maxExpireDuration)
	pipe.Expire(ctx, videoKey, r.maxExpireDuration)

	if err := r.executePipe(ctx, pipe); err != nil {
		return 0, err
	}

	return videoId, nil
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

func (r repo) getVideoIds(ctx context.Context, roomId string) ([]int, error) {
	playlistKey := r.getPlaylistKey(roomId)
	// todo: try ScanSlice
	videoIds, err := r.rc.ZRangeByScore(ctx, playlistKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, err
	}

	r.rc.Expire(ctx, playlistKey, r.maxExpireDuration)

	var idInt int
	videoIdsInt := make([]int, 0, len(videoIds))
	for _, id := range videoIds {
		idInt, _ = strconv.Atoi(id)
		videoIdsInt = append(videoIdsInt, idInt)
	}

	return videoIdsInt, nil
}

func (r repo) GetVideoIds(ctx context.Context, roomId string) ([]int, error) {
	return r.getVideoIds(ctx, roomId)
}

func (r repo) ReorderList(ctx context.Context, params *room.ReorderListParams) error {
	videoIds, err := r.getVideoIds(ctx, params.RoomId)
	if err != nil {
		return err
	}

	// todo: check that all params.VideoIds are in videoIds

	if len(videoIds) != len(params.VideoIds) {
		return room.ErrInvalidVideoIds
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
	pipe := r.rc.TxPipeline()

	playlistKey := r.getPlaylistKey(params.RoomId)
	remCmd := pipe.ZRem(ctx, playlistKey, params.VideoId)
	pipe.Expire(ctx, playlistKey, r.maxExpireDuration)

	if err := r.executePipe(ctx, pipe); err != nil {
		return err
	}

	if remCmd.Val() == 0 {
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

	res, err = r.rc.ExpireAt(ctx, r.getLastIdKey(params.RoomId), params.ExpireAt).Result()
	if err != nil {
		return err
	}

	if !res {
		return room.ErrLastVideoIdNotFound
	}

	res, err = r.rc.ExpireAt(ctx, r.getPlaylistVersionKey(params.RoomId), params.ExpireAt).Result()
	if err != nil {
		return err
	}

	if !res {
		return room.ErrPlaylistVersionNotFound
	}

	return nil
}

func (r repo) GetLastVideoId(ctx context.Context, roomId string) (*int, error) {
	lastVideoKey := r.getLastVideoKey(roomId)
	lastVideoId, err := r.rc.Get(ctx, lastVideoKey).Int()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		return nil, err
	}

	r.rc.Expire(ctx, lastVideoKey, r.maxExpireDuration)

	return &lastVideoId, nil
}

func (r repo) SetLastVideo(ctx context.Context, params *room.SetLastVideoParams) error {
	pipe := r.rc.TxPipeline()

	lastVideoKey := r.getLastVideoKey(params.RoomId)
	pipe.Set(ctx, lastVideoKey, params.VideoId, r.maxExpireDuration)
	pipe.Expire(ctx, lastVideoKey, r.maxExpireDuration)

	return r.executePipe(ctx, pipe)
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
