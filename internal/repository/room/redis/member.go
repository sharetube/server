package redis

import (
	"context"
	"fmt"

	"github.com/skewb1k/goutils/maps"

	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository/room"
)

const (
	usernameKey  = "username"
	colorKey     = "color"
	avatarUrlKey = "avatar_url"
	isMutedKey   = "is_muted"
	isAdminKey   = "is_admin"
	isReadyKey   = "is_ready"
)

func (r repo) getMemberKey(roomId, memberId string) string {
	return fmt.Sprintf("room:%s:member:%s", roomId, memberId)
}

func (r repo) getMemberListKey(roomId string) string {
	return fmt.Sprintf("room:%s:memberlist", roomId)
}

func (r repo) addMemberToList(ctx context.Context, pipe redis.Cmdable, roomId, memberId string) {
	// pipe.Expire(ctx, memberListKey, r.maxExpireDuration)
}

func (r repo) SetMember(ctx context.Context, params *room.SetMemberParams) error {
	// pipe := r.rc.TxPipeline()

	memberKey := r.getMemberKey(params.RoomId, params.MemberId)
	return r.rc.HSet(ctx, memberKey, maps.OmitNilPointers(map[string]any{
		usernameKey:  params.Username,
		avatarUrlKey: params.AvatarUrl,
		colorKey:     params.Color,
		isMutedKey:   params.IsMuted,
		isAdminKey:   params.IsAdmin,
		isReadyKey:   params.IsReady,
	})).Err()
	// pipe.Expire(ctx, memberKey, r.maxExpireDuration)
	// return r.executePipe(ctx, pipe)
}

func (r repo) AddMemberToList(ctx context.Context, params *room.AddMemberToListParams) error {
	// pipe := r.rc.TxPipeline()
	// return r.executePipe(ctx, pipe)

	exists := r.rc.Exists(ctx, r.getMemberKey(params.RoomId, params.MemberId)).Val()

	if exists == 0 {
		return room.ErrMemberNotFound
	}

	return r.addWithIncrement(ctx, r.rc, r.getMemberListKey(params.RoomId), params.MemberId).Err()
}

func (r repo) removeMember(ctx context.Context, roomId, memberId string) error {
	res, err := r.rc.Del(ctx, r.getMemberKey(roomId, memberId)).Result()
	if err != nil {
		return err
	}

	if res == 0 {
		return room.ErrMemberNotFound
	}

	return nil
}

func (r repo) removeMemberFromList(ctx context.Context, roomId, memberId string) error {
	return r.rc.ZRem(ctx, r.getMemberListKey(roomId), memberId).Err()
}

func (r repo) RemoveMemberFromList(ctx context.Context, params *room.RemoveMemberFromListParams) error {
	return r.removeMemberFromList(ctx, params.RoomId, params.MemberId)
}

func (r repo) RemoveMember(ctx context.Context, params *room.RemoveMemberParams) error {
	if err := r.removeMemberFromList(ctx, params.RoomId, params.MemberId); err != nil {
		return err
	}

	return r.removeMember(ctx, params.RoomId, params.MemberId)
}

func (r repo) ExpireMembers(ctx context.Context, params *room.ExpireMembersParams) error {
	r.expireKeysWithPrefix(ctx, r.rc, r.getMemberKey(params.RoomId, "*"), params.ExpireAt)
	return nil
}

func (r repo) GetMemberIsMuted(ctx context.Context, roomId, memberId string) (bool, error) {
	memberKey := r.getMemberKey(roomId, memberId)
	isAdmin, err := r.rc.HGet(ctx, memberKey, isMutedKey).Bool()
	if err != nil {
		return false, err
	}
	// r.rc.Expire(ctx, memberKey, r.maxExpireDuration)

	return isAdmin, nil
}

func (r repo) GetMemberIsAdmin(ctx context.Context, roomId, memberId string) (bool, error) {
	memberKey := r.getMemberKey(roomId, memberId)
	isAdmin, err := r.rc.HGet(ctx, memberKey, isAdminKey).Bool()
	if err != nil {
		return false, err
	}
	// r.rc.Expire(ctx, memberKey, r.maxExpireDuration)

	return isAdmin, nil
}

