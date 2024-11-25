package redis

import (
	"context"
)

type Player struct {
	CurrentVideoURL string  `redis:"current_video_url"`
	IsPlaying       bool    `redis:"is_playing"`
	CurrentTime     float64 `redis:"current_time"`
	PlaybackRate    float64 `redis:"playback_rate"`
	UpdatedAt       int64   `redis:"updated_at"`
}

type CreatePlayerParams struct {
	CurrentVideoURL string
	IsPlaying       bool
	CurrentTime     float64
	PlaybackRate    float64
	UpdatedAt       int64
	RoomID          string
}

func (r Repo) CreatePlayer(ctx context.Context, params *CreatePlayerParams) error {
	pipe := r.rc.TxPipeline()

	player := Player{
		CurrentVideoURL: params.CurrentVideoURL,
		IsPlaying:       params.IsPlaying,
		CurrentTime:     params.CurrentTime,
		PlaybackRate:    params.PlaybackRate,
		UpdatedAt:       params.UpdatedAt,
	}
	playerKey := "room" + ":" + params.RoomID + ":player"
	r.HSetIfNotExists(ctx, pipe, playerKey, player)
	// pipe.Expire(ctx, memberKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}
