package redis

import (
	"context"
	"time"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getVideoKey(roomId, videoId string) string {
	return "room:" + roomId + ":video:" + videoId
}

func (r repo) getPlaylistKey(roomId string) string {
	return "room:" + roomId + ":playlist"
}

func (r repo) getPreviousVideoKey(roomId string) string {
	return "room:" + roomId + ":previous-video"
}

func (r repo) getPlaylistVersionKey(roomId string) string {
	return "room:" + roomId + ":playlist-version"
}

func (r repo) GetVideosLength(ctx context.Context, roomId string) (int, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})
	playlistKey := r.getPlaylistKey(roomId)
	cmd := r.rc.ZCard(ctx, playlistKey)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return 0, err
	}

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
		URL:       params.URL,
		AddedById: params.AddedById,
		RoomId:    params.RoomId,
	}
	videoKey := r.getVideoKey(params.RoomId, params.VideoId)
	r.hSetIfNotExists(ctx, pipe, videoKey, video)
	pipe.Expire(ctx, videoKey, 10*time.Minute)

	playlistKey := r.getPlaylistKey(params.RoomId)
	r.addWithIncrement(ctx, pipe, playlistKey, params.VideoId)
	pipe.Expire(ctx, playlistKey, 10*time.Minute)

	if err := r.executePipe(ctx, pipe); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) getVideo(ctx context.Context, params *room.GetVideoParams) (room.Video, error) {
	video := room.Video{}
	if err := r.rc.HGetAll(ctx, r.getVideoKey(params.RoomId, params.VideoId)).Scan(&video); err != nil {
		return room.Video{}, err
	}

	if video.URL == "" {
		return room.Video{}, room.ErrVideoNotFound
	}

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

func (r repo) getVideoIds(ctx context.Context, roomId string) ([]string, error) {
	return r.rc.ZRange(ctx, r.getPlaylistKey(roomId), 0, -1).Result()
}

func (r repo) GetVideoIds(ctx context.Context, roomId string) ([]string, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})
	videoIds, err := r.getVideoIds(ctx, roomId)
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return nil, err
	}

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

	previousVideoId, err := r.getPreviousVideoId(ctx, params.RoomId)
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if err := r.rc.Del(ctx, r.getVideoKey(params.RoomId, previousVideoId)).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) getPreviousVideoId(ctx context.Context, roomId string) (string, error) {
	return r.rc.Get(ctx, r.getPreviousVideoKey(roomId)).Result()
}

func (r repo) GetPreviousVideoId(ctx context.Context, roomId string) (string, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]string{"room_id": roomId})
	previousVideoId, err := r.getPreviousVideoId(ctx, roomId)
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return "", err
	}

	if previousVideoId == "" {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrNoPreviousVideo)
		return "", room.ErrNoPreviousVideo
	}

	return previousVideoId, nil
}
