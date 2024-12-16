package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getMemberKey(memberID string) string {
	return "member:" + memberID
}

func (r repo) getMemberListKey(roomID string) string {
	return "room:" + roomID + ":memberlist"
}

func (r repo) addMemberToList(ctx context.Context, pipe redis.Pipeliner, roomID, memberID string) {
	memberListKey := r.getMemberListKey(roomID)

	r.addWithIncrement(ctx, pipe, memberListKey, memberID)
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
		IsOnline:  params.IsOnline,
		RoomID:    params.RoomID,
	}
	memberKey := r.getMemberKey(params.MemberID)
	//? replace with hsetifnotexists
	pipe.HSet(ctx, memberKey, member)
	pipe.Expire(ctx, memberKey, 10*time.Minute)

	r.addMemberToList(ctx, pipe, params.RoomID, params.MemberID)

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
	exists := r.rc.Exists(ctx, r.getMemberKey(params.MemberID)).Val()

	if exists == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	r.addMemberToList(ctx, pipe, params.RoomID, params.MemberID)

	if err := r.executePipe(ctx, pipe); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) removeMember(ctx context.Context, memberID string) error {
	res, err := r.rc.Del(ctx, r.getMemberKey(memberID)).Result()
	if err != nil {
		return err
	}

	if res == 0 {
		return room.ErrMemberNotFound
	}

	return nil
}

func (r repo) removeMemberFromList(ctx context.Context, roomID, memberID string) error {
	return r.rc.ZRem(ctx, r.getMemberListKey(roomID), memberID).Err()
}

func (r repo) RemoveMemberFromList(ctx context.Context, params *room.RemoveMemberFromListParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	if err := r.removeMemberFromList(ctx, params.RoomID, params.MemberID); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) RemoveMember(ctx context.Context, params *room.RemoveMemberParams) error {
	r.logger.DebugContext(ctx, "called", "params", params)
	if err := r.removeMemberFromList(ctx, params.RoomID, params.MemberID); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if err := r.removeMember(ctx, params.MemberID); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) GetMemberIsAdmin(ctx context.Context, roomID, memberID string) (bool, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"roomID":   roomID,
		"memberID": memberID,
	})
	isAdmin, err := r.rc.HGet(ctx, r.getMemberKey(memberID), "is_admin").Bool()
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return false, err
	}

	return isAdmin, nil
}

func (r repo) GetMemberIDs(ctx context.Context, roomID string) ([]string, error) {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"roomID": roomID,
	})
	memberIDs, err := r.rc.ZRange(ctx, r.getMemberListKey(roomID), 0, -1).Result()
	if err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return nil, err
	}

	return memberIDs, nil
}

func (r repo) GetMember(ctx context.Context, params *room.GetMemberParams) (room.Member, error) {
	r.logger.DebugContext(ctx, "called", "params", params)
	var member room.Member
	err := r.rc.HGetAll(ctx, r.getMemberKey(params.MemberID)).Scan(&member)
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

func (r repo) UpdateMemberIsAdmin(ctx context.Context, roomID, memberID string, isAdmin bool) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"roomID":   roomID,
		"memberID": memberID,
		"isAdmin":  isAdmin,
	})
	//? Maybe dont check existence because there is check on service layer that member in current room
	key := r.getMemberKey(memberID)
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

func (r repo) UpdateMemberIsOnline(ctx context.Context, roomID, memberID string, isOnline bool) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"roomID":   roomID,
		"memberID": memberID,
		"isOnline": isOnline,
	})
	key := r.getMemberKey(memberID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "is_online", isOnline).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberIsMuted(ctx context.Context, roomID, memberID string, isMuted bool) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"roomID":   roomID,
		"memberID": memberID,
		"isMuted":  isMuted,
	})
	key := r.getMemberKey(memberID)
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

func (r repo) UpdateMemberColor(ctx context.Context, roomID, memberID, color string) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"roomID":   roomID,
		"memberID": memberID,
		"color":    color,
	})
	key := r.getMemberKey(memberID)
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

func (r repo) UpdateMemberAvatarURL(ctx context.Context, roomID, memberID, avatarURL string) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"roomID":    roomID,
		"memberID":  memberID,
		"avatarURL": avatarURL,
	})
	key := r.getMemberKey(memberID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	if cmd.Val() == 0 {
		r.logger.DebugContext(ctx, "returned", "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "avatar_url", avatarURL).Err(); err != nil {
		r.logger.DebugContext(ctx, "returned", "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberUsername(ctx context.Context, roomID, memberID, username string) error {
	r.logger.DebugContext(ctx, "called", "params", map[string]any{
		"roomID":   roomID,
		"memberID": memberID,
		"username": username,
	})
	key := r.getMemberKey(memberID)
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
