package redis

import (
	"context"
	"time"

	"github.com/sharetube/server/internal/repository"
)

const videoPrefix = "video"

func (r Repo) SetVideo(ctx context.Context, params *repository.SetVideoParams) error {
	pipe := r.rc.TxPipeline()

	video := repository.Video{
		URL:       params.URL,
		AddedByID: params.AddedByID,
		RoomID:    params.RoomID,
	}

	videoKey := videoPrefix + ":" + params.VideoID
	r.hSetIfNotExists(ctx, pipe, videoKey, video)
	pipe.Expire(ctx, videoKey, 10*time.Minute)

	memberListKey := "room" + ":" + params.RoomID + ":" + "playlist"

	r.addWithIncrement(ctx, pipe, memberListKey, params.VideoID)
	pipe.Expire(ctx, memberListKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}
