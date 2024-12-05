package room

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository"
)

type CreateRoomParams struct {
	Username        string
	Color           string
	AvatarURL       string
	InitialVideoURL string
}

type CreateRoomResponse struct {
	AuthToken string
	MemberID  string
	RoomID    string
}

func (s service) CreateRoom(ctx context.Context, params *CreateRoomParams) (CreateRoomResponse, error) {
	slog.Info("Service CreateRoom", "params", params)
	roomID := s.generator.GenerateRandomString(8)

	memberID := uuid.NewString()
	if err := s.roomRepo.SetMember(ctx, &repository.SetMemberParams{
		MemberID:  memberID,
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		IsMuted:   false,
		IsAdmin:   true,
		IsOnline:  false,
		RoomID:    roomID,
	}); err != nil {
		return CreateRoomResponse{}, err
	}

	if err := s.roomRepo.SetPlayer(ctx, &repository.SetPlayerParams{
		CurrentVideoURL: params.InitialVideoURL,
		IsPlaying:       false,
		CurrentTime:     0,
		PlaybackRate:    1,
		UpdatedAt:       time.Now().Unix(),
		RoomID:          roomID,
	}); err != nil {
		return CreateRoomResponse{}, err
	}

	authToken := uuid.NewString()
	if err := s.roomRepo.SetAuthToken(ctx, &repository.SetAuthTokenParams{
		AuthToken: authToken,
		MemberID:  memberID,
	}); err != nil {
		return CreateRoomResponse{}, err
	}

	return CreateRoomResponse{
		AuthToken: authToken,
		MemberID:  memberID,
		RoomID:    roomID,
	}, nil
}

type ConnectMemberParams struct {
	Conn     *websocket.Conn
	MemberID string
}

func (s service) ConnectMember(params *ConnectMemberParams) error {
	return s.connRepo.Add(params.Conn, params.MemberID)
}

// todo: make RoomID optional if AuthToken is provided
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
		if errors.Is(err, repository.ErrAuthTokenNotFound) {
			return Member{}, false, nil
		}

		return Member{}, false, err
	}

	member, err := s.roomRepo.GetMember(ctx, memberID)
	if err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
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
	slog.Info("Service JoinRoom", "params", params)
	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		slog.Info("failed to get conns", "err", err)
		return JoinRoomResponse{}, err
	}

	member, found, err := s.getMemberByAuthToken(ctx, params.RoomID, params.AuthToken)
	if err != nil {
		slog.Info("failed to get member by auth token", "err", err)
		return JoinRoomResponse{}, err
	}

	authToken := params.AuthToken

	if found {
		if err := s.roomRepo.AddMemberToList(ctx, &repository.AddMemberToListParams{
			RoomID:   params.RoomID,
			MemberID: member.ID,
		}); err != nil {
			return JoinRoomResponse{}, err
		}

		if member.Username != params.Username {
			if err := s.roomRepo.UpdateMemberUsername(ctx, member.ID, params.Username); err != nil {
				return JoinRoomResponse{}, err
			}
			member.Username = params.Username
		}

		if member.Color != params.Color {
			if err := s.roomRepo.UpdateMemberColor(ctx, member.ID, params.Color); err != nil {
				return JoinRoomResponse{}, err
			}
			member.Color = params.Color
		}

		if member.AvatarURL != params.AvatarURL {
			if err := s.roomRepo.UpdateMemberAvatarURL(ctx, member.ID, params.AvatarURL); err != nil {
				return JoinRoomResponse{}, err
			}
			member.AvatarURL = params.AvatarURL
		}
	} else {
		memberID := uuid.NewString()
		if err := s.roomRepo.SetMember(ctx, &repository.SetMemberParams{
			MemberID:  memberID,
			Username:  params.Username,
			Color:     params.Color,
			AvatarURL: params.AvatarURL,
			IsMuted:   false,
			IsAdmin:   false,
			IsOnline:  false,
			RoomID:    params.RoomID,
		}); err != nil {
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
		if err := s.roomRepo.SetAuthToken(ctx, &repository.SetAuthTokenParams{
			AuthToken: authToken,
			MemberID:  memberID,
		}); err != nil {
			return JoinRoomResponse{}, err
		}
	}

	memberlist, err := s.getMemberList(ctx, params.RoomID)
	if err != nil {
		slog.Info("failed to get memberlist", "err", err)
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
		slog.Info("failed to get player", "err", err)
		return RoomState{}, err
	}

	memberlist, err := s.getMemberList(ctx, roomID)
	if err != nil {
		slog.Info("failed to get memberlist", "err", err)
		return RoomState{}, err
	}
	playlist, err := s.getPlaylist(ctx, roomID)
	if err != nil {
		slog.Info("failed to get playlist", "err", err)
		return RoomState{}, err
	}

	return RoomState{
		RoomID:     roomID,
		Player:     Player(player),
		MemberList: memberlist,
		Playlist:   playlist,
	}, nil
}
