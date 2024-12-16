package room

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getConnsByRoomID(ctx context.Context, roomID string) ([]*websocket.Conn, error) {
	memberIDs, err := s.roomRepo.GetMemberIDs(ctx, roomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member ids", "error", err)
		return nil, err
	}

	if len(memberIDs) == 0 {
		return nil, errors.New("room does not exist")
	}

	conns := make([]*websocket.Conn, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		conn, err := s.connRepo.GetConn(memberID)
		if err != nil {
			s.logger.InfoContext(ctx, "failed to get conn", "error", err)
			return nil, err
		}

		conns = append(conns, conn)
	}

	return conns, nil
}

func (s service) deleteRoom(ctx context.Context, roomID string) error {
	if err := s.roomRepo.RemovePlayer(ctx, roomID); err != nil {
		s.logger.InfoContext(ctx, "failed to remove player", "error", err)
	}

	videoIDs, err := s.roomRepo.GetVideoIDs(ctx, roomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get video ids", "error", err)
		return err
	}

	for _, videoID := range videoIDs {
		if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
			VideoID: videoID,
			RoomID:  roomID,
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
	AvatarURL       string
	InitialVideoURL string
}

type CreateRoomResponse struct {
	RoomID   string
	MemberID string
	JWT      string
}

func (s service) CreateRoom(ctx context.Context, params *CreateRoomParams) (*CreateRoomResponse, error) {
	roomID := s.generator.GenerateRandomString(8)

	// todo: check if member already exists in room and update if exists instead of creating new one
	// member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
	// 	MemberID: params.MemberID,
	// 	RoomID:   roomID,
	// })
	// if err != nil {
	// 	if errors.Is(err, room.ErrMemberNotFound) {

	// 	}
	// 	s.logger.InfoContext(ctx, "failed to get member", "error", err)
	// 	return CreateRoomResponse{}, err
	// } else {
	// 	if err := s.roomRepo.RemoveMember(ctx, &room.RemoveMemberParams{
	// 		MemberID: params.MemberID,
	// 		RoomID:   roomID,
	// 	});
	// }

	memberID := uuid.NewString()
	if err := s.roomRepo.SetMember(ctx, &room.SetMemberParams{
		MemberID:  memberID,
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		IsMuted:   false,
		IsAdmin:   true,
		IsOnline:  false,
		RoomID:    roomID,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to set member", "error", err)
		return nil, err
	}
	jwt, err := s.generateJWT(memberID)
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
		RoomID:          roomID,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to set player", "error", err)
		return nil, err
	}

	return &CreateRoomResponse{
		JWT:      jwt,
		RoomID:   roomID,
		MemberID: memberID,
	}, nil
}

type JoinRoomParams struct {
	JWT       string
	Username  string
	Color     string
	AvatarURL string
	RoomID    string
}

type JoinRoomResponse struct {
	JWT          string
	JoinedMember Member
	MemberList   []Member
	Conns        []*websocket.Conn
}

func (s service) getMemberByJWT(ctx context.Context, jwt string) (string, *room.Member, error) {
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
		MemberID: claims.MemberID,
	})
	if err != nil {
		if errors.Is(err, room.ErrMemberNotFound) {
			return "", nil, nil
		}

		s.logger.InfoContext(ctx, "failed to get member", "error", err)
		return "", nil, err
	}

	return claims.MemberID, &member, nil
}

func (s service) JoinRoom(ctx context.Context, params *JoinRoomParams) (JoinRoomResponse, error) {
	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns", "error", err)
		return JoinRoomResponse{}, err
	}

	jwt := params.JWT
	memberID, member, err := s.getMemberByJWT(ctx, params.JWT)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member", "error", err)
		return JoinRoomResponse{}, err
	}
	if member == nil {
		// member not found, creating new one
		memberID = uuid.NewString()
		setMemberParams := room.SetMemberParams{
			MemberID:  memberID,
			Username:  params.Username,
			Color:     params.Color,
			AvatarURL: params.AvatarURL,
			IsMuted:   false,
			IsAdmin:   false,
			IsOnline:  false,
			RoomID:    params.RoomID,
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
			IsOnline:  false,
			RoomID:    params.RoomID,
		}
		jwt, err = s.generateJWT(memberID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to generate jwt", "error", err)
			return JoinRoomResponse{}, err
		}
	} else {
		// member found, updating
		if err := s.roomRepo.AddMemberToList(ctx, &room.AddMemberToListParams{
			RoomID:   params.RoomID,
			MemberID: memberID,
		}); err != nil {
			s.logger.InfoContext(ctx, "failed to add member to list", "error", err)
			return JoinRoomResponse{}, err
		}

		if member.Username != params.Username {
			if err := s.roomRepo.UpdateMemberUsername(ctx, params.RoomID, memberID, params.Username); err != nil {
				s.logger.InfoContext(ctx, "failed to update member username", "error", err)
				return JoinRoomResponse{}, err
			}
			member.Username = params.Username
		}

		if member.Color != params.Color {
			if err := s.roomRepo.UpdateMemberColor(ctx, params.RoomID, memberID, params.Color); err != nil {
				s.logger.InfoContext(ctx, "failed to update member color", "error", err)
				return JoinRoomResponse{}, err
			}
			member.Color = params.Color
		}

		if member.AvatarURL != params.AvatarURL {
			if err := s.roomRepo.UpdateMemberAvatarURL(ctx, params.RoomID, memberID, params.AvatarURL); err != nil {
				s.logger.InfoContext(ctx, "failed to update member avatar URL", "error", err)
				return JoinRoomResponse{}, err
			}
			member.AvatarURL = params.AvatarURL
		}
	}

	memberlist, err := s.getMemberList(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return JoinRoomResponse{}, err
	}

	return JoinRoomResponse{
		JWT:        jwt,
		Conns:      conns,
		MemberList: memberlist,
		JoinedMember: Member{
			ID:        memberID,
			Username:  member.Username,
			Color:     member.Color,
			AvatarURL: member.AvatarURL,
			IsMuted:   member.IsMuted,
			IsAdmin:   member.IsAdmin,
			IsOnline:  member.IsOnline,
		},
	}, nil
}

func (s service) GetRoom(ctx context.Context, roomID string) (Room, error) {
	player, err := s.roomRepo.GetPlayer(ctx, roomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get player", "error", err)
		return Room{}, err
	}

	memberlist, err := s.getMemberList(ctx, roomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return Room{}, err
	}

	videos, err := s.getVideos(ctx, roomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get videos", "error", err)
		return Room{}, err
	}

	return Room{
		RoomID:     roomID,
		Player:     Player(player),
		MemberList: memberlist,
		Playlist: Playlist{
			Videos: videos,
		},
	}, nil
}
