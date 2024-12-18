package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getPlayerKey(roomId string) string {
	return "room:" + roomId + ":player"
}

func (r repo) SetPlayer(ctx context.Context, params *room.SetPlayerParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	pipe := r.rc.TxPipeline()

	player := room.Player{
		VideoURL:     params.CurrentVideoURL,
		IsPlaying:    params.IsPlaying,
		CurrentTime:  params.CurrentTime,
		PlaybackRate: params.PlaybackRate,
		UpdatedAt:    params.UpdatedAt,
	}
	playerKey := r.getPlayerKey(params.RoomId)
	r.hSetIfNotExists(ctx, pipe, playerKey, player)
	pipe.Expire(ctx, playerKey, 10*time.Minute)

	if err := r.executePipe(ctx, pipe); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return fmt.Errorf("failed to set player: %w", err)
	}

	return nil
}

func (r repo) IsPlayerExists(ctx context.Context, roomId string) (bool, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})
	res, err := r.rc.Exists(ctx, r.getPlayerKey(roomId)).Result()
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return false, fmt.Errorf("failed to check if player exists: %w", err)
	}

	exists := res > 0

	return exists, nil
}

func (r repo) GetPlayer(ctx context.Context, roomId string) (room.Player, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})
	var player room.Player
	if err := r.rc.HGetAll(ctx, r.getPlayerKey(roomId)).Scan(&player); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return room.Player{}, fmt.Errorf("failed to get player: %w", err)
	}

	return player, nil
}

func (r repo) GetPlayerVideoURL(ctx context.Context, roomId string) (string, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})
	videoURL, err := r.rc.HGet(ctx, r.getPlayerKey(roomId), "video_url").Result()
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return "", fmt.Errorf("failed to get player: %w", err)
	}

	return videoURL, nil
}

func (r repo) RemovePlayer(ctx context.Context, roomId string) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})
	res, err := r.rc.Del(ctx, r.getPlayerKey(roomId)).Result()
	if err != nil {
		return fmt.Errorf("failed to remove player: %w", err)
	}

	if res == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrPlayerNotFound)
		return room.ErrPlayerNotFound
	}

	return nil
}

func (r repo) UpdatePlayer(ctx context.Context, params *room.UpdatePlayerParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	key := r.getPlayerKey(params.RoomId)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrPlayerNotFound)
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
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) UpdatePlayerState(ctx context.Context, params *room.UpdatePlayerStateParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	key := r.getPlayerKey(params.RoomId)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrPlayerNotFound)
		return room.ErrPlayerNotFound
	}

	if err := r.rc.HSet(ctx, key,
		"current_time", params.CurrentTime,
		"is_playing", params.IsPlaying,
		"playback_rate", params.PlaybackRate,
		"updated_at", params.UpdatedAt,
	).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}
