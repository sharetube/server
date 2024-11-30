package room

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository"
)

var (
	ErrVideoNotFound = errors.New("video not found")
)

const videoPrefix = "video"

func (r repo) getPlaylistKey(roomID string) string {
	return "room" + ":" + roomID + ":" + "playlist"
}

func (r repo) getPlaylistVersionKey(roomID string) string {
	return "room" + ":" + roomID + ":" + "playlist-version"
}

func (r repo) GetPlaylistLength(ctx context.Context, roomID string) (int, error) {
	playlistKey := r.getPlaylistKey(roomID)
	res, err := r.rc.ZCard(ctx, playlistKey).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	return int(res), err
}

func (r repo) SetVideo(ctx context.Context, params *repository.SetVideoParams) error {
	pipe := r.rc.TxPipeline()

	// playlistVersionKey := r.getPlaylistVersionKey(params.RoomID)
	// playlistVersion, err := r.rc.Get(ctx, playlistVersionKey).Int()
	// fmt.Printf("playlistVersion: %d\n", playlistVersion)
	// if err != nil {
	// 	fmt.Printf("set video error: %v\n", err)
	// 	return err
	// }

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

func (r repo) GetVideo(ctx context.Context, videoID string) (repository.Video, error) {
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

func (r repo) GetVideosIDs(ctx context.Context, roomID string) ([]string, error) {
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

func (r repo) RemoveVideo(ctx context.Context, params *repository.RemoveVideoParams) error {
	if err := r.rc.ZRem(ctx, r.getPlaylistKey(params.RoomID), params.VideoID).Err(); err != nil {
		slog.Info("failed to remove video from playlist", "err", err)
		return err
	}

	res, err := r.rc.Del(ctx, videoPrefix+":"+params.VideoID).Result()
	if err != nil {
		slog.Info("failed to delete video", "err", err)
		return err
	}

	if res == 0 {
		return ErrVideoNotFound
	}

	return nil
}
