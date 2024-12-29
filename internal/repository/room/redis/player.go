package redis

import (
	"context"

	"github.com/sharetube/server/internal/repository/room"
)

const (
	videoIdKey         = "video_id"
	isPlayingKey       = "is_playing"
	waitingForReadyKey = "waiting_for_ready"
	isEndedKey         = "is_ended"
	currentTimeKey     = "current_time"
	playbackRateKey    = "playback_rate"
	updatedAtKey       = "updated_at"
)

func (r repo) getPlayerKey(roomId string) string {
	return "room:" + roomId + ":player"
}

func (r repo) SetPlayer(ctx context.Context, params *room.SetPlayerParams) error {
	pipe := r.rc.TxPipeline()

	playerKey := r.getPlayerKey(params.RoomId)
	pipe.HSet(ctx, playerKey, map[string]any{
		videoIdKey:         params.VideoId,
		isPlayingKey:       params.IsPlaying,
		waitingForReadyKey: params.WaitingForReady,
		isEndedKey:         params.IsEnded,
		currentTimeKey:     params.CurrentTime,
		playbackRateKey:    params.PlaybackRate,
		updatedAtKey:       params.UpdatedAt,
	})
	pipe.Expire(ctx, playerKey, r.maxExpireDuration)

	if err := r.executePipe(ctx, pipe); err != nil {
		return err
	}

	return nil
}

func (r repo) IsPlayerExists(ctx context.Context, roomId string) (bool, error) {
	playerKey := r.getPlayerKey(roomId)
	res, err := r.rc.Exists(ctx, playerKey).Result()
	if err != nil {
		return false, err
	}

	r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	exists := res > 0

	return exists, nil
}

func (r repo) GetPlayer(ctx context.Context, roomId string) (room.Player, error) {
	playerKey := r.getPlayerKey(roomId)
	playerMap, err := r.rc.HGetAll(ctx, playerKey).Result()
	if err != nil {
		return room.Player{}, err
	}

	r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	return room.Player{
		VideoId:         playerMap[videoIdKey],
		IsPlaying:       r.fieldToBool(playerMap[isPlayingKey]),
		WaitingForReady: r.fieldToBool(playerMap[waitingForReadyKey]),
		IsEnded:         r.fieldToBool(playerMap[isEndedKey]),
		CurrentTime:     r.fieldToInt(playerMap[currentTimeKey]),
		PlaybackRate:    r.fieldToFload64(playerMap[playbackRateKey]),
		UpdatedAt:       r.fieldToInt(playerMap[updatedAtKey]),
	}, nil
}

func (r repo) GetPlayerVideoId(ctx context.Context, roomId string) (string, error) {
	playerKey := r.getPlayerKey(roomId)
	videoId, err := r.rc.HGet(ctx, playerKey, videoIdKey).Result()
	if err != nil {
		return "", err
	}

	r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	return videoId, nil
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

	r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

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

	r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	return nil
}

func (r repo) UpdatePlayerVideoId(ctx context.Context, roomId, videoId string) error {
	return r.updatePlayerValue(ctx, roomId, videoIdKey, videoId)
}

func (r repo) UpdatePlayerIsPlaying(ctx context.Context, roomId string, isPlaying bool) error {
	return r.updatePlayerValue(ctx, roomId, isPlayingKey, isPlaying)
}

func (r repo) UpdatePlayerWaitingForReady(ctx context.Context, roomId string, waitingForReady bool) error {
	return r.updatePlayerValue(ctx, roomId, waitingForReadyKey, waitingForReady)
}

func (r repo) UpdatePlayerIsEnded(ctx context.Context, roomId string, isEnded bool) error {
	return r.updatePlayerValue(ctx, roomId, isEndedKey, isEnded)
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
