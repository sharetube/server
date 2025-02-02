package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
	"github.com/skewb1k/goutils/optional"
)

var ErrMemberIsAlreadyAdmin = errors.New("member is already admin")

func (s service) mapMembers(ctx context.Context, roomId string, memberIds []string) ([]Member, error) {
	members := make([]Member, 0, len(memberIds))
	for _, memberId := range memberIds {
		member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
			MemberId: memberId,
			RoomId:   roomId,
		})
		if err != nil {
			return []Member{}, fmt.Errorf("failed to get member: %w", err)
		}

		members = append(members, Member{
			Id:        memberId,
			Username:  member.Username,
			Color:     member.Color,
			AvatarUrl: member.AvatarUrl,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsReady:   member.IsReady,
		})
	}

	return members, nil
}

func (s service) getMembers(ctx context.Context, roomId string) ([]Member, error) {
	memberIds, err := s.roomRepo.GetMemberIds(ctx, roomId)
	if err != nil {
		return []Member{}, fmt.Errorf("failed to get member ids: %w", err)
	}

	return s.mapMembers(ctx, roomId, memberIds)
}

type RemoveMemberParams struct {
	RemovedMemberId string `json:"member_id"`
	SenderId        string `json:"sender_id"`
	RoomId          string `json:"room_id"`
}

type RemoveMemberResponse struct {
	Conn    *websocket.Conn
	Conns   []*websocket.Conn
	Members []Member
}

