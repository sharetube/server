package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getPlayerKey(roomID string) string {
	return "room:" + roomID + ":player"
}

func (r repo) SetPlayer(ctx context.Context, params *room.SetPlayerParams) error {
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
		return fmt.Errorf("failed to set player: %w", err)
	}

	if hsetErr != nil {
		return fmt.Errorf("failed to set player: %w", hsetErr)
	}

	if expErr != nil {
		return fmt.Errorf("failed to set player: %w", expErr)
	}

	return nil
}

func (r repo) IsPlayerExists(ctx context.Context, roomID string) (bool, error) {
	res, err := r.rc.Exists(ctx, r.getPlayerKey(roomID)).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if player exists: %w", err)
	}

	exists := res > 0

	return exists, nil
}

func (r repo) GetPlayer(ctx context.Context, roomID string) (room.Player, error) {
	var player room.Player
	if err := r.rc.HGetAll(ctx, r.getPlayerKey(roomID)).Scan(&player); err != nil {
		return room.Player{}, fmt.Errorf("failed to get player: %w", err)
	}

	return player, nil
}

func (r repo) GetPlayerVideoURL(ctx context.Context, roomID string) (string, error) {
	videoURL, err := r.rc.HGet(ctx, r.getPlayerKey(roomID), "video_url").Result()
	if err != nil {
		return "", fmt.Errorf("failed to get player: %w", err)
	}

	return videoURL, nil
}

func (r repo) RemovePlayer(ctx context.Context, roomID string) error {
	res, err := r.rc.Del(ctx, r.getPlayerKey(roomID)).Result()
	if err != nil {
		return fmt.Errorf("failed to remove player: %w", err)
	}

	if res == 0 {
		return room.ErrPlayerNotFound
	}

	return nil
}

func (r repo) UpdatePlayer(ctx context.Context, params *room.UpdatePlayerParams) error {
	key := r.getPlayerKey(params.RoomID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		return err
	}

	if cmd.Val() == 0 {
		return room.ErrPlayerNotFound
	}

	player := room.Player{
		VideoURL:     params.VideoURL,
		IsPlaying:    params.IsPlaying,
		CurrentTime:  params.CurrentTime,
		PlaybackRate: params.PlaybackRate,
		UpdatedAt:    params.UpdatedAt,
	}
	if err := r.rc.HSet(ctx, key, player).Err(); err != nil {
		return err
	}

	return nil
}

func (r repo) UpdatePlayerState(ctx context.Context, params *room.UpdatePlayerStateParams) error {
	key := r.getPlayerKey(params.RoomID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		return err
	}

	if cmd.Val() == 0 {
		return room.ErrPlayerNotFound
	}

	if err := r.rc.HSet(ctx, key,
		"current_time", params.CurrentTime,
		"is_playing", params.IsPlaying,
		"playback_rate", params.PlaybackRate,
		"updated_at", params.UpdatedAt,
	).Err(); err != nil {
		return err
	}

	return nil
}
