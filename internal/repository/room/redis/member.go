package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getMemberKey(memberID string) string {
	return "member:" + memberID
}

func (r repo) getMemberListKey(roomID string) string {
	return "room:" + roomID + ":memberlist"
}

func (r repo) addMemberToList(ctx context.Context, roomID, memberID string) error {
	pipe := r.rc.TxPipeline()
	memberListKey := r.getMemberListKey(roomID)

	addErr := r.addWithIncrement(ctx, pipe, memberListKey, memberID)
	expErr := pipe.Expire(ctx, memberListKey, 10*time.Minute).Err()

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	if addErr != nil {
		return addErr
	}

	if expErr != nil {
		return expErr
	}

	return nil
}

func (r repo) SetMember(ctx context.Context, params *room.SetMemberParams) error {
	funcName := "room.redis.SetMember"
	slog.DebugContext(ctx, funcName, "params", params)
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
	if err := r.hSetIfNotExists(ctx, r.rc, memberKey, member); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if err := r.rc.Expire(ctx, memberKey, 10*time.Minute).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if err := r.addMemberToList(ctx, params.RoomID, params.MemberID); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}

func (r repo) AddMemberToList(ctx context.Context, params *room.AddMemberToListParams) error {
	funcName := "room.redis.AddMemberToList"
	slog.DebugContext(ctx, funcName, "params", params)
	if err := r.addMemberToList(ctx, params.RoomID, params.MemberID); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}

func (r repo) RemoveMember(ctx context.Context, params *room.RemoveMemberParams) error {
	funcName := "room.redis.RemoveMember"
	slog.DebugContext(ctx, funcName, "params", params)
	if err := r.rc.ZRem(ctx, r.getMemberListKey(params.RoomID), params.MemberID).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	// res, err := r.rc.Del(ctx, memberPrefix+":"+params.MemberID).Result()
	// if err != nil {
	// 	slog.DebugContext(ctx,funcName, "err", err)
	// 	return err
	// }

	// if res == 0 {
	// 	return room.ErrMemberNotFound
	// }

	return nil
}

func (r repo) GetMemberRoomID(ctx context.Context, memberID string) (string, error) {
	funcName := "room.redis.GetMemberRoomID"
	slog.DebugContext(ctx, funcName, "memberID", memberID)
	res, err := r.rc.HGet(ctx, r.getMemberKey(memberID), "room_id").Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return "", err
	}

	fmt.Printf("res: %s\n", res)
	if res == "" {
		slog.DebugContext(ctx, funcName, "roomID", res)
		return "", room.ErrMemberNotFound
	}

	slog.DebugContext(ctx, funcName, "roomID", res)
	return res, nil
}

func (r repo) IsMemberAdmin(ctx context.Context, memberID string) (bool, error) {
	funcName := "room.redis.IsMemberAdmin"
	slog.DebugContext(ctx, funcName, "memberID", memberID)
	isAdmin, err := r.rc.HGet(ctx, r.getMemberKey(memberID), "is_admin").Bool()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return false, err
	}

	slog.DebugContext(ctx, funcName, "isAdmin", isAdmin)
	return isAdmin, nil
}

func (r repo) GetMemberIDs(ctx context.Context, roomID string) ([]string, error) {
	funcName := "room.redis.GetMembersIDs"
	slog.DebugContext(ctx, funcName, "roomID", roomID)
	memberIDs, err := r.rc.ZRange(ctx, r.getMemberListKey(roomID), 0, -1).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return nil, err
	}

	slog.DebugContext(ctx, funcName, "memberIDs", memberIDs)
	return memberIDs, nil
}

func (r repo) GetMember(ctx context.Context, memberID string) (room.Member, error) {
	funcName := "room.redis.GetMember"
	slog.DebugContext(ctx, funcName, "memberID", memberID)
	var member room.Member
	err := r.rc.HGetAll(ctx, r.getMemberKey(memberID)).Scan(&member)
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return room.Member{}, err
	}

	if member.Username == "" {
		slog.DebugContext(ctx, funcName, "error", room.ErrMemberNotFound)
		return room.Member{}, room.ErrMemberNotFound
	}

	slog.DebugContext(ctx, funcName, "member", member)
	return member, nil
}

func (r repo) UpdateMemberIsAdmin(ctx context.Context, memberID string, isAdmin bool) error {
	funcName := "room.redis.UpdateMemberIsAdmin"
	slog.DebugContext(ctx, funcName, "memberID", memberID, "isAdmin", isAdmin)
	//? Maybe dont check existence because there is check on service layer that member in current room
	key := r.getMemberKey(memberID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if cmd.Val() == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "is_admin", isAdmin).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberIsOnline(ctx context.Context, memberID string, isOnline bool) error {
	funcName := "room.redis.UpdateMemberIsOnline"
	slog.DebugContext(ctx, funcName, "memberID", memberID, "isOnline", isOnline)
	key := r.getMemberKey(memberID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if cmd.Val() == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "is_online", isOnline).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberIsMuted(ctx context.Context, memberID string, isMuted bool) error {
	funcName := "room.redis.UpdateMemberIsMuted"
	slog.DebugContext(ctx, funcName, "memberID", memberID, "isMuted", isMuted)
	key := r.getMemberKey(memberID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if cmd.Val() == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "is_muted", isMuted).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberColor(ctx context.Context, memberID, color string) error {
	funcName := "room.redis.UpdateMemberColor"
	slog.DebugContext(ctx, funcName, "memberID", memberID, "color", color)
	key := r.getMemberKey(memberID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if cmd.Val() == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "color", color).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberAvatarURL(ctx context.Context, memberID, avatarURL string) error {
	funcName := "room.redis.UpdateMemberAvatarURL"
	slog.DebugContext(ctx, funcName, "memberID", memberID, "avatarURL", avatarURL)
	key := r.getMemberKey(memberID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if cmd.Val() == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "avatar_url", avatarURL).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}

func (r repo) UpdateMemberUsername(ctx context.Context, memberID, username string) error {
	funcName := "room.redis.UpdateMemberUsername"
	slog.DebugContext(ctx, funcName, "memberID", memberID, "username", username)
	key := r.getMemberKey(memberID)
	cmd := r.rc.Exists(ctx, key)
	if err := cmd.Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if cmd.Val() == 0 {
		slog.DebugContext(ctx, funcName, "error", room.ErrMemberNotFound)
		return room.ErrMemberNotFound
	}

	if err := r.rc.HSet(ctx, key, "username", username).Err(); err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	return nil
}