func (s service) RemoveMember(ctx context.Context, params *RemoveMemberParams) (*RemoveMemberResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return nil, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.RemovedMemberId, MemberIdRule...),
	); err != nil {
		return nil, err
	}

	if err := s.roomRepo.RemoveMemberFromList(ctx, &room.RemoveMemberFromListParams{
		MemberId: params.RemovedMemberId,
		RoomId:   params.RoomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to remove member from list: %w", err)
	}

	if err := s.roomRepo.RemoveMember(ctx, &room.RemoveMemberParams{
		MemberId: params.RemovedMemberId,
		RoomId:   params.RoomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to remove member: %w", err)
	}

	conn, err := s.connRepo.RemoveByMemberId(params.RemovedMemberId)
	if err != nil {
		return nil, fmt.Errorf("failed to remove conn: %w", err)
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	return &RemoveMemberResponse{
		Conn:    conn,
		Conns:   conns,
		Members: members,
	}, nil
}

type PromoteMemberParams struct {
	PromotedMemberId string `json:"promoted_member_id"`
	SenderId         string `json:"sender_id"`
	RoomId           string `json:"room_id"`
}

type PromoteMemberResponse struct {
	PromotedMember     Member
	PromotedMemberConn *websocket.Conn
	Members            []Member
	Conns              []*websocket.Conn
}

func (s service) PromoteMember(ctx context.Context, params *PromoteMemberParams) (*PromoteMemberResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return nil, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.PromotedMemberId, MemberIdRule...),
	); err != nil {
		return nil, err
	}

	//? check that member is in list (maybe he was, but disconnected)
	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberId: params.PromotedMemberId,
		RoomId:   params.RoomId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	if member.IsAdmin {
		return nil, ErrMemberIsAlreadyAdmin
	}

	updatedMemberIsAdmin := true
	if err := s.roomRepo.UpdateMemberIsAdmin(ctx, params.RoomId, params.PromotedMemberId, updatedMemberIsAdmin); err != nil {
		return nil, fmt.Errorf("failed to update member is admin: %w", err)
	}
	member.IsAdmin = updatedMemberIsAdmin

	// todo: refactor by do not use getConnsByRoomId to save conn inside for
	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	promotedMemberConn, err := s.connRepo.GetConn(params.PromotedMemberId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conn: %w", err)
	}

	return &PromoteMemberResponse{
		Conns:              conns,
		PromotedMemberConn: promotedMemberConn,
		PromotedMember: Member{
			Id:        params.PromotedMemberId,
			Username:  member.Username,
			Color:     member.Color,
			AvatarUrl: member.AvatarUrl,
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
	Conns              []*websocket.Conn
	Members            []Member
	PromotedMemberConn *websocket.Conn
	IsRoomDeleted      bool
}

func (s service) DisconnectMember(ctx context.Context, params *DisconnectMemberParams) (*DisconnectMemberResponse, error) {
	if err := s.roomRepo.RemoveMemberFromList(ctx, &room.RemoveMemberFromListParams{
		MemberId: params.MemberId,
		RoomId:   params.RoomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to remove member from list: %w", err)
	}

	expireAt := time.Now().Add(s.roomExp)

	//?
	// if err := s.roomRepo.ExpireMember(ctx, &room.ExpireMemberParams{
	// 	MemberId: params.MemberId,
	// 	RoomId:   params.RoomId,
	// 	ExpireAt: expireAt,
	// }); err != nil {
	// 	return nil, fmt.Errorf("failed to expire member: %w", err)
	// }

	if _, err := s.connRepo.RemoveByMemberId(params.MemberId); err != nil {
		return nil, fmt.Errorf("failed to remove conn: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	// delete room if no member left
	if len(members) == 0 {
		videoIds, err := s.roomRepo.GetVideoIds(ctx, params.RoomId)
		if err != nil {
			return nil, fmt.Errorf("failed to get video ids: %w", err)
		}

		lastVideoId, err := s.roomRepo.GetLastVideoId(ctx, params.RoomId)
		if err != nil {
			return nil, fmt.Errorf("failed to get last video id: %w", err)
		}

		if lastVideoId != nil {
			videoIds = append(videoIds, *lastVideoId)
		}

		playerVideoId, err := s.roomRepo.GetCurrentVideoId(ctx, params.RoomId)
		if err != nil {
			return nil, fmt.Errorf("failed to get player video id: %w", err)
		}

		videoIds = append(videoIds, playerVideoId)

		for _, videoId := range videoIds {
			if err := s.roomRepo.ExpireVideo(ctx, &room.ExpireVideoParams{
				VideoId:  videoId,
				RoomId:   params.RoomId,
				ExpireAt: expireAt,
			}); err != nil {
				return nil, fmt.Errorf("failed to expire video: %w", err)
			}
		}

		if err := s.roomRepo.ExpireMembers(ctx, &room.ExpireMembersParams{
			RoomId:   params.RoomId,
			ExpireAt: expireAt,
		}); err != nil {
			return nil, fmt.Errorf("failed to expire members: %w", err)
		}

		if err := s.roomRepo.ExpirePlayer(ctx, &room.ExpirePlayerParams{
			RoomId:   params.RoomId,
			ExpireAt: expireAt,
		}); err != nil {
			return nil, fmt.Errorf("failed to expire player: %w", err)
		}

		if err := s.roomRepo.ExpireVideoEnded(ctx, &room.ExpireVideoEndedParams{
			RoomId:   params.RoomId,
			ExpireAt: expireAt,
		}); err != nil {
			return nil, fmt.Errorf("failed to expire video ended: %w", err)
		}

		if err := s.roomRepo.ExpirePlayerVersion(ctx, &room.ExpirePlayerVersionParams{
			RoomId:   params.RoomId,
			ExpireAt: expireAt,
		}); err != nil {
			return nil, fmt.Errorf("failed to expire player version: %w", err)
		}

		if err := s.roomRepo.ExpireLastVideo(ctx, &room.ExpireLastVideoParams{
			RoomId:   params.RoomId,
			ExpireAt: expireAt,
		}); err != nil {
			if err != room.ErrLastVideoNotFound {
				return nil, fmt.Errorf("failed to expire last video: %w", err)
			}
		}

		if err := s.roomRepo.ExpirePlaylist(ctx, &room.ExpirePlaylistParams{
			RoomId:   params.RoomId,
			ExpireAt: expireAt,
		}); err != nil {
			if err != room.ErrPlaylistNotFound {
				return nil, fmt.Errorf("failed to expire playlist: %w", err)
			}
		}

		return &DisconnectMemberResponse{
			IsRoomDeleted: true,
		}, nil
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	// promote single left member to admin
	if len(members) == 1 && !members[0].IsAdmin {
		if err := s.roomRepo.UpdateMemberIsAdmin(ctx, params.RoomId, members[0].Id, true); err != nil {
			return nil, fmt.Errorf("failed to update member is admin: %w", err)
		}
		members[0].IsAdmin = true

		return &DisconnectMemberResponse{
			PromotedMemberConn: conns[0],
			Conns:              conns,
			Members:            members,
			IsRoomDeleted:      false,
		}, nil
	}

	return &DisconnectMemberResponse{
		Conns:         conns,
		Members:       members,
		IsRoomDeleted: false,
	}, nil
}

type UpdateProfileParams struct {
	Username  *string                `json:"username"`
	Color     *string                `json:"color"`
	AvatarUrl optional.Field[string] `json:"avatar_url"`
	SenderId  string                 `json:"sender_id"`
	RoomId    string                 `json:"room_id"`
}

type UpdateProfileResponse struct {
	Conns         []*websocket.Conn
	UpdatedMember Member
	Members       []Member
}

func (s service) UpdateProfile(ctx context.Context, params *UpdateProfileParams) (*UpdateProfileResponse, error) {
	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.Username, UsernameRule...),
		validation.Field(&params.Color, ColorRule...),
		// todo: add validation
		// validation.Field(&params.AvatarUrl, AvatarUrlRule...),
	); err != nil {
		return nil, err
	}

	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberId: params.SenderId,
		RoomId:   params.RoomId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	// todo: wrap in transaction
	if params.Username != nil && member.Username != *params.Username {
		if err := s.roomRepo.UpdateMemberUsername(ctx, params.RoomId, params.SenderId, *params.Username); err != nil {
			return nil, fmt.Errorf("failed to update member username: %w", err)
		}
		member.Username = *params.Username
	}

	if params.Color != nil && member.Color != *params.Color {
		if err := s.roomRepo.UpdateMemberColor(ctx, params.RoomId, params.SenderId, *params.Color); err != nil {
			return nil, fmt.Errorf("failed to update member color: %w", err)
		}
		member.Color = *params.Color
	}

	if params.AvatarUrl.Defined {
		if member.AvatarUrl != params.AvatarUrl.Value {
			if err := s.roomRepo.UpdateMemberAvatarUrl(ctx, params.RoomId, params.SenderId, params.AvatarUrl.Value); err != nil {
				return nil, fmt.Errorf("failed to update member avatar url: %w", err)
			}
			member.AvatarUrl = params.AvatarUrl.Value
		}
	}

	// todo: fix double get ids
	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	return &UpdateProfileResponse{
		Conns: conns,
		UpdatedMember: Member{
			Id:        params.SenderId,
			Username:  member.Username,
			Color:     member.Color,
			AvatarUrl: member.AvatarUrl,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsReady:   member.IsReady,
		},
		Members: members,
	}, nil
}

type UpdateIsReadyParams struct {
	SenderConn *websocket.Conn `json:"sender_conn"`
	IsReady    bool            `json:"is_ready"`
	SenderId   string          `json:"sender_id"`
	RoomId     string          `json:"room_id"`
}

type UpdateIsReadyResponse struct {
	Conns         []*websocket.Conn
	UpdatedMember Member
	Members       []Member
	Player        *Player
}

func (s service) UpdateIsReady(ctx context.Context, params *UpdateIsReadyParams) (*UpdateIsReadyResponse, error) {
	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberId: params.SenderId,
		RoomId:   params.RoomId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	if member.IsReady == params.IsReady {
		members, err := s.getMembers(ctx, params.RoomId)
		if err != nil {
			return nil, fmt.Errorf("failed to get members: %w", err)
		}

		return &UpdateIsReadyResponse{
			UpdatedMember: Member{
				Id:        params.SenderId,
				Username:  member.Username,
				Color:     member.Color,
				AvatarUrl: member.AvatarUrl,
				IsMuted:   member.IsMuted,
				IsAdmin:   member.IsAdmin,
				IsReady:   member.IsReady,
			},
			Members: members,
			Conns:   []*websocket.Conn{params.SenderConn},
		}, nil
	}

	if err := s.roomRepo.UpdateMemberIsReady(ctx, params.RoomId, params.SenderId, params.IsReady); err != nil {
		return nil, fmt.Errorf("failed to update member is ready: %w", err)
	}

	// todo: fix double get ids
	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	updatedMember := Member{
		Id:        params.SenderId,
		Username:  member.Username,
		Color:     member.Color,
		AvatarUrl: member.AvatarUrl,
		IsMuted:   member.IsMuted,
		IsAdmin:   member.IsAdmin,
		IsReady:   params.IsReady,
	}

	neededIsReady := members[0].IsReady
	if neededIsReady {
		ok := true
		for i := 1; i < len(members); i++ {
			if members[i].IsReady != neededIsReady {
				ok = false
				break
			}
		}

		if ok {
			player, err := s.roomRepo.GetPlayer(ctx, params.RoomId)
			if err != nil {
				return nil, fmt.Errorf("failed to get player: %w", err)
			}

			if player.IsPlaying == neededIsReady {
				return &UpdateIsReadyResponse{
					Conns:         conns,
					UpdatedMember: updatedMember,
					Members:       members,
				}, nil
			}

			if player.WaitingForReady == neededIsReady {
				player.IsPlaying = neededIsReady
				player.UpdatedAt = int(time.Now().UnixMicro())

				// todo: check isEnded
				if err := s.roomRepo.UpdatePlayerWaitingForReady(ctx, params.RoomId, !player.WaitingForReady); err != nil {
					return nil, fmt.Errorf("failed to update player waiting for ready: %w", err)
				}

				if err := s.roomRepo.UpdatePlayerIsPlaying(ctx, params.RoomId, params.IsReady); err != nil {
					return nil, fmt.Errorf("failed to update player is playing: %w", err)
				}

				playerVersion, err := s.roomRepo.IncrPlayerVersion(ctx, params.RoomId)
				if err != nil {
					return nil, fmt.Errorf("failed to incr player version: %w", err)
				}

				isEnded, err := s.roomRepo.GetVideoEnded(ctx, params.RoomId)
				if err != nil {
					return nil, fmt.Errorf("failed to get video ended: %w", err)
				}

				return &UpdateIsReadyResponse{
					Conns:         conns,
					UpdatedMember: updatedMember,
					Members:       members,
					Player: &Player{
						State: PlayerState{
							CurrentTime:  player.CurrentTime,
							IsPlaying:    player.IsPlaying,
							PlaybackRate: player.PlaybackRate,
							UpdatedAt:    player.UpdatedAt,
						},
						IsEnded: isEnded,
						Version: playerVersion,
					},
				}, nil
			}
		}
	}

	return &UpdateIsReadyResponse{
		Conns:         conns,
		UpdatedMember: updatedMember,
		Members:       members,
	}, nil
}

type UpdateIsMutedParams struct {
	SenderConn *websocket.Conn `json:"sender_conn"`
	IsMuted    bool            `json:"is_muted"`
	SenderId   string          `json:"sender_id"`
	RoomId     string          `json:"room_id"`
}

type UpdateIsMutedResponse struct {
	Conns         []*websocket.Conn
	UpdatedMember Member
	Members       []Member
}

func (s service) UpdateIsMuted(ctx context.Context, params *UpdateIsMutedParams) (*UpdateIsMutedResponse, error) {
	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberId: params.SenderId,
		RoomId:   params.RoomId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	if member.IsMuted == params.IsMuted {
		members, err := s.getMembers(ctx, params.RoomId)
		if err != nil {
			return nil, fmt.Errorf("failed to get members: %w", err)
		}

		return &UpdateIsMutedResponse{
			UpdatedMember: Member{
				Id:        params.SenderId,
				Username:  member.Username,
				Color:     member.Color,
				AvatarUrl: member.AvatarUrl,
				IsMuted:   member.IsMuted,
				IsAdmin:   member.IsAdmin,
				IsReady:   member.IsReady,
			},
			Members: members,
			Conns:   []*websocket.Conn{params.SenderConn},
		}, nil
	}

	if err := s.roomRepo.UpdateMemberIsMuted(ctx, params.RoomId, params.SenderId, params.IsMuted); err != nil {
		return nil, fmt.Errorf("failed to update member is muted: %w", err)
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	return &UpdateIsMutedResponse{
		Conns: conns,
		UpdatedMember: Member{
			Id:        params.SenderId,
			Username:  member.Username,
			Color:     member.Color,
			AvatarUrl: member.AvatarUrl,
			IsMuted:   params.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsReady:   member.IsReady,
		},
		Members: members,
	}, nil
}
