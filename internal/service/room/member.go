package room

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
	o "github.com/skewb1k/optional"
)

var ErrMemberIsAlreadyAdmin = errors.New("member is already admin")

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
		return RemoveMemberResponse{}, fmt.Errorf("failed to check if member is admin: %w", err)
	}

	if err := s.roomRepo.RemoveMemberFromList(ctx, &room.RemoveMemberFromListParams{
		MemberId: params.RemovedMemberId,
		RoomId:   params.RoomId,
	}); err != nil {
		return RemoveMemberResponse{}, fmt.Errorf("failed to remove member: %w", err)
	}

	conn, err := s.connRepo.GetConn(params.RemovedMemberId)
	if err != nil {
		return RemoveMemberResponse{}, fmt.Errorf("failed to get conn: %w", err)
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
	PromotedMember     Member
	PromotedMemberConn *websocket.Conn
	Members            []Member
	Conns              []*websocket.Conn
}

func (s service) PromoteMember(ctx context.Context, params *PromoteMemberParams) (PromoteMemberResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return PromoteMemberResponse{}, fmt.Errorf("failed to check if member is admin: %w", err)
	}

	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberId: params.PromotedMemberId,
		RoomId:   params.RoomId,
	})
	if err != nil {
		return PromoteMemberResponse{}, fmt.Errorf("failed to get member: %w", err)
	}

	if member.IsAdmin {
		return PromoteMemberResponse{}, ErrMemberIsAlreadyAdmin
	}

	updatedMemberIsAdmin := true
	if err := s.roomRepo.UpdateMemberIsAdmin(ctx, params.RoomId, params.PromotedMemberId, updatedMemberIsAdmin); err != nil {
		return PromoteMemberResponse{}, fmt.Errorf("failed to update member is admin: %w", err)
	}
	member.IsAdmin = updatedMemberIsAdmin

	// todo: refactor by do not use getConnsByRoomId to save conn inside for
	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return PromoteMemberResponse{}, fmt.Errorf("failed to get conns by room id: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return PromoteMemberResponse{}, fmt.Errorf("failed to get member list: %w", err)
	}

	promotedMemberConn, err := s.connRepo.GetConn(params.PromotedMemberId)
	if err != nil {
		return PromoteMemberResponse{}, fmt.Errorf("failed to get conn: %w", err)
	}

	return PromoteMemberResponse{
		Conns:              conns,
		PromotedMemberConn: promotedMemberConn,
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
		return fmt.Errorf("failed to connect member: %w", err)
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
		return DisconnectMemberResponse{}, fmt.Errorf("failed to remove member from list: %w", err)
	}

	conn, err := s.connRepo.RemoveByMemberId(params.MemberId)
	if err != nil {
		return DisconnectMemberResponse{}, fmt.Errorf("failed to remove conn by member id: %w", err)
	}

	conn.Close()

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return DisconnectMemberResponse{}, fmt.Errorf("failed to get member list: %w", err)
	}

	// delete room if no member left
	if len(members) == 0 {
		if err := s.deleteRoom(ctx, params.RoomId); err != nil {
			return DisconnectMemberResponse{}, fmt.Errorf("failed to delete room: %w", err)
		}

		return DisconnectMemberResponse{
			IsRoomDeleted: true,
		}, nil
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return DisconnectMemberResponse{}, fmt.Errorf("failed to get conns by room id: %w", err)
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
		return UpdateProfileResponse{}, fmt.Errorf("failed to get member: %w", err)
	}

	// todo: wrap in transaction
	if params.Username != nil && member.Username != *params.Username {
		if err := s.roomRepo.UpdateMemberUsername(ctx, params.RoomId, params.SenderId, *params.Username); err != nil {
			return UpdateProfileResponse{}, fmt.Errorf("failed to update member username: %w", err)
		}
		member.Username = *params.Username
	}

	if params.Color != nil && member.Color != *params.Color {
		if err := s.roomRepo.UpdateMemberColor(ctx, params.RoomId, params.SenderId, *params.Color); err != nil {
			return UpdateProfileResponse{}, fmt.Errorf("failed to update member color: %w", err)
		}
		member.Color = *params.Color
	}

	if params.AvatarURL.Defined {
		if member.AvatarURL != params.AvatarURL.Value {
			if err := s.roomRepo.UpdateMemberAvatarURL(ctx, params.RoomId, params.SenderId, params.AvatarURL.Value); err != nil {
				return UpdateProfileResponse{}, fmt.Errorf("failed to update member avatar url: %w", err)
			}
			member.AvatarURL = params.AvatarURL.Value
		}
	}

	// todo: fix double get ids
	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return UpdateProfileResponse{}, fmt.Errorf("failed to get conns by room id: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return UpdateProfileResponse{}, fmt.Errorf("failed to get members: %w", err)
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

type UpdateIsReadyParams struct {
	SenderConn *websocket.Conn
	IsReady    bool
	SenderId   string
	RoomId     string
}

type UpdateIsReadyResponse struct {
	Conns         []*websocket.Conn
	UpdatedMember Member
	Members       []Member
	PlayerState   *PlayerState
}

func (s service) UpdateIsReady(ctx context.Context, params *UpdateIsReadyParams) (UpdateIsReadyResponse, error) {
	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberId: params.SenderId,
		RoomId:   params.RoomId,
	})
	if err != nil {
		return UpdateIsReadyResponse{}, fmt.Errorf("failed to get member: %w", err)
	}

	if member.IsReady == params.IsReady {
		member1 := Member{
			Id:        params.SenderId,
			Username:  member.Username,
			Color:     member.Color,
			AvatarURL: member.AvatarURL,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsReady:   member.IsReady,
		}

		return UpdateIsReadyResponse{
			UpdatedMember: member1,
			Members:       []Member{member1},
			Conns:         []*websocket.Conn{params.SenderConn},
		}, nil
	}

	if err := s.roomRepo.UpdateMemberIsReady(ctx, params.RoomId, params.SenderId, params.IsReady); err != nil {
		return UpdateIsReadyResponse{}, fmt.Errorf("failed to update member is ready: %w", err)
	}

	// todo: fix double get ids
	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return UpdateIsReadyResponse{}, fmt.Errorf("failed to get conns by room id: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return UpdateIsReadyResponse{}, fmt.Errorf("failed to get members: %w", err)
	}

	member.IsReady = params.IsReady

	updatedMember := Member{
		Id:        params.SenderId,
		Username:  member.Username,
		Color:     member.Color,
		AvatarURL: member.AvatarURL,
		IsMuted:   member.IsMuted,
		IsAdmin:   member.IsAdmin,
		IsReady:   member.IsReady,
	}

	ok := true
	neededIsReady := members[0].IsReady
	for i := 1; i < len(members); i++ {
		if members[i].IsReady != neededIsReady {
			ok = false
			break
		}
	}

	if ok {
		playerState, err := s.roomRepo.GetPlayer(ctx, params.RoomId)
		if err != nil {
			return UpdateIsReadyResponse{}, fmt.Errorf("failed to get player: %w", err)
		}

		updatePlayerStateParams := room.UpdatePlayerStateParams{
			IsPlaying:    neededIsReady,
			CurrentTime:  playerState.CurrentTime,
			PlaybackRate: playerState.PlaybackRate,
			UpdatedAt:    int(time.Now().Unix()),
			RoomId:       params.RoomId,
		}
		if err := s.roomRepo.UpdatePlayerState(ctx, &updatePlayerStateParams); err != nil {
			return UpdateIsReadyResponse{}, fmt.Errorf("failed to update player state: %w", err)
		}

		return UpdateIsReadyResponse{
			Conns:         conns,
			UpdatedMember: updatedMember,
			Members:       members,
			PlayerState: &PlayerState{
				IsPlaying:    updatePlayerStateParams.IsPlaying,
				CurrentTime:  playerState.CurrentTime,
				PlaybackRate: playerState.PlaybackRate,
				UpdatedAt:    playerState.UpdatedAt,
			},
		}, nil
	}

	return UpdateIsReadyResponse{
		Conns:         conns,
		UpdatedMember: updatedMember,
		Members:       members,
	}, nil
}
