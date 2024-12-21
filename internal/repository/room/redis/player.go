package redis

import (
	"context"
	"fmt"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getPlayerKey(roomId string) string {
	return "room:" + roomId + ":player"
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
	playerKey := r.getPlayerKey(params.RoomId)
	r.hSetIfNotExists(ctx, pipe, playerKey, player)
	pipe.Expire(ctx, playerKey, r.expireDuration)

	if err := r.executePipe(ctx, pipe); err != nil {
		return fmt.Errorf("failed to set player: %w", err)
	}

	return nil
}

func (r repo) IsPlayerExists(ctx context.Context, roomId string) (bool, error) {
	playerKey := r.getPlayerKey(roomId)
	res, err := r.rc.Exists(ctx, playerKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if player exists: %w", err)
	}

	r.rc.Expire(ctx, playerKey, r.expireDuration)

	exists := res > 0

	return exists, nil
}

func (r repo) GetPlayer(ctx context.Context, roomId string) (room.Player, error) {
	playerKey := r.getPlayerKey(roomId)
	var player room.Player
	if err := r.rc.HGetAll(ctx, playerKey).Scan(&player); err != nil {
		return room.Player{}, fmt.Errorf("failed to get player: %w", err)
	}

	r.rc.Expire(ctx, playerKey, r.expireDuration)

	return player, nil
}

func (r repo) GetPlayerVideoURL(ctx context.Context, roomId string) (string, error) {
	playerKey := r.getPlayerKey(roomId)
	videoURL, err := r.rc.HGet(ctx, playerKey, "video_url").Result()
	if err != nil {
		return "", fmt.Errorf("failed to get player: %w", err)
	}

	r.rc.Expire(ctx, playerKey, r.expireDuration)

	return videoURL, nil
}

func (r repo) RemovePlayer(ctx context.Context, roomId string) error {
	playerKey := r.getPlayerKey(roomId)
	res, err := r.rc.Del(ctx, playerKey).Result()
	if err != nil {
		return fmt.Errorf("failed to remove player: %w", err)
	}

	if res == 0 {
		return room.ErrPlayerNotFound
	}

	r.rc.Expire(ctx, playerKey, r.expireDuration)

	return nil
}

func (r repo) UpdatePlayer(ctx context.Context, params *room.UpdatePlayerParams) error {
	playerKey := r.getPlayerKey(params.RoomId)
	cmd := r.rc.Exists(ctx, playerKey)
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
	if err := r.rc.HSet(ctx, playerKey, player).Err(); err != nil {
		return err
	}

	r.rc.Expire(ctx, playerKey, r.expireDuration)

	return nil
}

func (r repo) UpdatePlayerState(ctx context.Context, params *room.UpdatePlayerStateParams) error {
	playerKey := r.getPlayerKey(params.RoomId)
	cmd := r.rc.Exists(ctx, playerKey)
	if err := cmd.Err(); err != nil {
		return err
	}

	if cmd.Val() == 0 {
		return room.ErrPlayerNotFound
	}

	if err := r.rc.HSet(ctx, playerKey,
		"current_time", params.CurrentTime,
		"is_playing", params.IsPlaying,
		"playback_rate", params.PlaybackRate,
		"updated_at", params.UpdatedAt,
	).Err(); err != nil {
		return err
	}

	r.rc.Expire(ctx, playerKey, r.expireDuration)

	return nil
}
