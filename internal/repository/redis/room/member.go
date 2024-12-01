package room

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/sharetube/server/internal/repository"
)

var (
	ErrMemberNotFound = errors.New("member not found")
)

const memberPrefix = "member"

func (r repo) getMemberListKey(roomID string) string {
	return "room" + ":" + roomID + ":" + "memberlist"
}

func (r repo) SetMember(ctx context.Context, params *repository.SetMemberParams) error {
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

func (r repo) RemoveMember(ctx context.Context, params *repository.RemoveMemberParams) error {
	if err := r.rc.ZRem(ctx, r.getMemberListKey(params.RoomID), params.MemberID).Err(); err != nil {
		slog.Info("failed to remove member from memberlist", "err", err)
		return err
	}

	res, err := r.rc.Del(ctx, memberPrefix+":"+params.MemberID).Result()
	if err != nil {
		slog.Info("failed to delete member", "err", err)
		return err
	}

	if res == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r repo) GetMemberRoomId(ctx context.Context, memberID string) (string, error) {
	roomID := r.rc.HGet(ctx, memberPrefix+":"+memberID, "room_id").Val()
	if roomID == "" {
		return "", ErrMemberNotFound
	}

	return roomID, nil
}

func (r repo) IsMemberAdmin(ctx context.Context, memberID string) (bool, error) {
	isAdmin, err := r.rc.HGet(ctx, memberPrefix+":"+memberID, "is_admin").Bool()
	if err != nil {
		return false, err
	}

	return isAdmin, nil
}

func (r repo) GetMembersIDs(ctx context.Context, roomID string) ([]string, error) {
	memberListKey := r.getMemberListKey(roomID)
	memberIDs, err := r.rc.ZRange(ctx, memberListKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	return memberIDs, nil
}

func (r repo) GetMember(ctx context.Context, memberID string) (repository.Member, error) {
	var member repository.Member
	err := r.rc.HGetAll(ctx, memberPrefix+":"+memberID).Scan(&member)
	if err != nil {
		return repository.Member{}, err
	}

	return member, nil
}

func (r repo) UpdateMemberIsAdmin(ctx context.Context, memberID string, isAdmin bool) error {
	//? Maybe dont check existence because there is check on service layer that member in current room
	key := memberPrefix + ":" + memberID
	if r.rc.Exists(ctx, key).Val() == 0 {
		return ErrMemberNotFound
	}

	return r.rc.HSet(ctx, key, "is_admin", isAdmin).Err()
}

func (r repo) UpdateMemberIsOnline(ctx context.Context, memberID string, isOnline bool) error {
	key := memberPrefix + ":" + memberID
	if r.rc.Exists(ctx, key).Val() == 0 {
		return ErrMemberNotFound
	}

	return r.rc.HSet(ctx, key, "is_online", isOnline).Err()
}

func (r repo) UpdateMemberIsMuted(ctx context.Context, memberID string, isMuted bool) error {
	key := memberPrefix + ":" + memberID
	if r.rc.Exists(ctx, key).Val() == 0 {
		return ErrMemberNotFound
	}

	return r.rc.HSet(ctx, key, "is_muted", isMuted).Err()
}

func (r repo) UpdateMemberColor(ctx context.Context, memberID, color string) error {
	key := memberPrefix + ":" + memberID
	if r.rc.Exists(ctx, key).Val() == 0 {
		return ErrMemberNotFound
	}

	return r.rc.HSet(ctx, key, "color", color).Err()
}

func (r repo) UpdateMemberAvatarURL(ctx context.Context, memberID, avatarURL string) error {
	key := memberPrefix + ":" + memberID
	if r.rc.Exists(ctx, key).Val() == 0 {
		return ErrMemberNotFound
	}

	return r.rc.HSet(ctx, key, "avatar_url", avatarURL).Err()
}

func (r repo) UpdateMemberUsername(ctx context.Context, memberID, username string) error {
	key := memberPrefix + ":" + memberID
	if r.rc.Exists(ctx, key).Val() == 0 {
		return ErrMemberNotFound
	}

	return r.rc.HSet(ctx, key, "username", username).Err()
}
