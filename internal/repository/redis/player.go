package redis

import (
	"context"
	"time"

	"github.com/sharetube/server/internal/repository"
)

func (r Repo) SetPlayer(ctx context.Context, params *repository.SetPlayerParams) error {
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
