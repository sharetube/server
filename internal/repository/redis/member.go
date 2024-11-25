package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const memberPrefix = "member"

type Member struct {
	Username  string `redis:"username"`
	Color     string `redis:"color"`
	AvatarURL string `redis:"avatar_url"`
	IsMuted   bool   `redis:"is_muted"`
	IsAdmin   bool   `redis:"is_admin"`
	IsOnline  bool   `redis:"is_online"`
	RoomID    string `redis:"room_id"`
}

type CreateMemberParams struct {
	ID        string
	Username  string
	Color     string
	AvatarURL string
	IsMuted   bool
	IsAdmin   bool
	IsOnline  bool
	RoomID    string
}

func (r Repo) CreateMember(ctx context.Context, params *CreateMemberParams) error {
	pipe := r.rc.TxPipeline()

	member := Member{
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		IsMuted:   params.IsMuted,
		IsAdmin:   params.IsAdmin,
		IsOnline:  params.IsOnline,
		RoomID:    params.RoomID,
	}
	memberKey := memberPrefix + ":" + params.ID
	r.HSetIfNotExists(ctx, pipe, memberKey, member)
	// pipe.Expire(ctx, memberKey, 10*time.Minute)

	memberListKey := "room" + ":" + params.RoomID + ":" + "memberlist"
	lastScore := pipe.ZCard(ctx, memberListKey).Val()
	pipe.ZAdd(ctx, memberListKey, redis.Z{
		Score:  float64(lastScore + 1),
		Member: memberKey,
	})
	// pipe.Expire(ctx, memberListKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func (r Repo) GetMemberRoomId(ctx context.Context, memberID string) (string, error) {
	roomID, err := r.rc.HGet(ctx, memberPrefix+":"+memberID, "room_id").Result()
	if err != nil {
		return "", err
	}

	return roomID, nil
}
