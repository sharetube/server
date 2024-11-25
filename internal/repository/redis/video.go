package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const videoPrefix = "video"

type Video struct {
	URL       string `redis:"url"`
	AddedByID string `redis:"added_by"`
	RoomID    string `redis:"room_id"`
}

type CreateVideoParams struct {
	ID        string
	RoomID    string
	URL       string
	AddedByID string
}

func (r Repo) CreateVideo(ctx context.Context, params *CreateVideoParams) error {
	pipe := r.rc.TxPipeline()

	video := Video{
		URL:       params.URL,
		AddedByID: params.AddedByID,
		RoomID:    params.RoomID,
	}
	videoKey := videoPrefix + ":" + params.ID
	r.HSetIfNotExists(ctx, pipe, videoKey, video)
	// pipe.Expire(ctx, memberKey, 10*time.Minute)

	memberListKey := "room" + ":" + params.RoomID + ":" + "playlist"
	lastScore := pipe.ZCard(ctx, memberListKey).Val()
	pipe.ZAdd(ctx, memberListKey, redis.Z{
		Score:  float64(lastScore + 1),
		Member: videoKey,
	})
	// pipe.Expire(ctx, memberListKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}
