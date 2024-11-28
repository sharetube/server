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

var (
	ErrRoomNotFound = errors.New("room not found")
)

// todo: move params structs from redis to repository package
type iRedisRepo interface {
	SetMember(context.Context, *repository.SetMemberParams) error
	SetVideo(context.Context, *repository.SetVideoParams) error
	SetPlayer(context.Context, *repository.SetPlayerParams) error
	SetCreateRoomSession(context.Context, *repository.SetCreateRoomSessionParams) error
	SetJoinRoomSession(context.Context, *repository.SetJoinRoomSessionParams) error
	GetCreateRoomSession(context.Context, string) (repository.CreateRoomSession, error)
	GetMemberRoomId(context.Context, string) (string, error)
	GetMemberIDs(context.Context, string) ([]string, error)
	GetJoinRoomSession(context.Context, string) (repository.JoinRoomSession, error)
}

type iWSRepo interface {
	Add(*websocket.Conn, string) error
	RemoveByMemberID(string) error
	RemoveByConn(*websocket.Conn) error
	GetConn(string) (*websocket.Conn, error)
	GetMemberID(*websocket.Conn) (string, error)
}

type Service struct {
	redisRepo       iRedisRepo
	wsRepo          iWSRepo
	membersLimit    int
	playlistLimit   int
	updatesInterval time.Duration
}

func NewService(redisRepo iRedisRepo, wsRepo iWSRepo, updatesInterval time.Duration, membersLimit, playlistLimit int) Service {
	return Service{
		redisRepo:       redisRepo,
		wsRepo:          wsRepo,
		membersLimit:    membersLimit,
		playlistLimit:   playlistLimit,
		updatesInterval: updatesInterval,
	}
}

type CreateRoomCreateSessionParams struct {
	Username        string
	Color           string
	AvatarURL       string
	InitialVideoURL string
}

// todo: rename
func (s Service) CreateRoomCreateSession(ctx context.Context, params *CreateRoomCreateSessionParams) (string, error) {
	connectToken := uuid.NewString()
	if err := s.redisRepo.SetCreateRoomSession(ctx, &repository.SetCreateRoomSessionParams{
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

func (s Service) CreateRoomJoinSession(ctx context.Context, params *CreateRoomJoinSessionParams) (string, error) {
	// todo: check room exists
	connectToken := uuid.NewString()
	if err := s.redisRepo.SetJoinRoomSession(ctx, &repository.SetJoinRoomSessionParams{
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

func (s Service) CreateRoom(ctx context.Context, params *CreateRoomParams) error {
	roomID := uuid.NewString()
	slog.Info("create room", "roomID", roomID)

	createRoomSession, err := s.redisRepo.GetCreateRoomSession(ctx, params.ConnectToken)
	if err != nil {
		return err
	}

	memberID := uuid.NewString()
	if err := s.redisRepo.SetMember(ctx, &repository.SetMemberParams{
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

	if err := s.redisRepo.SetPlayer(ctx, &repository.SetPlayerParams{
		CurrentVideoURL: createRoomSession.InitialVideoURL,
		IsPlaying:       false,
		CurrentTime:     0,
		PlaybackRate:    1,
		UpdatedAt:       time.Now().Unix(),
		RoomID:          roomID,
	}); err != nil {
		return err
	}

	if err := s.wsRepo.Add(params.Conn, memberID); err != nil {
		return err
	}

	return nil
}

type JoinRoomParams struct {
	ConnectToken string
	Conn         *websocket.Conn
	RoomID       string
}

func (s Service) JoinRoom(ctx context.Context, params *JoinRoomParams) error {
	slog.Info("joining room", "params", params)

	joinRoomSession, err := s.redisRepo.GetJoinRoomSession(ctx, params.ConnectToken)
	if err != nil {
		return err
	}

	// if joinRoomSession.RoomID != params.RoomID {
	// 	return errors.New("wrong room id")
	// }

	memberID := uuid.NewString()
	if err := s.redisRepo.SetMember(ctx, &repository.SetMemberParams{
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

	if err := s.wsRepo.Add(params.Conn, memberID); err != nil {
		return err
	}

	return nil
}

type AddVideoParams struct {
	Conn     *websocket.Conn
	VideoURL string
}

type AddVideoResponse struct {
	VideoID   string
	AddedByID string
	Conns     []*websocket.Conn
}

func (s Service) AddVideo(ctx context.Context, params *AddVideoParams) (AddVideoResponse, error) {
	memberID, err := s.wsRepo.GetMemberID(params.Conn)
	if err != nil {
		slog.Info("failed to get member id", "err", err)
		return AddVideoResponse{}, err
	}

	roomID, err := s.redisRepo.GetMemberRoomId(ctx, memberID)
	if err != nil {
		slog.Info("failed to get room id", "err", err)
		return AddVideoResponse{}, err
	}

	videoID := uuid.NewString()
	if err := s.redisRepo.SetVideo(ctx, &repository.SetVideoParams{
		VideoID:   videoID,
		RoomID:    roomID,
		URL:       params.VideoURL,
		AddedByID: memberID,
	}); err != nil {
		slog.Info("failed to create video", "err", err)
		return AddVideoResponse{}, err
	}

	conns, err := s.getConnsByRoomID(ctx, roomID)
	if err != nil {
		slog.Info("failed to get conns", "err", err)
		return AddVideoResponse{}, err
	}

	return AddVideoResponse{
		VideoID:   videoID,
		AddedByID: memberID,
		Conns:     conns,
	}, nil
}

func (s Service) getConnsByRoomID(ctx context.Context, roomID string) ([]*websocket.Conn, error) {
	memberIDs, err := s.redisRepo.GetMemberIDs(ctx, roomID)
	if err != nil {
		slog.Info("failed to get member ids", "err", err)
		return nil, err
	}

	conns := make([]*websocket.Conn, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		conn, err := s.wsRepo.GetConn(memberID)
		if err != nil {
			slog.Info("failed to get conn", "err", err)
			return nil, err
		}

		conns = append(conns, conn)
	}

	return conns, nil
}
