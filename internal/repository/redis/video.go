package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository"
)

const videoPrefix = "video"

func (r Repo) getPlaylistKey(roomID string) string {
	return "room" + ":" + roomID + ":" + "playlist"
}

func (r Repo) getPlaylistVersionKey(roomID string) string {
	return "room" + ":" + roomID + ":" + "playlist-version"
}

func (r Repo) GetPlaylistLength(ctx context.Context, roomID string) (int, error) {
	playlistKey := r.getPlaylistKey(roomID)
	res, err := r.rc.ZCard(ctx, playlistKey).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	return int(res), err
}

func (r Repo) SetVideo(ctx context.Context, params *repository.SetVideoParams) error {
	pipe := r.rc.TxPipeline()

	video := repository.Video{
		URL:       params.URL,
		AddedByID: params.AddedByID,
		RoomID:    params.RoomID,
	}

	videoKey := videoPrefix + ":" + params.VideoID
	r.hSetIfNotExists(ctx, pipe, videoKey, video)
	pipe.Expire(ctx, videoKey, 10*time.Minute)

	playlistKey := r.getPlaylistKey(params.RoomID)

	r.addWithIncrement(ctx, pipe, playlistKey, params.VideoID)
	pipe.Expire(ctx, playlistKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func (r Repo) GetVideo(ctx context.Context, videoID string) (repository.Video, error) {
	videoKey := videoPrefix + ":" + videoID
	video := repository.Video{}
	if err := r.rc.HGetAll(ctx, videoKey).Scan(&video); err != nil {
		return repository.Video{}, err
	}

	if video.URL == "" {
		return repository.Video{}, errors.New("video not found")
	}

	return video, nil
}

func (r Repo) GetPlaylist(ctx context.Context, roomID string) ([]string, error) {
	playlistKey := r.getPlaylistKey(roomID)
	res, err := r.rc.ZRangeByScore(ctx, playlistKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, err
	}

	return res, nil
}
