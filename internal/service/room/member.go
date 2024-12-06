package room

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getMemberList(ctx context.Context, roomID string) ([]Member, error) {
	memberlistIDs, err := s.roomRepo.GetMemberIDs(ctx, roomID)
	if err != nil {
		return []Member{}, err
	}

	memberlist := make([]Member, 0, len(memberlistIDs))
	for _, memberID := range memberlistIDs {
		member, err := s.roomRepo.GetMember(ctx, memberID)
		if err != nil {
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
	Conn *websocket.Conn
}

func (s service) RemoveMember(ctx context.Context, params *RemoveMemberParams) (RemoveMemberResponse, error) {
	isAdmin, err := s.roomRepo.IsMemberAdmin(ctx, params.SenderID)
	if err != nil {
		return RemoveMemberResponse{}, err
	}

	if !isAdmin {
		return RemoveMemberResponse{}, ErrPermissionDenied
	}

	roomID, err := s.roomRepo.GetMemberRoomID(ctx, params.RemovedMemberID)
	if err != nil {
		return RemoveMemberResponse{}, err
	}

	if roomID != params.RoomID {
		return RemoveMemberResponse{}, ErrMemberNotFound
	}

	conn, err := s.connRepo.GetConn(params.RemovedMemberID)
	if err != nil {
		return RemoveMemberResponse{}, err
	}

	return RemoveMemberResponse{
		Conn: conn,
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
		return PromoteMemberResponse{}, err
	}
	if !isAdmin {
		return PromoteMemberResponse{}, ErrPermissionDenied
	}

	roomid, err := s.roomRepo.GetMemberRoomID(ctx, params.PromotedMemberID)
	if err != nil {
		return PromoteMemberResponse{}, err
	}

	if roomid != params.RoomID {
		return PromoteMemberResponse{}, ErrMemberNotFound
	}

	if err := s.roomRepo.UpdateMemberIsAdmin(ctx, params.PromotedMemberID, true); err != nil {
		return PromoteMemberResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		return PromoteMemberResponse{}, err
	}

	return PromoteMemberResponse{
		Conns: conns,
	}, nil
}

type DisconnectMemberParams struct {
	MemberID string
	RoomID   string
}

type DisconnectMemberResponse struct {
	Conns         []*websocket.Conn
	Memberlist    []Member
	IsRoomDeleted bool
}

func (s service) DisconnectMember(ctx context.Context, params *DisconnectMemberParams) (DisconnectMemberResponse, error) {
	s.roomRepo.RemoveMember(ctx, &room.RemoveMemberParams{
		MemberID: params.MemberID,
		RoomID:   params.RoomID,
	})
	s.connRepo.RemoveByMemberID(params.MemberID)

	memberlist, err := s.getMemberList(ctx, params.RoomID)
	if err != nil {
		return DisconnectMemberResponse{}, err
	}

	if len(memberlist) == 0 {
		if err := s.deleteRoom(ctx, params.RoomID); err != nil {
			return DisconnectMemberResponse{}, err
		}

		return DisconnectMemberResponse{
			IsRoomDeleted: true,
		}, nil
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		return DisconnectMemberResponse{}, err
	}

	return DisconnectMemberResponse{
		Conns:         conns,
		Memberlist:    memberlist,
		IsRoomDeleted: false,
	}, nil
}
