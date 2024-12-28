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
	RemovedMemberId string
	SenderId        string
	RoomId          string
}

type RemoveMemberResponse struct {
	Conn    *websocket.Conn
	Conns   []*websocket.Conn
	Members []Member
}

func (s service) RemoveMember(ctx context.Context, params *RemoveMemberParams) (RemoveMemberResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return RemoveMemberResponse{}, err
	}

	if err := s.roomRepo.RemoveMember(ctx, &room.RemoveMemberParams{
		MemberId: params.RemovedMemberId,
		RoomId:   params.RoomId,
	}); err != nil {
		return RemoveMemberResponse{}, fmt.Errorf("failed to remove member: %w", err)
	}

	if err := s.roomRepo.RemoveMemberFromList(ctx, &room.RemoveMemberFromListParams{
		MemberId: params.RemovedMemberId,
		RoomId:   params.RoomId,
	}); err != nil {
		return RemoveMemberResponse{}, fmt.Errorf("failed to remove member from list: %w", err)
	}

	conn, err := s.connRepo.RemoveByMemberId(params.RemovedMemberId)
	if err != nil {
		return RemoveMemberResponse{}, fmt.Errorf("failed to remove conn: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return RemoveMemberResponse{}, err
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return RemoveMemberResponse{}, err
	}

	return RemoveMemberResponse{
		Conn:    conn,
		Conns:   conns,
		Members: members,
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
		return PromoteMemberResponse{}, err
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
		return PromoteMemberResponse{}, err
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return PromoteMemberResponse{}, err
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

	expireAt := time.Now().Add(s.roomExp)

	if err := s.roomRepo.ExpireMember(ctx, &room.ExpireMemberParams{
		MemberId: params.MemberId,
		RoomId:   params.RoomId,
		ExpireAt: expireAt,
	}); err != nil {
		return DisconnectMemberResponse{}, fmt.Errorf("failed to expire member: %w", err)
	}

	if _, err := s.connRepo.RemoveByMemberId(params.MemberId); err != nil {
		return DisconnectMemberResponse{}, fmt.Errorf("failed to remove conn: %w", err)
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return DisconnectMemberResponse{}, err
	}

	// delete room if no member left
	if len(members) == 0 {
		videoIds, err := s.roomRepo.GetVideoIds(ctx, params.RoomId)
		if err != nil {
			return DisconnectMemberResponse{}, fmt.Errorf("failed to get video ids: %w", err)
		}

		lastVideoId, err := s.roomRepo.GetLastVideoId(ctx, params.RoomId)
		if err != nil {
			return DisconnectMemberResponse{}, fmt.Errorf("failed to get last video id: %w", err)
		}

		if lastVideoId != nil {
			videoIds = append(videoIds, *lastVideoId)
		}

		playerVideoId, err := s.roomRepo.GetPlayerVideoId(ctx, params.RoomId)
		if err != nil {
			return DisconnectMemberResponse{}, fmt.Errorf("failed to get player video id: %w", err)
		}

		videoIds = append(videoIds, playerVideoId)

		for _, videoId := range videoIds {
			if err := s.roomRepo.ExpireVideo(ctx, &room.ExpireVideoParams{
				VideoId:  videoId,
				RoomId:   params.RoomId,
				ExpireAt: expireAt,
			}); err != nil {
				return DisconnectMemberResponse{}, fmt.Errorf("failed to expire video: %w", err)
			}
		}

		if err := s.roomRepo.ExpirePlayer(ctx, &room.ExpirePlayerParams{
			RoomId:   params.RoomId,
			ExpireAt: expireAt,
		}); err != nil {
			return DisconnectMemberResponse{}, fmt.Errorf("failed to expire player: %w", err)
		}

		if err := s.roomRepo.ExpireLastVideo(ctx, &room.ExpireLastVideoParams{
			RoomId:   params.RoomId,
			ExpireAt: expireAt,
		}); err != nil {
			if err != room.ErrLastVideoNotFound {
				return DisconnectMemberResponse{}, fmt.Errorf("failed to expire last video: %w", err)
			}
		}

		if err := s.roomRepo.ExpirePlaylist(ctx, &room.ExpirePlaylistParams{
			RoomId:   params.RoomId,
			ExpireAt: expireAt,
		}); err != nil {
			if err != room.ErrPlaylistNotFound {
				return DisconnectMemberResponse{}, fmt.Errorf("failed to expire playlist: %w", err)
			}
		}

		return DisconnectMemberResponse{
			IsRoomDeleted: true,
		}, nil
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
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
	AvatarUrl o.Field[string]
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

	if params.AvatarUrl.Defined {
		if member.AvatarUrl != params.AvatarUrl.Value {
			if err := s.roomRepo.UpdateMemberAvatarUrl(ctx, params.RoomId, params.SenderId, params.AvatarUrl.Value); err != nil {
				return UpdateProfileResponse{}, fmt.Errorf("failed to update member avatar url: %w", err)
			}
			member.AvatarUrl = params.AvatarUrl.Value
		}
	}

	// todo: fix double get ids
	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return UpdateProfileResponse{}, err
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return UpdateProfileResponse{}, err
	}

	return UpdateProfileResponse{
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
	SenderConn *websocket.Conn
	IsReady    bool
	SenderId   string
	RoomId     string
}

type UpdateIsReadyResponse struct {
	Conns         []*websocket.Conn
	UpdatedMember Member
	Members       []Member
	Player        *Player
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
		members, err := s.getMembers(ctx, params.RoomId)
		if err != nil {
			return UpdateIsReadyResponse{}, err
		}

		return UpdateIsReadyResponse{
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
		return UpdateIsReadyResponse{}, fmt.Errorf("failed to update member is ready: %w", err)
	}

	// todo: fix double get ids
	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return UpdateIsReadyResponse{}, err
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return UpdateIsReadyResponse{}, err
	}

	member.IsReady = params.IsReady

	updatedMember := Member{
		Id:        params.SenderId,
		Username:  member.Username,
		Color:     member.Color,
		AvatarUrl: member.AvatarUrl,
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
		player, err := s.roomRepo.GetPlayer(ctx, params.RoomId)
		if err != nil {
			return UpdateIsReadyResponse{}, fmt.Errorf("failed to get player: %w", err)
		}

		if player.IsPlaying == neededIsReady {
			return UpdateIsReadyResponse{
				Conns:         conns,
				UpdatedMember: updatedMember,
				Members:       members,
			}, nil
		}

		if player.WaitingForReady == neededIsReady {
			player.IsPlaying = neededIsReady
			player.UpdatedAt = int(time.Now().UnixMicro())

			if err := s.roomRepo.UpdatePlayerState(ctx, &room.UpdatePlayerStateParams{
				IsPlaying:       params.IsReady,
				CurrentTime:     player.CurrentTime,
				PlaybackRate:    player.PlaybackRate,
				WaitingForReady: !player.WaitingForReady,
				UpdatedAt:       player.UpdatedAt,
				RoomId:          params.RoomId,
			}); err != nil {
				return UpdateIsReadyResponse{}, fmt.Errorf("failed to update player state: %w", err)
			}

			video, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
				VideoId: player.VideoId,
				RoomId:  params.RoomId,
			})
			if err != nil {
				return UpdateIsReadyResponse{}, fmt.Errorf("failed to get video: %w", err)
			}

			return UpdateIsReadyResponse{
				Conns:         conns,
				UpdatedMember: updatedMember,
				Members:       members,
				Player: &Player{
					VideoUrl:     video.Url,
					IsPlaying:    player.IsPlaying,
					CurrentTime:  player.CurrentTime,
					PlaybackRate: player.PlaybackRate,
					UpdatedAt:    player.UpdatedAt,
				},
			}, nil
		}
	}

	return UpdateIsReadyResponse{
		Conns:         conns,
		UpdatedMember: updatedMember,
		Members:       members,
	}, nil
}

type UpdateIsMutedParams struct {
	SenderConn *websocket.Conn
	IsMuted    bool
	SenderId   string
	RoomId     string
}

type UpdateIsMutedResponse struct {
	Conns         []*websocket.Conn
	UpdatedMember Member
	Members       []Member
}

func (s service) UpdateIsMuted(ctx context.Context, params *UpdateIsMutedParams) (UpdateIsMutedResponse, error) {
	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberId: params.SenderId,
		RoomId:   params.RoomId,
	})
	if err != nil {
		return UpdateIsMutedResponse{}, fmt.Errorf("failed to get member: %w", err)
	}

	if member.IsMuted == params.IsMuted {
		members, err := s.getMembers(ctx, params.RoomId)
		if err != nil {
			return UpdateIsMutedResponse{}, err
		}

		return UpdateIsMutedResponse{
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
		return UpdateIsMutedResponse{}, fmt.Errorf("failed to update member is muted: %w", err)
	}

	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return UpdateIsMutedResponse{}, err
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return UpdateIsMutedResponse{}, err
	}

	return UpdateIsMutedResponse{
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
