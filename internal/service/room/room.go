package room

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getConnsByRoomId(ctx context.Context, roomId string) ([]*websocket.Conn, error) {
	memberIds, err := s.roomRepo.GetMemberIds(ctx, roomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member ids", "error", err)
		return nil, err
	}

	if len(memberIds) == 0 {
		return nil, errors.New("room does not exist")
	}

	conns := make([]*websocket.Conn, 0, len(memberIds))
	for _, memberId := range memberIds {
		conn, err := s.connRepo.GetConn(memberId)
		if err != nil {
			s.logger.InfoContext(ctx, "failed to get conn", "error", err)
			return nil, err
		}

		conns = append(conns, conn)
	}

	return conns, nil
}

func (s service) deleteRoom(ctx context.Context, roomId string) error {
	if err := s.roomRepo.RemovePlayer(ctx, roomId); err != nil {
		s.logger.InfoContext(ctx, "failed to remove player", "error", err)
	}

	videoIds, err := s.roomRepo.GetVideoIds(ctx, roomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get video ids", "error", err)
		return err
	}

	for _, videoId := range videoIds {
		if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
			VideoId: videoId,
			RoomId:  roomId,
		}); err != nil {
			s.logger.InfoContext(ctx, "failed to remove video", "error", err)
			return err
		}
	}

	return nil
}

type CreateRoomParams struct {
	Username        string
	Color           string
	AvatarURL       *string
	InitialVideoURL string
}

type CreateRoomResponse struct {
	RoomId       string
	JoinedMember Member
	JWT          string
}

func (s service) CreateRoom(ctx context.Context, params *CreateRoomParams) (*CreateRoomResponse, error) {
	roomId := s.generator.GenerateRandomString(8)

	memberId := uuid.NewString()
	setMemberParams := room.SetMemberParams{
		MemberId:  memberId,
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		IsMuted:   false,
		IsAdmin:   true,
		IsReady:   false,
		RoomId:    roomId,
	}
	if err := s.roomRepo.SetMember(ctx, &setMemberParams); err != nil {
		s.logger.InfoContext(ctx, "failed to set member", "error", err)
		return nil, err
	}

	jwt, err := s.generateJWT(memberId)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate jwt", "error", err)
		return nil, err
	}

	if err := s.roomRepo.SetPlayer(ctx, &room.SetPlayerParams{
		CurrentVideoURL: params.InitialVideoURL,
		IsPlaying:       s.getDefaultPlayerIsPlaying(),
		CurrentTime:     s.getDefaultPlayerCurrentTime(),
		PlaybackRate:    s.getDefaultPlayerPlaybackRate(),
		UpdatedAt:       int(time.Now().Unix()),
		RoomId:          roomId,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to set player", "error", err)
		return nil, err
	}

	return &CreateRoomResponse{
		JWT:    jwt,
		RoomId: roomId,
		JoinedMember: Member{
			Id:        memberId,
			Username:  setMemberParams.Username,
			Color:     setMemberParams.Color,
			AvatarURL: setMemberParams.AvatarURL,
			IsMuted:   setMemberParams.IsMuted,
			IsAdmin:   setMemberParams.IsAdmin,
			IsReady:   setMemberParams.IsReady,
		},
	}, nil
}

func (s service) getMemberByJWT(ctx context.Context, roomId, jwt string) (string, *room.Member, error) {
	if jwt == "" {
		return "", nil, nil
	}

	claims, err := s.parseJWT(jwt)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to parse jwt", "error", err)
		return "", nil, err
	}
	// todo: add validation

	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		RoomId:   roomId,
		MemberId: claims.MemberId,
	})
	if err != nil {
		if errors.Is(err, room.ErrMemberNotFound) {
			return "", nil, nil
		}

		s.logger.InfoContext(ctx, "failed to get member", "error", err)
		return "", nil, err
	}

	return claims.MemberId, &member, nil
}

