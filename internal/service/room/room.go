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
	s.roomRepo.RemovePlayer(ctx, roomID)
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

// todo: also return room state
type CreateRoomResponse struct {
	AuthToken string
	MemberID  string
	RoomID    string
}

func (s service) CreateRoom(ctx context.Context, params *CreateRoomParams) (CreateRoomResponse, error) {
	roomID := s.generator.GenerateRandomString(8)

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
		return CreateRoomResponse{}, err
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
		return CreateRoomResponse{}, err
	}

	authToken := uuid.NewString()
	if err := s.roomRepo.SetAuthToken(ctx, &room.SetAuthTokenParams{
		AuthToken: authToken,
		MemberID:  memberID,
	}); err != nil {
		s.logger.InfoContext(ctx, "failed to set auth token", "error", err)
		return CreateRoomResponse{}, err
	}

	return CreateRoomResponse{
		AuthToken: authToken,
		MemberID:  memberID,
		RoomID:    roomID,
	}, nil
}

type JoinRoomParams struct {
	Username  string
	Color     string
	AvatarURL string
	AuthToken string
	RoomID    string
}

type JoinRoomResponse struct {
	AuthToken    string
	JoinedMember Member
	MemberList   []Member
	Conns        []*websocket.Conn
}

func (s service) getMemberByAuthToken(ctx context.Context, roomID, authToken string) (Member, bool, error) {
	memberID, err := s.roomRepo.GetMemberIDByAuthToken(ctx, authToken)
	if err != nil {
		if errors.Is(err, room.ErrAuthTokenNotFound) {
			return Member{}, false, nil
		}

		return Member{}, false, err
	}

	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		MemberID: memberID,
		RoomID:   roomID,
	})
	if err != nil {
		if errors.Is(err, room.ErrMemberNotFound) {
			return Member{}, false, nil
		}

		return Member{}, false, err
	}

	if member.RoomID != roomID {
		return Member{}, false, nil
	}

	return Member{
		ID:        memberID,
		Username:  member.Username,
		Color:     member.Color,
		AvatarURL: member.AvatarURL,
		IsMuted:   member.IsMuted,
		IsAdmin:   member.IsAdmin,
		IsOnline:  member.IsOnline,
	}, true, nil
}

func (s service) JoinRoom(ctx context.Context, params *JoinRoomParams) (JoinRoomResponse, error) {
	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get conns", "error", err)
		return JoinRoomResponse{}, err
	}

	member, found, err := s.getMemberByAuthToken(ctx, params.RoomID, params.AuthToken)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member by auth token", "error", err)
		return JoinRoomResponse{}, err
	}

	authToken := params.AuthToken

	if found {
		if err := s.roomRepo.AddMemberToList(ctx, &room.AddMemberToListParams{
			RoomID:   params.RoomID,
			MemberID: member.ID,
		}); err != nil {
			s.logger.InfoContext(ctx, "failed to add member to list", "error", err)
			return JoinRoomResponse{}, err
		}

		if member.Username != params.Username {
			if err := s.roomRepo.UpdateMemberUsername(ctx, params.RoomID, member.ID, params.Username); err != nil {
				s.logger.InfoContext(ctx, "failed to update member username", "error", err)
				return JoinRoomResponse{}, err
			}
			member.Username = params.Username
		}

		if member.Color != params.Color {
			if err := s.roomRepo.UpdateMemberColor(ctx, params.RoomID, member.ID, params.Color); err != nil {
				s.logger.InfoContext(ctx, "failed to update member color", "error", err)
				return JoinRoomResponse{}, err
			}
			member.Color = params.Color
		}

		if member.AvatarURL != params.AvatarURL {
			if err := s.roomRepo.UpdateMemberAvatarURL(ctx, params.RoomID, member.ID, params.AvatarURL); err != nil {
				s.logger.InfoContext(ctx, "failed to update member avatar URL", "error", err)
				return JoinRoomResponse{}, err
			}
			member.AvatarURL = params.AvatarURL
		}
	} else {
		playerExists, err := s.roomRepo.IsPlayerExists(ctx, params.RoomID)
		if err != nil {
			s.logger.InfoContext(ctx, "failed to check if player exists", "error", err)
			return JoinRoomResponse{}, err
		}

		if !playerExists {
			s.logger.InfoContext(ctx, "room not found")
			return JoinRoomResponse{}, errors.New("room not found")
		}

		memberID := uuid.NewString()
		if err := s.roomRepo.SetMember(ctx, &room.SetMemberParams{
			MemberID:  memberID,
			Username:  params.Username,
			Color:     params.Color,
			AvatarURL: params.AvatarURL,
			IsMuted:   false,
			IsAdmin:   false,
			IsOnline:  false,
			RoomID:    params.RoomID,
		}); err != nil {
			s.logger.InfoContext(ctx, "failed to set member", "error", err)
			return JoinRoomResponse{}, err
		}
		member = Member{
			ID:        memberID,
			Username:  params.Username,
			Color:     params.Color,
			AvatarURL: params.AvatarURL,
			IsMuted:   false,
			IsAdmin:   false,
			IsOnline:  false,
		}

		authToken = uuid.NewString()
		if err := s.roomRepo.SetAuthToken(ctx, &room.SetAuthTokenParams{
			AuthToken: authToken,
			MemberID:  memberID,
		}); err != nil {
			s.logger.InfoContext(ctx, "failed to set auth token", "error", err)
			return JoinRoomResponse{}, err
		}
	}

	memberlist, err := s.getMemberList(ctx, params.RoomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return JoinRoomResponse{}, err
	}

	return JoinRoomResponse{
		AuthToken:    authToken,
		Conns:        conns,
		MemberList:   memberlist,
		JoinedMember: member,
	}, nil
}

func (s service) GetRoomState(ctx context.Context, roomID string) (RoomState, error) {
	player, err := s.roomRepo.GetPlayer(ctx, roomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get player", "error", err)
		return RoomState{}, err
	}

	memberlist, err := s.getMemberList(ctx, roomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get member list", "error", err)
		return RoomState{}, err
	}

	videos, err := s.getVideos(ctx, roomID)
	if err != nil {
		s.logger.InfoContext(ctx, "failed to get videos", "error", err)
		return RoomState{}, err
	}

	return RoomState{
		RoomID:     roomID,
		Player:     Player(player),
		MemberList: memberlist,
		Playlist: Playlist{
			Videos: videos,
		},
	}, nil
}
