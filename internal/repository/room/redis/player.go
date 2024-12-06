package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getPlayerKey(roomID string) string {
	return "room:" + roomID + ":player"
}

func (r repo) SetPlayer(ctx context.Context, params *room.SetPlayerParams) error {
	funcName := "room.redis.SetPlayer"
	slog.DebugContext(ctx, funcName, "params", params)
	pipe := r.rc.TxPipeline()

	player := room.Player{
		VideoURL:     params.CurrentVideoURL,
		IsPlaying:    params.IsPlaying,
		CurrentTime:  params.CurrentTime,
		PlaybackRate: params.PlaybackRate,
		UpdatedAt:    params.UpdatedAt,
	}
	playerKey := r.getPlayerKey(params.RoomID)
	hsetErr := r.hSetIfNotExists(ctx, pipe, playerKey, player)
	expErr := pipe.Expire(ctx, playerKey, 10*time.Minute).Err()

	_, err := pipe.Exec(ctx)
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return fmt.Errorf("failed to set player: %w", err)
	}

	if hsetErr != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return fmt.Errorf("failed to set player: %w", hsetErr)
	}

	if expErr != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return fmt.Errorf("failed to set player: %w", expErr)
	}

	return nil
}

func (r repo) GetPlayer(ctx context.Context, roomID string) (room.Player, error) {
	funcName := "room.redis.GetPlayer"
	slog.DebugContext(ctx, funcName, "roomID", roomID)
	var player room.Player
	if err := r.rc.HGetAll(ctx, r.getPlayerKey(roomID)).Scan(&player); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return room.Player{}, fmt.Errorf("failed to get player: %w", err)
	}

	slog.DebugContext(ctx, funcName, "player", player)
	return player, nil
}

func (r repo) RemovePlayer(ctx context.Context, roomID string) error {
	funcName := "room.redis.RemovePlayer"
	slog.DebugContext(ctx, funcName, "roomID", roomID)
	res, err := r.rc.Del(ctx, r.getPlayerKey(roomID)).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return fmt.Errorf("failed to remove player: %w", err)
	}

	if res == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrPlayerNotFound)
		return room.ErrPlayerNotFound
	}

	return nil
}

func (r repo) UpdatePlayerVideo(ctx context.Context, roomID string, videoURL string) error {
	funcName := "room.redis.UpdatePlayerVideo"
	slog.DebugContext(ctx, funcName, "roomID", roomID, "videoURL", videoURL)
	key := r.getPlayerKey(roomID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return fmt.Errorf("failed to update player video: %w", err)
	}

	if cmd.Val() == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrPlayerNotFound)
		return fmt.Errorf("failed to update player video: %w", room.ErrPlayerNotFound)
	}

	if err := r.rc.HSet(ctx, key, "video_url", videoURL).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return fmt.Errorf("failed to update player video: %w", err)
	}

	return nil
}

func (r repo) UpdatePlayerState(ctx context.Context, params *room.UpdatePlayerStateParams) error {
	funcName := "room.redis.UpdatePlayerState"
	slog.DebugContext(ctx, funcName, "params", params)
	key := r.getPlayerKey(params.RoomID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if cmd.Val() == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrPlayerNotFound)
		return room.ErrPlayerNotFound
	}

	if err := r.rc.HSet(ctx, key,
		"is_playing", params.IsPlaying,
		"current_time", params.CurrentTime,
		"playback_rate", params.PlaybackRate,
		"updated_at", params.UpdatedAt,
	).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}
