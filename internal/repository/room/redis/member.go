package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getMemberKey(memberId string) string {
	return "member:" + memberId
}

func (r repo) getMemberListKey(roomId string) string {
	return "room:" + roomId + ":memberlist"
}

func (r repo) addMemberToList(ctx context.Context, pipe redis.Pipeliner, roomId, memberId string) {
	memberListKey := r.getMemberListKey(roomId)

	r.addWithIncrement(ctx, pipe, memberListKey, memberId)
	pipe.Expire(ctx, memberListKey, 10*time.Minute)
}

func (r repo) SetMember(ctx context.Context, params *room.SetMemberParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	pipe := r.rc.TxPipeline()

	member := room.Member{
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		IsMuted:   params.IsMuted,
		IsAdmin:   params.IsAdmin,
		IsReady:   params.IsReady,
	}

	memberKey := r.getMemberKey(params.MemberId)
	//? replace with hsetifnotexists
	r.HSetStruct(ctx, pipe, memberKey, member)
	pipe.Expire(ctx, memberKey, 10*time.Minute)

	r.addMemberToList(ctx, pipe, params.RoomId, params.MemberId)

	if err := r.executePipe(ctx, pipe); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}
	return nil
}

func (r repo) AddMemberToList(ctx context.Context, params *room.AddMemberToListParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	pipe := r.rc.TxPipeline()

	// todo: execute in transaction
	exists := r.rc.Exists(ctx, r.getMemberKey(params.MemberId)).Val()

	if exists == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	r.addMemberToList(ctx, pipe, params.RoomId, params.MemberId)

	if err := r.executePipe(ctx, pipe); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) removeMember(ctx context.Context, memberId string) error {
	res, err := r.rc.Del(ctx, r.getMemberKey(memberId)).Result()
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
	r.logger.DebugContext(ctx, "called", "params", params)
	if err := r.removeMemberFromList(ctx, params.RoomId, params.MemberId); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) RemoveMember(ctx context.Context, params *room.RemoveMemberParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	if err := r.removeMemberFromList(ctx, params.RoomId, params.MemberId); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if err := r.removeMember(ctx, params.MemberId); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) GetMemberIsAdmin(ctx context.Context, roomId, memberId string) (bool, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"room_id":   roomId,
		"member_id": memberId,
	})
	isAdmin, err := r.rc.HGet(ctx, r.getMemberKey(memberId), "is_admin").Bool()
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return false, err
	}

	return isAdmin, nil
}

func (r repo) GetMemberIds(ctx context.Context, roomId string) ([]string, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"room_id": roomId,
	})
	memberIds, err := r.rc.ZRange(ctx, r.getMemberListKey(roomId), 0, -1).Result()
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return nil, err
	}

	return memberIds, nil
}

func (r repo) GetMember(ctx context.Context, params *room.GetMemberParams) (room.Member, error) {
	r.logger.DebugContext(ctx, "called", "params", params)
	var member room.Member
	err := r.rc.HGetAll(ctx, r.getMemberKey(params.MemberId)).Scan(&member)
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return room.Member{}, err
	}

	if member.Username == "" {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.Member{}, room.ErrMemberNotFound
	}

	return member, nil
}

func (r repo) UpdateMemberIsAdmin(ctx context.Context, roomId, memberId string, isAdmin bool) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"room_id":   roomId,
		"member_id": memberId,
		"is_admin":  isAdmin,
	})
	//? Maybe dont check existence because there is check on service layer that member in current room
	key := r.getMemberKey(memberId)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "is_admin", isAdmin).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberIsReady(ctx context.Context, roomId, memberId string, isReady bool) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"room_id":   roomId,
		"member_id": memberId,
		"is_ready":  isReady,
	})
	key := r.getMemberKey(memberId)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "is_ready", isReady).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberIsMuted(ctx context.Context, roomId, memberId string, isMuted bool) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"room_id":   roomId,
		"member_id": memberId,
		"is_muted":  isMuted,
	})
	key := r.getMemberKey(memberId)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "is_muted", isMuted).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberColor(ctx context.Context, roomId, memberId, color string) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"room_id":   roomId,
		"member_id": memberId,
		"color":     color,
	})
	key := r.getMemberKey(memberId)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "color", color).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberAvatarURL(ctx context.Context, roomId, memberId string, avatarURL *string) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"room_id":    roomId,
		"member_id":  memberId,
		"avatar_url": avatarURL,
	})
	key := r.getMemberKey(memberId)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if avatarURL == nil {
		if err := r.rc.HDel(ctx, key, "avatar_url").Err(); err != nil {
			r.logger.DebugContext(ctx, "returned", "error", err)
			return err
		}
	} else {
		if err := r.rc.HSet(ctx, key, "avatar_url", avatarURL).Err(); err != nil {
			r.logger.DebugContext(ctx, "returned", "error", err)
			return err
		}
	}

	return nil
}

func (r repo) UpdateMemberUsername(ctx context.Context, roomId, memberId, username string) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"room_id":   roomId,
		"member_id": memberId,
		"username":  username,
	})
	key := r.getMemberKey(memberId)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "username", username).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}
