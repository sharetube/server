package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getVideoKey(roomId, videoId string) string {
	return "room:" + roomId + ":video:" + videoId
}

func (r repo) getPlaylistKey(roomId string) string {
	return "room:" + roomId + ":playlist"
}

func (r repo) getLastVideoKey(roomId string) string {
	return "room:" + roomId + ":last-video"
}

// func (r repo) getPlaylistVersionKey(roomId string) string {
// 	return "room:" + roomId + ":playlist-version"
// }

func (r repo) GetVideosLength(ctx context.Context, roomId string) (int, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})
	playlistKey := r.getPlaylistKey(roomId)
	cmd := r.rc.ZCard(ctx, playlistKey)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return 0, err
	}

	r.rc.Expire(ctx, playlistKey, r.expireDuration)

	videosLength := int(cmd.Val())
	return videosLength, nil
}

func (r repo) SetVideo(ctx context.Context, params *room.SetVideoParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	pipe := r.rc.TxPipeline()

	// playlistVersionKey := r.getPlaylistVersionKey(params.RoomID)
	// playlistVersion, err := r.rc.Get(ctx, playlistVersionKey).Int()
	// fmt.Printf("playlistVersion: %d\n", playlistVersion)
	// if err != nil {
	// 	fmt.Printf("set video error: %v\n", err)
	// 	return err
	// }

	video := room.Video{
		URL: params.URL,
	}
	videoKey := r.getVideoKey(params.RoomId, params.VideoId)
	r.hSetIfNotExists(ctx, pipe, videoKey, video)
	pipe.Expire(ctx, videoKey, r.expireDuration)

	playlistKey := r.getPlaylistKey(params.RoomId)
	r.addWithIncrement(ctx, pipe, playlistKey, params.VideoId)
	pipe.Expire(ctx, playlistKey, r.expireDuration)

	if err := r.executePipe(ctx, pipe); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) getVideo(ctx context.Context, params *room.GetVideoParams) (room.Video, error) {
	var video room.Video
	videoKey := r.getVideoKey(params.RoomId, params.VideoId)
	if err := r.rc.HGetAll(ctx, videoKey).Scan(&video); err != nil {
		return room.Video{}, err
	}

	if video.URL == "" {
		return room.Video{}, room.ErrVideoNotFound
	}

	r.rc.Expire(ctx, videoKey, r.expireDuration)

	return video, nil
}

func (r repo) GetVideo(ctx context.Context, params *room.GetVideoParams) (room.Video, error) {
	r.logger.DebugContext(ctx, "called", "params", params)
	video, err := r.getVideo(ctx, params)
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return room.Video{}, err
	}

	return video, nil
}

func (r repo) GetVideoIds(ctx context.Context, roomId string) ([]string, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})

	playlistKey := r.getPlaylistKey(roomId)
	videoIds, err := r.rc.ZRange(ctx, playlistKey, 0, -1).Result()
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return nil, err
	}

	r.rc.Expire(ctx, playlistKey, r.expireDuration)

	return videoIds, nil
}

func (r repo) RemoveVideo(ctx context.Context, params *room.RemoveVideoParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	res, err := r.rc.ZRem(ctx, r.getPlaylistKey(params.RoomId), params.VideoId).Result()
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if res == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrVideoNotFound)
		return room.ErrVideoNotFound
	}

	if err := r.rc.Del(ctx, r.getVideoKey(params.RoomId, params.VideoId)).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) GetLastVideoId(ctx context.Context, roomId string) (*string, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})

	lastVideoKey := r.getLastVideoKey(roomId)
	lastVideoId, err := r.rc.Get(ctx, lastVideoKey).Result()
	if err != nil && err != redis.Nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return nil, err
	}

	if lastVideoId == "" {
		return nil, nil
	}

	r.rc.Expire(ctx, lastVideoKey, r.expireDuration)

	return &lastVideoId, nil
}
