package room

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/redis"
)

var (
	ErrRoomNotFound = errors.New("room not found")
)

// todo: move params structs from redis to repository package
type iRedisRepo interface {
	CreateMember(context.Context, *redis.CreateMemberParams) error
	CreateVideo(context.Context, *redis.CreateVideoParams) error
	CreatePlayer(context.Context, *redis.CreatePlayerParams) error
	CreateConnectToken(context.Context, string, string) error
	GetMemberIDByConnectToken(context.Context, string) (string, error)
	GetMemberRoomId(context.Context, string) (string, error)
}

type iWSRepo interface {
	Add(*websocket.Conn, string) error
	Remove(*websocket.Conn)
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

type CreateRoomParams struct {
	Username        string
	Color           string
	AvatarURL       string
	InitialVideoURL string
}

type CreateRoomResponse struct {
	RoomID       string
	MemberID     string
	ConnectToken string
}

func (s Service) CreateRoom(ctx context.Context, params *CreateRoomParams) (CreateRoomResponse, error) {
	slog.Info("creating room", "params", params)
	roomID := uuid.NewString()

	memberID := uuid.NewString()
	if err := s.redisRepo.CreateMember(ctx, &redis.CreateMemberParams{
		ID:        memberID,
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

	if err := s.redisRepo.CreatePlayer(ctx, &redis.CreatePlayerParams{
		CurrentVideoURL: params.InitialVideoURL,
		IsPlaying:       false,
		CurrentTime:     0,
		PlaybackRate:    1,
		UpdatedAt:       time.Now().Unix(),
		RoomID:          roomID,
	}); err != nil {
		return CreateRoomResponse{}, err
	}

	connectToken := uuid.NewString()
	if err := s.redisRepo.CreateConnectToken(ctx, connectToken, memberID); err != nil {
		return CreateRoomResponse{}, err
	}

	return CreateRoomResponse{
		RoomID:       roomID,
		MemberID:     memberID,
		ConnectToken: connectToken,
	}, nil
}

func (s Service) GetMemberIDByConnectToken(ctx context.Context, connectToken string) (string, error) {
	return s.redisRepo.GetMemberIDByConnectToken(ctx, connectToken)

	//? check if room exists
	// roomID, err := s.redisRepo.GetMemberRoomId(ctx, memberID)
	// if err != nil {
	// 	return err
	// }
}

func (s Service) ConnectMember(ctx context.Context, conn *websocket.Conn, memberID string) error {
	if err := s.wsRepo.Add(conn, memberID); err != nil {
		return err
	}

	return nil
}

func (s Service) AddVideo(ctx context.Context) {

}
