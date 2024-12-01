package room

import (
	"context"
	"errors"
	"time"

	"github.com/sharetube/server/internal/repository"
)

var (
	ErrPlayerNotFound = errors.New("player not found")
)

func (r repo) getPlayerKey(roomID string) string {
	return "room" + ":" + roomID + ":player"
}

func (r repo) SetPlayer(ctx context.Context, params *repository.SetPlayerParams) error {
	pipe := r.rc.TxPipeline()

	player := repository.Player{
		CurrentVideoURL: params.CurrentVideoURL,
		IsPlaying:       params.IsPlaying,
		CurrentTime:     params.CurrentTime,
		PlaybackRate:    params.PlaybackRate,
		UpdatedAt:       params.UpdatedAt,
	}
	playerKey := r.getPlayerKey(params.RoomID)
	r.hSetIfNotExists(ctx, pipe, playerKey, player)
	pipe.Expire(ctx, playerKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func (r repo) GetPlayer(ctx context.Context, roomID string) (repository.Player, error) {
	var player repository.Player
	if err := r.rc.HGetAll(ctx, r.getPlayerKey(roomID)).Scan(&player); err != nil {
		return repository.Player{}, err
	}

	return player, nil
}

func (r repo) RemovePlayer(ctx context.Context, roomID string) error {
	res, err := r.rc.Del(ctx, r.getPlayerKey(roomID)).Result()
	if err != nil {
		return err
	}

	if res == 0 {
		return ErrPlayerNotFound
	}

	return nil
}
