package room

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository"
	"github.com/sharetube/server/pkg/random"
)

type CreateRoomParams struct {
	Username        string
	Color           string
	AvatarURL       string
	InitialVideoURL string
}

type CreateRoomResponse struct {
	MemberID string
	RoomID   string
}

func (s service) CreateRoom(ctx context.Context, params *CreateRoomParams) (CreateRoomResponse, error) {
	roomID := random.GenerateRandomString(8)

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

	return CreateRoomResponse{
		MemberID: memberID,
		RoomID:   roomID,
	}, nil
}

type ConnectMemberParams struct {
	Conn     *websocket.Conn
	MemberID string
}

func (s service) ConnectMember(params *ConnectMemberParams) error {
	return s.connRepo.Add(params.Conn, params.MemberID)
}

type JoinRoomParams struct {
	Username  string
	Color     string
	AvatarURL string
	RoomID    string
}

type JoinRoomResponse struct {
	JoinedMember Member
	MemberList   []Member
	Conns        []*websocket.Conn
}

func (s service) JoinRoom(ctx context.Context, params *JoinRoomParams) (JoinRoomResponse, error) {
	slog.Info("joining room", "params", params)

	conns, err := s.getConnsByRoomID(ctx, params.RoomID)
	if err != nil {
		slog.Info("failed to get conns", "err", err)
		return JoinRoomResponse{}, err
	}

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

	memberlist, err := s.getMemberList(ctx, params.RoomID)
	if err != nil {
		slog.Info("failed to get memberlist", "err", err)
		return JoinRoomResponse{}, err
	}

	return JoinRoomResponse{
		Conns:      conns,
		MemberList: memberlist,
		JoinedMember: Member{
			ID:        memberID,
			Username:  params.Username,
			Color:     params.Color,
			AvatarURL: params.AvatarURL,
			IsMuted:   false,
			IsAdmin:   false,
			IsOnline:  false,
		},
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
