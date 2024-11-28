package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type SetMemberParams struct {
	MemberID  string
	Username  string
	Color     string
	AvatarURL string
	IsMuted   bool
	IsAdmin   bool
	IsOnline  bool
	RoomID    string
}

func (r Repo) getMemberListKey(roomID string) string {
	return "room" + ":" + roomID + ":" + "memberlist"
}

func (r Repo) SetMember(ctx context.Context, params *SetMemberParams) error {
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
	memberKey := memberPrefix + ":" + params.MemberID
	r.HSetIfNotExists(ctx, pipe, memberKey, member)
	pipe.Expire(ctx, memberKey, 10*time.Minute)

	memberListKey := r.getMemberListKey(params.RoomID)
	lastScore := pipe.ZCard(ctx, memberListKey).Val()
	fmt.Printf("lastScore: %v\n", lastScore)
	pipe.ZAdd(ctx, memberListKey, redis.Z{
		Score:  float64(lastScore + 1),
		Member: params.MemberID,
	})
	pipe.Expire(ctx, memberListKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func (r Repo) GetMemberRoomId(ctx context.Context, memberID string) (string, error) {
	roomID := r.rc.HGet(ctx, memberPrefix+":"+memberID, "room_id").Val()
	if roomID == "" {
		return "", errors.New("member not found")
	}

	return roomID, nil
}

func (r Repo) GetMemberIDs(ctx context.Context, roomID string) ([]string, error) {
	memberListKey := r.getMemberListKey(roomID)
	memberIDs, err := r.rc.ZRange(ctx, memberListKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	return memberIDs, nil
}
