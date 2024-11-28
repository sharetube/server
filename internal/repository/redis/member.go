package redis

import (
	"context"
	"errors"
	"time"

	"github.com/sharetube/server/internal/repository"
)

const memberPrefix = "member"

func (r Repo) getMemberListKey(roomID string) string {
	return "room" + ":" + roomID + ":" + "memberlist"
}

func (r Repo) SetMember(ctx context.Context, params *repository.SetMemberParams) error {
	pipe := r.rc.TxPipeline()

	member := repository.Member{
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		IsMuted:   params.IsMuted,
		IsAdmin:   params.IsAdmin,
		IsOnline:  params.IsOnline,
		RoomID:    params.RoomID,
	}
	memberKey := memberPrefix + ":" + params.MemberID
	r.hSetIfNotExists(ctx, pipe, memberKey, member)
	pipe.Expire(ctx, memberKey, 10*time.Minute)

	memberListKey := r.getMemberListKey(params.RoomID)

	r.addWithIncrement(ctx, pipe, memberListKey, params.MemberID)
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
