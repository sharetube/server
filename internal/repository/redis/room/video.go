package room

import (
	"context"
	"log/slog"
	"time"

	"github.com/sharetube/server/internal/repository"
)

func (r repo) getVideoKey(videoID string) string {
	return "video:" + videoID
}

func (r repo) getPlaylistKey(roomID string) string {
	return "room:" + roomID + ":playlist"
}

func (r repo) getPlaylistVersionKey(roomID string) string {
	return "room:" + roomID + ":playlist-version"
}

func (r repo) GetPlaylistLength(ctx context.Context, roomID string) (int, error) {
	funcName := "RedisRepo:GetPlaylistLength"
	slog.DebugContext(ctx, funcName, "roomID", roomID)
	playlistKey := r.getPlaylistKey(roomID)
	cmd := r.rc.ZCard(ctx, playlistKey)
	if err := cmd.Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return 0, err
	}

	res := int(cmd.Val())
	slog.DebugContext(ctx, funcName, "length", res)
	return res, nil
}

func (r repo) SetVideo(ctx context.Context, params *repository.SetVideoParams) error {
	funcName := "RedisRepo:SetVideo"
	slog.DebugContext(ctx, funcName, "params", params)
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
	videoKey := r.getVideoKey(params.VideoID)
	hsetErr := r.hSetIfNotExists(ctx, pipe, videoKey, video)
	expErr := pipe.Expire(ctx, videoKey, 10*time.Minute).Err()

	playlistKey := r.getPlaylistKey(params.RoomID)
	addErr := r.addWithIncrement(ctx, pipe, playlistKey, params.VideoID)
	exp2Err := pipe.Expire(ctx, playlistKey, 10*time.Minute).Err()

	_, err := pipe.Exec(ctx)
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if hsetErr != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return hsetErr
	}

	if expErr != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return expErr
	}

	if addErr != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return addErr
	}

	if exp2Err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return exp2Err
	}

	return nil
}

func (r repo) GetVideo(ctx context.Context, videoID string) (repository.Video, error) {
	funcName := "RedisRepo:GetVideo"
	slog.DebugContext(ctx, funcName, "videoID", videoID)
	video := repository.Video{}
	if err := r.rc.HGetAll(ctx, r.getVideoKey(videoID)).Scan(&video); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return repository.Video{}, err
	}

	if video.URL == "" {
		slog.DebugContext(ctx, funcName, "error", repository.ErrVideoNotFound)
		return repository.Video{}, repository.ErrVideoNotFound
	}

	slog.DebugContext(ctx, funcName, "video", video)
	return video, nil
}

func (r repo) GetVideosIDs(ctx context.Context, roomID string) ([]string, error) {
	funcName := "RedisRepo:GetVideosIDs"
	slog.DebugContext(ctx, funcName, "roomID", roomID)
	playlistKey := r.getPlaylistKey(roomID)
	videoIDs, err := r.rc.ZRange(ctx, playlistKey, 0, -1).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return nil, err
	}

	slog.DebugContext(ctx, funcName, "videoIDs", videoIDs)
	return videoIDs, nil
}

func (r repo) RemoveVideo(ctx context.Context, params *repository.RemoveVideoParams) error {
	funcName := "RedisRepo:RemoveVideo"
	slog.DebugContext(ctx, funcName, "params", params)
	res, err := r.rc.Del(ctx, r.getVideoKey(params.VideoID)).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if res == 0 {
		slog.DebugContext(ctx, funcName, "error", repository.ErrVideoNotFound)
		return repository.ErrVideoNotFound
	}

	if err := r.rc.ZRem(ctx, r.getPlaylistKey(params.RoomID), params.VideoID).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}
