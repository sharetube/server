package room

import (
	"context"
	"log/slog"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository"
)

func (s service) getMemberList(ctx context.Context, roomID string) ([]Member, error) {
	memberlistIDs, err := s.roomRepo.GetMembersIDs(ctx, roomID)
	if err != nil {
		slog.Info("failed to get memberlist", "err", err)
		return []Member{}, err
	}

	memberlist := make([]Member, 0, len(memberlistIDs))
	for _, memberID := range memberlistIDs {
		member, err := s.roomRepo.GetMember(ctx, memberID)
		if err != nil {
			slog.Info("failed to get member", "err", err)
			return []Member{}, err
		}

		memberlist = append(memberlist, Member{
			ID:        memberID,
			Username:  member.Username,
			Color:     member.Color,
			AvatarURL: member.AvatarURL,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsOnline:  member.IsOnline,
		})
	}

	return memberlist, nil
}

type RemoveMemberParams struct {
	RemovedMemberID string
	SenderID        string
	RoomID          string
}

type RemoveMemberResponse struct {
	Conns      []*websocket.Conn
	Memberlist []Member
}

func (s service) RemoveMember(ctx context.Context, params *RemoveMemberParams) (RemoveMemberResponse, error) {
	isAdmin, err := s.roomRepo.IsMemberAdmin(ctx, params.SenderID)
	if err != nil {
		slog.Info("failed to check if member is admin", "err", err)
		return RemoveMemberResponse{}, err
	}
	if !isAdmin {
		return RemoveMemberResponse{}, ErrPermissionDenied
	}

	if err := s.roomRepo.RemoveMember(ctx, &repository.RemoveMemberParams{
		MemberID: params.RemovedMemberID,
		RoomID:   params.RoomID,
	}); err != nil {
		slog.Info("failed to remove member", "err", err)
		return RemoveMemberResponse{}, err
	}

	if err := s.connRepo.RemoveByMemberID(params.RemovedMemberID); err != nil {
		slog.Info("failed to remove conn", "err", err)
		return RemoveMemberResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		slog.Info("failed to get conns", "err", err)
		return RemoveMemberResponse{}, err
	}

	memberlist, err := s.getMemberList(ctx, params.RoomID)
	if err != nil {
		slog.Info("failed to get memberlist", "err", err)
		return RemoveMemberResponse{}, err
	}

	return RemoveMemberResponse{
		Conns:      conns,
		Memberlist: memberlist,
	}, nil
}

type PromoteMemberParams struct {
	PromotedMemberID string
	SenderID         string
	RoomID           string
}

type PromoteMemberResponse struct {
	Conns []*websocket.Conn
}

func (s service) PromoteMember(ctx context.Context, params *PromoteMemberParams) (PromoteMemberResponse, error) {
	isAdmin, err := s.roomRepo.IsMemberAdmin(ctx, params.SenderID)
	if err != nil {
		slog.Info("failed to check if member is admin", "err", err)
		return PromoteMemberResponse{}, err
	}
	if !isAdmin {
		return PromoteMemberResponse{}, ErrPermissionDenied
	}

	if err := s.roomRepo.UpdateMemberIsAdmin(ctx, params.PromotedMemberID, true); err != nil {
		slog.Info("failed to promote member", "err", err)
		return PromoteMemberResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		slog.Info("failed to get conns", "err", err)
		return PromoteMemberResponse{}, err
	}

	return PromoteMemberResponse{
		Conns: conns,
	}, nil
}
