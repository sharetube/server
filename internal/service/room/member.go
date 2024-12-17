package room

import (
	"context"
	"errors"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
	o "github.com/skewb1k/optional"
)

func (s service) getMembers(ctx context.Context, roomId string) ([]Member, error) {
	memberIds, err := s.roomRepo.GetMemberIds(ctx, roomId)
	if err != nil {
		return []Member{}, err
	}

	members := make([]Member, 0, len(memberIds))
	for _, memberId := range memberIds {
		member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
			MemberId: memberId,
			RoomId:   roomId,
		})
		if err != nil {
			return []Member{}, err
		}

		members = append(members, Member{
			Id:        memberId,
			Username:  member.Username,
			Color:     member.Color,
			AvatarURL: member.AvatarURL,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsReady:   member.IsReady,
		})
	}

	return members, nil
}

type RemoveMemberParams struct {
	RemovedMemberId string
	SenderId        string
	RoomId          string
}

type RemoveMemberResponse struct {
	Conn *websocket.Conn
}

func (s service) RemoveMember(ctx context.Context, params *RemoveMemberParams) (RemoveMemberResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return RemoveMemberResponse{}, err
	}

	if err := s.roomRepo.RemoveMemberFromList(ctx, &room.RemoveMemberFromListParams{
		MemberId: params.RemovedMemberId,
		RoomId:   params.RoomId,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to remove member", "error", err)
		return RemoveMemberResponse{}, err
	}

	conn, err := s.connRepo.GetConn(params.RemovedMemberId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conn", "error", err)
		return RemoveMemberResponse{}, err
	}

	return RemoveMemberResponse{
		Conn: conn,
	}, nil
}

type PromoteMemberParams struct {
	PromotedMemberId string
	SenderId         string
	RoomId           string
}

type PromoteMemberResponse struct {
	PromotedMember Member
	Members        []Member
	Conns          []*websocket.Conn
}

func (s service) PromoteMember(ctx context.Context, params *PromoteMemberParams) (PromoteMemberResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return PromoteMemberResponse{}, err
	}

	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberId: params.PromotedMemberId,
		RoomId:   params.RoomId,
	})
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member", "error", err)
		return PromoteMemberResponse{}, err
	}

	if member.IsAdmin {
		s.logger.InfoContext(ctx, "member is already admin", "member_id", params.PromotedMemberId)
		return PromoteMemberResponse{}, errors.New("member is already admin")
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return PromoteMemberResponse{}, err
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return PromoteMemberResponse{}, err
	}

	return PromoteMemberResponse{
		Conns: conns,
		PromotedMember: Member{
			Id:        params.PromotedMemberId,
			Username:  member.Username,
			Color:     member.Color,
			AvatarURL: member.AvatarURL,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsReady:   member.IsReady,
		},
		Members: members,
	}, nil
}

type ConnectMemberParams struct {
	Conn     *websocket.Conn
	MemberId string
}

func (s service) ConnectMember(ctx context.Context, params *ConnectMemberParams) error {
	if err := s.connRepo.Add(params.Conn, params.MemberId); err != nil {
		s.logger.InfoContext(ctx, "failed to connect member", "error", err)
		return err
	}

	return nil
}

type DisconnectMemberParams struct {
	MemberId string
	RoomId   string
}

type DisconnectMemberResponse struct {
	Conns         []*websocket.Conn
	Members       []Member
	IsRoomDeleted bool
}

func (s service) DisconnectMember(ctx context.Context, params *DisconnectMemberParams) (DisconnectMemberResponse, error) {
	if err := s.roomRepo.RemoveMemberFromList(ctx, &room.RemoveMemberFromListParams{
		MemberId: params.MemberId,
		RoomId:   params.RoomId,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to remove member", "error", err)
	}

	conn, err := s.connRepo.RemoveByMemberId(params.MemberId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to remove conn", "error", err)
	}
	//! for testing
	if conn.NetConn() != nil {
		conn.Close()
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return DisconnectMemberResponse{}, err
	}

	// delete room if no member left
	if len(members) == 0 {
		if err := s.deleteRoom(ctx, params.RoomId); err != nil {
			s.logger.InfoContext(ctx, "failed to delete room", "error", err)
			return DisconnectMemberResponse{}, err
		}

		return DisconnectMemberResponse{
			IsRoomDeleted: true,
		}, nil
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return DisconnectMemberResponse{}, err
	}

	return DisconnectMemberResponse{
		Conns:         conns,
		Members:       members,
		IsRoomDeleted: false,
	}, nil
}

type UpdateProfileParams struct {
	Username  *string
	Color     *string
	AvatarURL o.Field[string]
	SenderId  string
	RoomId    string
}

type UpdateProfileResponse struct {
	Conns         []*websocket.Conn
	UpdatedMember Member
	Members       []Member
}

func (s service) UpdateProfile(ctx context.Context, params *UpdateProfileParams) (UpdateProfileResponse, error) {
	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberId: params.SenderId,
		RoomId:   params.RoomId,
	})
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member", "error", err)
		return UpdateProfileResponse{}, err
	}

	// todo: wrap in transaction
	if params.Username != nil && member.Username != *params.Username {
		if err := s.roomRepo.UpdateMemberUsername(ctx, params.RoomId, params.SenderId, *params.Username); err != nil {
			s.logger.InfoContext(ctx, "failed to update member username", "error", err)
			return UpdateProfileResponse{}, err
		}
		member.Username = *params.Username
	}

	if params.Color != nil && member.Color != *params.Color {
		if err := s.roomRepo.UpdateMemberColor(ctx, params.RoomId, params.SenderId, *params.Color); err != nil {
			s.logger.InfoContext(ctx, "failed to update member color", "error", err)
			return UpdateProfileResponse{}, err
		}
		member.Color = *params.Color
	}

	if params.AvatarURL.Defined {
		if member.AvatarURL != params.AvatarURL.Value {
			if err := s.roomRepo.UpdateMemberAvatarURL(ctx, params.RoomId, params.SenderId, params.AvatarURL.Value); err != nil {
				s.logger.InfoContext(ctx, "failed to update member avatar url", "error", err)
				return UpdateProfileResponse{}, err
			}
			member.AvatarURL = params.AvatarURL.Value
		}
	}

	// todo: fix double get ids
	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns by room id", "error", err)
		return UpdateProfileResponse{}, err
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return UpdateProfileResponse{}, err
	}

	return UpdateProfileResponse{
		Conns: conns,
		UpdatedMember: Member{
			Id:        params.SenderId,
			Username:  member.Username,
			Color:     member.Color,
			AvatarURL: member.AvatarURL,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsReady:   member.IsReady,
		},
		Members: members,
	}, nil
}
