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
		member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
			MemberID: memberID,
			RoomID:   roomID,
		})
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
	if err := s.checkIfMemberAdmin(ctx, params.RoomID, params.SenderID); err != nil {
		return RemoveMemberResponse{}, err
	}

	if err := s.roomRepo.RemoveMemberFromList(ctx, &room.RemoveMemberFromListParams{
		MemberID: params.RemovedMemberID,
		RoomID:   params.RoomID,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to remove member", "error", err)
		return RemoveMemberResponse{}, err
	}

	conn, err := s.connRepo.GetConn(params.RemovedMemberID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conn", "error", err)
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
	if err := s.checkIfMemberAdmin(ctx, params.RoomID, params.SenderID); err != nil {
		return PromoteMemberResponse{}, err
	}

	if err := s.roomRepo.UpdateMemberIsAdmin(ctx, params.RoomID, params.PromotedMemberID, true); err != nil {
		s.logger.InfoContext(ctx, "failed to promote member", "error", err)
		return PromoteMemberResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return PromoteMemberResponse{}, err
	}

	return PromoteMemberResponse{
		Conns: conns,
	}, nil
}

type ConnectMemberParams struct {
	Conn     *websocket.Conn
	MemberID string
}

func (s service) ConnectMember(ctx context.Context, params *ConnectMemberParams) error {
	if err := s.connRepo.Add(params.Conn, params.MemberID); err != nil {
		s.logger.InfoContext(ctx, "failed to connect member", "error", err)
		return err
	}

	return nil
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
	if err := s.roomRepo.RemoveMemberFromList(ctx, &room.RemoveMemberFromListParams{
		MemberID: params.MemberID,
		RoomID:   params.RoomID,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to remove member", "error", err)
	}

	conn, err := s.connRepo.RemoveByMemberID(params.MemberID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to remove conn", "error", err)
	}
	//! for testing
	if conn.NetConn() != nil {
		conn.Close()
	}

	memberlist, err := s.getMemberList(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return DisconnectMemberResponse{}, err
	}

	if len(memberlist) == 0 {
		if err := s.deleteRoom(ctx, params.RoomID); err != nil {
			s.logger.InfoContext(ctx, "failed to delete room", "error", err)
			return DisconnectMemberResponse{}, err
		}

		return DisconnectMemberResponse{
			IsRoomDeleted: true,
		}, nil
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return DisconnectMemberResponse{}, err
	}

	return DisconnectMemberResponse{
		Conns:         conns,
		Memberlist:    memberlist,
		IsRoomDeleted: false,
	}, nil
}
