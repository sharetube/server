package redis

import (
	"context"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getPlayerKey(roomId string) string {
	return "room:" + roomId + ":player"
}

func (r repo) SetPlayer(ctx context.Context, params *room.SetPlayerParams) error {
	pipe := r.rc.TxPipeline()

	player := room.Player{
		VideoId:      params.VideoId,
		IsPlaying:    params.IsPlaying,
		CurrentTime:  params.CurrentTime,
		PlaybackRate: params.PlaybackRate,
		UpdatedAt:    params.UpdatedAt,
	}
	playerKey := r.getPlayerKey(params.RoomId)
	r.hSetIfNotExists(ctx, pipe, playerKey, player)
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
	var player room.Player
	if err := r.rc.HGetAll(ctx, playerKey).Scan(&player); err != nil {
		return room.Player{}, err
	}

	r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	return player, nil
}

func (r repo) GetPlayerVideoId(ctx context.Context, roomId string) (string, error) {
	playerKey := r.getPlayerKey(roomId)
	videoId, err := r.rc.HGet(ctx, playerKey, "video_id").Result()
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
		VideoId:      params.VideoId,
		IsPlaying:    params.IsPlaying,
		CurrentTime:  params.CurrentTime,
		PlaybackRate: params.PlaybackRate,
		UpdatedAt:    params.UpdatedAt,
	}
	if err := r.rc.HSet(ctx, playerKey, player).Err(); err != nil {
		return err
	}

	r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

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

	r.rc.Expire(ctx, playerKey, r.maxExpireDuration)

	return nil
}
