package room

import (
	"context"
	"time"

	"github.com/sharetube/server/internal/repository"
)

func (r repo) SetPlayer(ctx context.Context, params *repository.SetPlayerParams) error {
	pipe := r.rc.TxPipeline()

	player := repository.Player{
		CurrentVideoURL: params.CurrentVideoURL,
		IsPlaying:       params.IsPlaying,
		CurrentTime:     params.CurrentTime,
		PlaybackRate:    params.PlaybackRate,
		UpdatedAt:       params.UpdatedAt,
	}
	playerKey := "room" + ":" + params.RoomID + ":player"
	r.hSetIfNotExists(ctx, pipe, playerKey, player)
	pipe.Expire(ctx, playerKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func (r repo) GetPlayer(ctx context.Context, roomID string) (repository.Player, error) {
	playerKey := "room" + ":" + roomID + ":player"
	var player repository.Player
	if err := r.rc.HGetAll(ctx, playerKey).Scan(&player); err != nil {
		return repository.Player{}, err
	}

	return player, nil
}