package room

import (
	"context"
	"errors"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
	o "github.com/skewb1k/optional"
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
	PromotedMember Member
	Members        []Member
	Conns          []*websocket.Conn
}

func (s service) PromoteMember(ctx context.Context, params *PromoteMemberParams) (PromoteMemberResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomID, params.SenderID); err != nil {
		return PromoteMemberResponse{}, err
	}

	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberID: params.PromotedMemberID,
		RoomID:   params.RoomID,
	})
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member", "error", err)
		return PromoteMemberResponse{}, err
	}

	if member.IsAdmin {
		s.logger.InfoContext(ctx, "member is already admin", "member_id", params.PromotedMemberID)
		return PromoteMemberResponse{}, errors.New("member is already admin")
	}

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return PromoteMemberResponse{}, err
	}

	members, err := s.getMemberList(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return PromoteMemberResponse{}, err
	}

	return PromoteMemberResponse{
		Conns: conns,
		PromotedMember: Member{
			ID:        params.PromotedMemberID,
			Username:  member.Username,
			Color:     member.Color,
			AvatarURL: member.AvatarURL,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsOnline:  member.IsOnline,
		},
		Members: members,
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

	// delete room if no member left
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

type UpdateProfileParams struct {
	Username  *string
	Color     *string
	AvatarURL o.Field[string]
	SenderID  string
	RoomID    string
}

type UpdateProfileResponse struct {
	Conns         []*websocket.Conn
	UpdatedMember Member
	Members       []Member
}

func (s service) UpdateProfile(ctx context.Context, params *UpdateProfileParams) (UpdateProfileResponse, error) {
	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberID: params.SenderID,
		RoomID:   params.RoomID,
	})
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member", "error", err)
		return UpdateProfileResponse{}, err
	}

	// todo: wrap in transaction
	if params.Username != nil && member.Username != *params.Username {
		if err := s.roomRepo.UpdateMemberUsername(ctx, params.RoomID, params.SenderID, *params.Username); err != nil {
			s.logger.InfoContext(ctx, "failed to update member username", "error", err)
			return UpdateProfileResponse{}, err
		}
		member.Username = *params.Username
	}

	if params.Color != nil && member.Color != *params.Color {
		if err := s.roomRepo.UpdateMemberColor(ctx, params.RoomID, params.SenderID, *params.Color); err != nil {
			s.logger.InfoContext(ctx, "failed to update member color", "error", err)
			return UpdateProfileResponse{}, err
		}
		member.Color = *params.Color
	}

	if params.AvatarURL.Defined {
		if member.AvatarURL != params.AvatarURL.Value {
			if err := s.roomRepo.UpdateMemberAvatarURL(ctx, params.RoomID, params.SenderID, params.AvatarURL.Value); err != nil {
				s.logger.InfoContext(ctx, "failed to update member avatar url", "error", err)
				return UpdateProfileResponse{}, err
			}
			member.AvatarURL = params.AvatarURL.Value
		}
	}

	// todo: fix double get ids
	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return UpdateProfileResponse{}, err
	}

	members, err := s.getMemberList(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return UpdateProfileResponse{}, err
	}

	return UpdateProfileResponse{
		Conns: conns,
		UpdatedMember: Member{
			ID:        params.SenderID,
			Username:  member.Username,
			Color:     member.Color,
			AvatarURL: member.AvatarURL,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsOnline:  member.IsOnline,
		},
		Members: members,
	}, nil
}