func (r repo) GetMemberIds(ctx context.Context, roomId string) ([]string, error) {
	memberListKey := r.getMemberListKey(roomId)
	memberIds, err := r.rc.ZRange(ctx, memberListKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	// r.rc.Expire(ctx, memberListKey, r.maxExpireDuration)

	return memberIds, nil
}

func (r repo) GetMember(ctx context.Context, params *room.GetMemberParams) (room.Member, error) {
	memberKey := r.getMemberKey(params.RoomId, params.MemberId)
	memberMap, err := r.rc.HGetAll(ctx, memberKey).Result()
	if err != nil {
		return room.Member{}, err
	}

	if len(memberMap) == 0 {
		return room.Member{}, room.ErrMemberNotFound
	}

	// r.rc.Expire(ctx, memberKey, r.maxExpireDuration)

	return room.Member{
		Username:  memberMap[usernameKey],
		Color:     memberMap[colorKey],
		AvatarUrl: maps.PtrFromStringMap(memberMap, avatarUrlKey),
		IsMuted:   r.fieldToBool(memberMap[isMutedKey]),
		IsAdmin:   r.fieldToBool(memberMap[isAdminKey]),
		IsReady:   r.fieldToBool(memberMap[isReadyKey]),
	}, nil
}

func (r repo) UpdateMemberIsAdmin(ctx context.Context, roomId, memberId string, isAdmin bool) error {
	//? dont check existence because there is check on service layer that member in current room
	memberKey := r.getMemberKey(roomId, memberId)
	// todo: refacotr with result
	cmd := r.rc.Exists(ctx, memberKey)
	if err := cmd.Err(); err != nil {
		return err
	}

	if cmd.Val() == 0 {
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, memberKey, isAdminKey, isAdmin).Err(); err != nil {
		return err
	}

	// r.rc.Expire(ctx, memberKey, r.maxExpireDuration)

	return nil
}

func (r repo) UpdateMemberIsReady(ctx context.Context, roomId, memberId string, isReady bool) error {
	memberKey := r.getMemberKey(roomId, memberId)
	cmd := r.rc.Exists(ctx, memberKey)
	if err := cmd.Err(); err != nil {
		return err
	}

	if cmd.Val() == 0 {
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, memberKey, isReadyKey, isReady).Err(); err != nil {
		return err
	}

	// r.rc.Expire(ctx, memberKey, r.maxExpireDuration)

	return nil
}

func (r repo) UpdateMemberIsMuted(ctx context.Context, roomId, memberId string, isMuted bool) error {
	memberKey := r.getMemberKey(roomId, memberId)
	cmd := r.rc.Exists(ctx, memberKey)
	if err := cmd.Err(); err != nil {
		return err
	}

	if cmd.Val() == 0 {
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, memberKey, isMutedKey, isMuted).Err(); err != nil {
		return err
	}

	// r.rc.Expire(ctx, memberKey, r.maxExpireDuration)

	return nil
}

func (r repo) UpdateMemberColor(ctx context.Context, roomId, memberId, color string) error {
	memberKey := r.getMemberKey(roomId, memberId)
	existsCmd := r.rc.Exists(ctx, memberKey)
	if err := existsCmd.Err(); err != nil {
		return err
	}

	if existsCmd.Val() == 0 {
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, memberKey, colorKey, color).Err(); err != nil {
		return err
	}

	// r.rc.Expire(ctx, memberKey, r.maxExpireDuration)

	return nil
}

func (r repo) UpdateMemberAvatarUrl(ctx context.Context, roomId, memberId string, avatarUrl *string) error {
	memberKey := r.getMemberKey(roomId, memberId)
	existsCmd := r.rc.Exists(ctx, memberKey)
	if err := existsCmd.Err(); err != nil {
		return err
	}

	if existsCmd.Val() == 0 {
		return room.ErrMemberNotFound
	}

	if avatarUrl == nil {
		if err := r.rc.HDel(ctx, memberKey, avatarUrlKey).Err(); err != nil {
			return err
		}
	} else {
		if err := r.rc.HSet(ctx, memberKey, avatarUrlKey, avatarUrl).Err(); err != nil {
			return err
		}
	}

	// r.rc.Expire(ctx, memberKey, r.maxExpireDuration)

	return nil
}

func (r repo) UpdateMemberUsername(ctx context.Context, roomId, memberId, username string) error {
	memberKey := r.getMemberKey(roomId, memberId)
	cmd := r.rc.Exists(ctx, memberKey)
	if err := cmd.Err(); err != nil {
		return err
	}

	if cmd.Val() == 0 {
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, memberKey, usernameKey, username).Err(); err != nil {
		return err
	}

	// r.rc.Expire(ctx, memberKey, r.maxExpireDuration)

	return nil
}
