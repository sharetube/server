package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/sharetube/server/internal/repository/room"
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
	funcName := "room.redis.GetPlaylistLength"
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

func (r repo) SetVideo(ctx context.Context, params *room.SetVideoParams) error {
	funcName := "room.redis.SetVideo"
	slog.DebugContext(ctx, funcName, "params", params)
	pipe := r.rc.TxPipeline()

	// playlistVersionKey := r.getPlaylistVersionKey(params.RoomID)
	// playlistVersion, err := r.rc.Get(ctx, playlistVersionKey).Int()
	// fmt.Printf("playlistVersion: %d\n", playlistVersion)
	// if err != nil {
	// 	fmt.Printf("set video error: %v\n", err)
	// 	return err
	// }

	video := room.Video{
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

func (r repo) GetVideo(ctx context.Context, videoID string) (room.Video, error) {
	funcName := "room.redis.GetVideo"
	slog.DebugContext(ctx, funcName, "videoID", videoID)
	video := room.Video{}
	if err := r.rc.HGetAll(ctx, r.getVideoKey(videoID)).Scan(&video); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return room.Video{}, err
	}

	if video.URL == "" {
		slog.DebugContext(ctx, funcName, "error", room.ErrVideoNotFound)
		return room.Video{}, room.ErrVideoNotFound
	}

	slog.DebugContext(ctx, funcName, "result", slog.Group("", "video", video))
	return video, nil
}

func (r repo) GetVideoIDs(ctx context.Context, roomID string) ([]string, error) {
	funcName := "room.redis.GetVideosIDs"
	slog.DebugContext(ctx, funcName, "roomID", roomID)
	playlistKey := r.getPlaylistKey(roomID)
	videoIDs, err := r.rc.ZRange(ctx, playlistKey, 0, -1).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return nil, err
	}

	slog.DebugContext(ctx, funcName, "result", map[string]any{"videoIDs": videoIDs})
	return videoIDs, nil
}

func (r repo) RemoveVideo(ctx context.Context, params *room.RemoveVideoParams) error {
	funcName := "room.redis.RemoveVideo"
	slog.DebugContext(ctx, funcName, "params", params)
	res, err := r.rc.Del(ctx, r.getVideoKey(params.VideoID)).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if res == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrVideoNotFound)
		return room.ErrVideoNotFound
	}

	if err := r.rc.ZRem(ctx, r.getPlaylistKey(params.RoomID), params.VideoID).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}
