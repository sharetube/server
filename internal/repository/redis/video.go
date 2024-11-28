package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const videoPrefix = "video"

type Video struct {
	URL       string `redis:"url"`
	AddedByID string `redis:"added_by"`
	RoomID    string `redis:"room_id"`
}

type SetVideoParams struct {
	VideoID   string
	RoomID    string
	URL       string
	AddedByID string
}

func (r Repo) SetVideo(ctx context.Context, params *SetVideoParams) error {
	pipe := r.rc.TxPipeline()

	video := Video{
		URL:       params.URL,
		AddedByID: params.AddedByID,
		RoomID:    params.RoomID,
	}

	videoKey := videoPrefix + ":" + params.VideoID
	r.HSetIfNotExists(ctx, pipe, videoKey, video)
	pipe.Expire(ctx, videoKey, 10*time.Minute)

	memberListKey := "room" + ":" + params.RoomID + ":" + "playlist"
	lastScore := pipe.ZCard(ctx, memberListKey).Val()
	pipe.ZAdd(ctx, memberListKey, redis.Z{
		Score:  float64(lastScore + 1),
		Member: params.VideoID,
	})
	pipe.Expire(ctx, memberListKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}
