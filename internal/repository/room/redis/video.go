package redis

import (
	"context"
	"time"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getVideoKey(roomID, videoID string) string {
	return "room:" + roomID + ":video:" + videoID
}

func (r repo) getPlaylistKey(roomID string) string {
	return "room:" + roomID + ":playlist"
}

func (r repo) getPreviousVideoKey(roomID string) string {
	return "room:" + roomID + ":previous-video"
}

func (r repo) getPlaylistVersionKey(roomID string) string {
	return "room:" + roomID + ":playlist-version"
}

func (r repo) GetVideosLength(ctx context.Context, roomID string) (int, error) {
	playlistKey := r.getPlaylistKey(roomID)
	cmd := r.rc.ZCard(ctx, playlistKey)
	if err := cmd.Err(); err != nil {
		return 0, err
	}

	videosLength := int(cmd.Val())
	return videosLength, nil
}

func (r repo) SetVideo(ctx context.Context, params *room.SetVideoParams) error {
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
		AddedByID: params.AddedByID,
		RoomID:    params.RoomID,
	}
	videoKey := r.getVideoKey(params.RoomID, params.VideoID)
	r.hSetIfNotExists(ctx, pipe, videoKey, video)
	pipe.Expire(ctx, videoKey, 10*time.Minute)

	playlistKey := r.getPlaylistKey(params.RoomID)
	r.addWithIncrement(ctx, pipe, playlistKey, params.VideoID)
	pipe.Expire(ctx, playlistKey, 10*time.Minute)

	if err := r.executePipe(ctx, pipe); err != nil {
		return err
	}

	return nil
}

func (r repo) getVideo(ctx context.Context, params *room.GetVideoParams) (room.Video, error) {
	video := room.Video{}
	if err := r.rc.HGetAll(ctx, r.getVideoKey(params.RoomID, params.VideoID)).Scan(&video); err != nil {
		return room.Video{}, err
	}

	if video.URL == "" {
		return room.Video{}, room.ErrVideoNotFound
	}

	return video, nil
}

func (r repo) GetVideo(ctx context.Context, params *room.GetVideoParams) (room.Video, error) {
	video, err := r.getVideo(ctx, params)
	if err != nil {
		return room.Video{}, err
	}

	return video, nil
}

func (r repo) getVideoIDs(ctx context.Context, roomID string) ([]string, error) {
	return r.rc.ZRange(ctx, r.getPlaylistKey(roomID), 0, -1).Result()
}

func (r repo) GetVideoIDs(ctx context.Context, roomID string) ([]string, error) {
	videoIDs, err := r.getVideoIDs(ctx, roomID)
	if err != nil {
		return nil, err
	}

	return videoIDs, nil
}

func (r repo) RemoveVideo(ctx context.Context, params *room.RemoveVideoParams) error {
	res, err := r.rc.ZRem(ctx, r.getPlaylistKey(params.RoomID), params.VideoID).Result()
	if err != nil {
		return err
	}

	if res == 0 {
		return room.ErrVideoNotFound
	}

	previousVideoID, err := r.getPreviousVideoID(ctx, params.RoomID)
	if err != nil {
		return err
	}

	if err := r.rc.Del(ctx, r.getVideoKey(params.RoomID, previousVideoID)).Err(); err != nil {
		return err
	}

	return nil
}

func (r repo) getPreviousVideoID(ctx context.Context, roomID string) (string, error) {
	return r.rc.Get(ctx, r.getPreviousVideoKey(roomID)).Result()
}

func (r repo) GetPreviousVideoID(ctx context.Context, roomID string) (string, error) {
	previousVideoID, err := r.getPreviousVideoID(ctx, roomID)
	if err != nil {
		return "", err
	}

	if previousVideoID == "" {
		return "", room.ErrNoPreviousVideo
	}

	return previousVideoID, nil
}
