package room

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository"
)

type CreateRoomCreateSessionParams struct {
	Username        string
	Color           string
	AvatarURL       string
	InitialVideoURL string
}

// todo: rename
func (s service) CreateCreateRoomSession(ctx context.Context, params *CreateRoomCreateSessionParams) (string, error) {
	connectToken := uuid.NewString()
	if err := s.roomRepo.SetCreateRoomSession(ctx, &repository.SetCreateRoomSessionParams{
		ID:              connectToken,
		Username:        params.Username,
		Color:           params.Color,
		AvatarURL:       params.AvatarURL,
		InitialVideoURL: params.InitialVideoURL,
	}); err != nil {
		return "", err
	}

	return connectToken, nil
}

type CreateRoomJoinSessionParams struct {
	Username  string
	Color     string
	AvatarURL string
	RoomID    string
}

func (s service) CreateJoinRoomSession(ctx context.Context, params *CreateRoomJoinSessionParams) (string, error) {
	// todo: check room exists
	connectToken := uuid.NewString()
	if err := s.roomRepo.SetJoinRoomSession(ctx, &repository.SetJoinRoomSessionParams{
		ID:        connectToken,
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		RoomID:    params.RoomID,
	}); err != nil {
		return "", err
	}

	return connectToken, nil
}

type CreateRoomParams struct {
	ConnectToken string
	Conn         *websocket.Conn
}

func (s service) CreateRoom(ctx context.Context, params *CreateRoomParams) error {
	roomID := uuid.NewString()
	slog.Info("create room", "roomID", roomID)

	createRoomSession, err := s.roomRepo.GetCreateRoomSession(ctx, params.ConnectToken)
	if err != nil {
		return err
	}

	memberID := uuid.NewString()
	if err := s.roomRepo.SetMember(ctx, &repository.SetMemberParams{
		MemberID:  memberID,
		Username:  createRoomSession.Username,
		Color:     createRoomSession.Color,
		AvatarURL: createRoomSession.AvatarURL,
		IsMuted:   false,
		IsAdmin:   true,
		IsOnline:  false,
		RoomID:    roomID,
	}); err != nil {
		return err
	}

	if err := s.roomRepo.SetPlayer(ctx, &repository.SetPlayerParams{
		CurrentVideoURL: createRoomSession.InitialVideoURL,
		IsPlaying:       false,
		CurrentTime:     0,
		PlaybackRate:    1,
		UpdatedAt:       time.Now().Unix(),
		RoomID:          roomID,
	}); err != nil {
		return err
	}

	if err := s.connRepo.Add(params.Conn, memberID); err != nil {
		return err
	}

	return nil
}

type JoinRoomParams struct {
	ConnectToken string
	Conn         *websocket.Conn
	RoomID       string
}

func (s service) JoinRoom(ctx context.Context, params *JoinRoomParams) error {
	slog.Info("joining room", "params", params)

	joinRoomSession, err := s.roomRepo.GetJoinRoomSession(ctx, params.ConnectToken)
	if err != nil {
		return err
	}

	// if joinRoomSession.RoomID != params.RoomID {
	// 	return errors.New("wrong room id")
	// }

	memberID := uuid.NewString()
	if err := s.roomRepo.SetMember(ctx, &repository.SetMemberParams{
		MemberID:  memberID,
		Username:  joinRoomSession.Username,
		Color:     joinRoomSession.Color,
		AvatarURL: joinRoomSession.AvatarURL,
		IsMuted:   false,
		IsAdmin:   false,
		IsOnline:  false,
		RoomID:    joinRoomSession.RoomID,
	}); err != nil {
		return err
	}

	if err := s.connRepo.Add(params.Conn, memberID); err != nil {
		return err
	}

	return nil
}
