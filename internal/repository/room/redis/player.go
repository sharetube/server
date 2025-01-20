package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository/room"
)

const (
	isPlayingKey       = "is_playing"
	waitingForReadyKey = "waiting_for_ready"
	currentTimeKey     = "current_time"
	playbackRateKey    = "playback_rate"
	updatedAtKey       = "updated_at"
)

func (r repo) getPlayerKey(roomId string) string {
	return fmt.Sprintf("room:%s:player", roomId)
}

func (r repo) getVideoEndedKey(roomId string) string {
	return fmt.Sprintf("room:%s:video-ended", roomId)
}

func (r repo) getPlayerVersionKey(roomId string) string {
	return fmt.Sprintf("room:%s:player-version", roomId)
}

func (r repo) IncrPlayerVersion(ctx context.Context, roomId string) (int, error) {
	playerVersionKey := r.getPlayerVersionKey(roomId)
	playerVersion, err := r.rc.Incr(ctx, playerVersionKey).Result()
	if err != nil {
		return 0, err
	}

	return int(playerVersion), nil
}

func (r repo) GetPlayerVersion(ctx context.Context, roomId string) (int, error) {
	playerVersionKey := r.getPlayerVersionKey(roomId)
	playerVersion, err := r.rc.Get(ctx, playerVersionKey).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}

	return playerVersion, nil
}

func (r repo) SetVideoEnded(ctx context.Context, params *room.SetVideoEndedParams) error {
	// pipe := r.rc.TxPipeline()
	// pipe.Expire(ctx, videoEndedKey, r.maxExpireDuration)
	// return r.executePipe(ctx, pipe)

	return r.rc.Set(ctx, r.getVideoEndedKey(params.RoomId), params.VideoEnded, 0).Err()
}

func (r repo) GetVideoEnded(ctx context.Context, roomId string) (bool, error) {
	videoEndedKey := r.getVideoEndedKey(roomId)
	res, err := r.rc.Get(ctx, videoEndedKey).Bool()
	if err != nil {
		return false, err
	}

	return res, nil
}

func (r repo) ExpireVideoEnded(ctx context.Context, params *room.ExpireVideoEndedParams) error {
	res, err := r.rc.ExpireAt(ctx, r.getVideoEndedKey(params.RoomId), params.ExpireAt).Result()
	if err != nil {
		return err
	}

	if !res {
		return room.ErrVideoEndedNotFound
	}

	return nil
}

func (r repo) SetPlayer(ctx context.Context, params *room.SetPlayerParams) error {
	// pipe := r.rc.TxPipeline()
	// pipe.Expire(ctx, playerKey, r.maxExpireDuration)
	// return r.executePipe(ctx, pipe)

	playerKey := r.getPlayerKey(params.RoomId)
	return r.rc.HSet(ctx, playerKey, map[string]any{
		isPlayingKey:       params.IsPlaying,
		waitingForReadyKey: params.WaitingForReady,
		currentTimeKey:     params.CurrentTime,
		playbackRateKey:    params.PlaybackRate,
		updatedAtKey:       params.UpdatedAt,
	}).Err()
}

func (r repo) IsPlayerExists(ctx context.Context, roomId string) (bool, error) {
	playerKey := r.getPlayerKey(roomId)
	res, err := r.rc.Exists(ctx, playerKey).Result()
	if err != nil {
		return false, err
	}

	// r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	exists := res > 0

	return exists, nil
}

func (r repo) GetPlayer(ctx context.Context, roomId string) (room.Player, error) {
	playerKey := r.getPlayerKey(roomId)
	playerMap, err := r.rc.HGetAll(ctx, playerKey).Result()
	if err != nil {
		return room.Player{}, err
	}

	// r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	return room.Player{
		IsPlaying:       r.fieldToBool(playerMap[isPlayingKey]),
		WaitingForReady: r.fieldToBool(playerMap[waitingForReadyKey]),
		CurrentTime:     r.fieldToInt(playerMap[currentTimeKey]),
		PlaybackRate:    r.fieldToFload64(playerMap[playbackRateKey]),
		UpdatedAt:       r.fieldToInt(playerMap[updatedAtKey]),
	}, nil
}

func (r repo) RemovePlayer(ctx context.Context, roomId string) error {
	playerKey := r.getPlayerKey(roomId)
	res, err := r.rc.Del(ctx, playerKey).Result()
	if err != nil {
		return err
	}

	if res == 0 {
		return room.ErrPlayerNotFound
	}

	// r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	return nil
}

func (r repo) ExpirePlayer(ctx context.Context, params *room.ExpirePlayerParams) error {
	res, err := r.rc.ExpireAt(ctx, r.getPlayerKey(params.RoomId), params.ExpireAt).Result()
	if err != nil {
		return err
	}

	if !res {
		return room.ErrPlayerNotFound
	}

	return nil
}

func (r repo) updatePlayerValue(ctx context.Context, roomId string, key string, value any) error {
	playerKey := r.getPlayerKey(roomId)
	cmd := r.rc.Exists(ctx, playerKey)
	if err := cmd.Err(); err != nil {
		return err
	}

	if cmd.Val() == 0 {
		return room.ErrPlayerNotFound
	}

	if err := r.rc.HSet(ctx, playerKey, key, value).Err(); err != nil {
		return err
	}

	// r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	return nil
}

func (r repo) UpdatePlayerIsPlaying(ctx context.Context, roomId string, isPlaying bool) error {
	return r.updatePlayerValue(ctx, roomId, isPlayingKey, isPlaying)
}

func (r repo) UpdatePlayerWaitingForReady(ctx context.Context, roomId string, waitingForReady bool) error {
	return r.updatePlayerValue(ctx, roomId, waitingForReadyKey, waitingForReady)
}

func (r repo) UpdatePlayerCurrentTime(ctx context.Context, roomId string, currentTime int) error {
	return r.updatePlayerValue(ctx, roomId, currentTimeKey, currentTime)
}

func (r repo) UpdatePlayerPlaybackRate(ctx context.Context, roomId string, playbackRate float64) error {
	return r.updatePlayerValue(ctx, roomId, playbackRateKey, playbackRate)
}

func (r repo) UpdatePlayerUpdatedAt(ctx context.Context, roomId string, updatedAt int) error {
	return r.updatePlayerValue(ctx, roomId, updatedAtKey, updatedAt)
}
