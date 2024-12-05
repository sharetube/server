package room

import (
	"context"
	"time"

	"github.com/sharetube/server/internal/repository"
)

var ()

func (r repo) getPlayerKey(roomID string) string {
	return "room" + ":" + roomID + ":player"
}

func (r repo) SetPlayer(ctx context.Context, params *repository.SetPlayerParams) error {
	pipe := r.rc.TxPipeline()

	player := repository.Player{
		VideoURL:     params.CurrentVideoURL,
		IsPlaying:    params.IsPlaying,
		CurrentTime:  params.CurrentTime,
		PlaybackRate: params.PlaybackRate,
		UpdatedAt:    params.UpdatedAt,
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
		return repository.ErrPlayerNotFound
	}

	return nil
}

func (r repo) UpdatePlayerVideo(ctx context.Context, roomID string, videoURL string) error {
	key := r.getPlayerKey(roomID)
	if r.rc.Exists(ctx, key).Val() == 0 {
		return repository.ErrPlayerNotFound
	}

	return r.rc.HSet(ctx, key, "video_url", videoURL).Err()
}

func (r repo) UpdatePlayerState(ctx context.Context, params *repository.UpdatePlayerStateParams) error {
	key := r.getPlayerKey(params.RoomID)
	if r.rc.Exists(ctx, key).Val() == 0 {
		return repository.ErrPlayerNotFound
	}

	return r.rc.HSet(ctx, key,
		"is_playing", params.IsPlaying,
		"current_time", params.CurrentTime,
		"playback_rate", params.PlaybackRate,
		"updated_at", params.UpdatedAt,
	).Err()
}