type JoinRoomParams struct {
	JWT       string
	Username  string
	Color     string
	AvatarURL *string
	RoomId    string
}

type JoinRoomResponse struct {
	JWT          string
	JoinedMember Member
	Members      []Member
	Conns        []*websocket.Conn
}

func (s service) JoinRoom(ctx context.Context, params *JoinRoomParams) (JoinRoomResponse, error) {
	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns", "error", err)
		return JoinRoomResponse{}, err
	}

	jwt := params.JWT
	memberId, member, err := s.getMemberByJWT(ctx, params.RoomId, params.JWT)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member", "error", err)
		return JoinRoomResponse{}, err
	}
	if member == nil {
		// member not found, creating new one
		memberId = uuid.NewString()
		setMemberParams := room.SetMemberParams{
			MemberId:  memberId,
			Username:  params.Username,
			Color:     params.Color,
			AvatarURL: params.AvatarURL,
			IsMuted:   false,
			IsAdmin:   false,
			IsReady:   false,
			RoomId:    params.RoomId,
		}
		if err := s.roomRepo.SetMember(ctx, &setMemberParams); err != nil {
			s.logger.InfoContext(ctx, "failed to set member", "error", err)
			return JoinRoomResponse{}, err
		}

		member = &room.Member{
			Username:  params.Username,
			Color:     params.Color,
			AvatarURL: params.AvatarURL,
			IsMuted:   false,
			IsAdmin:   false,
			IsReady:   false,
		}
		jwt, err = s.generateJWT(memberId)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to generate jwt", "error", err)
			return JoinRoomResponse{}, err
		}
	} else {
		// member found, updating
		if err := s.roomRepo.AddMemberToList(ctx, &room.AddMemberToListParams{
			RoomId:   params.RoomId,
			MemberId: memberId,
		}); err != nil {
			s.logger.InfoContext(ctx, "failed to add member to list", "error", err)
			return JoinRoomResponse{}, err
		}

		if member.Username != params.Username {
			if err := s.roomRepo.UpdateMemberUsername(ctx, params.RoomId, memberId, params.Username); err != nil {
				s.logger.InfoContext(ctx, "failed to update member username", "error", err)
				return JoinRoomResponse{}, err
			}
			member.Username = params.Username
		}

		if member.Color != params.Color {
			if err := s.roomRepo.UpdateMemberColor(ctx, params.RoomId, memberId, params.Color); err != nil {
				s.logger.InfoContext(ctx, "failed to update member color", "error", err)
				return JoinRoomResponse{}, err
			}
			member.Color = params.Color
		}

		if member.AvatarURL != params.AvatarURL {
			if err := s.roomRepo.UpdateMemberAvatarURL(ctx, params.RoomId, memberId, params.AvatarURL); err != nil {
				s.logger.InfoContext(ctx, "failed to update member avatar URL", "error", err)
				return JoinRoomResponse{}, err
			}
			member.AvatarURL = params.AvatarURL
		}
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return JoinRoomResponse{}, err
	}

	return JoinRoomResponse{
		JWT:     jwt,
		Conns:   conns,
		Members: members,
		JoinedMember: Member{
			Id:        memberId,
			Username:  member.Username,
			Color:     member.Color,
			AvatarURL: member.AvatarURL,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsReady:   member.IsReady,
		},
	}, nil
}

func (s service) GetRoom(ctx context.Context, roomId string) (Room, error) {
	player, err := s.roomRepo.GetPlayer(ctx, roomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get player", "error", err)
		return Room{}, err
	}

	members, err := s.getMembers(ctx, roomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return Room{}, err
	}

	playlist, err := s.getPlaylist(ctx, roomId)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get videos", "error", err)
		return Room{}, err
	}

	return Room{
		RoomId:   roomId,
		Player:   Player(player),
		Members:  members,
		Playlist: playlist,
	}, nil
}
