package room

import (
	"context"
	"errors"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository"
)

var (
	ErrPermissionDenied     = errors.New("permission denied")
	ErrPlaylistLimitReached = errors.New("playlist limit reached")
	ErrRoomNotFound         = errors.New("room not found")
)

type iRoomRepo interface {
	// member
	SetMember(context.Context, *repository.SetMemberParams) error
	GetMember(context.Context, string) (repository.Member, error)
	RemoveMember(context.Context, string) error
	GetMemberRoomId(context.Context, string) (string, error)
	GetMemberIDs(context.Context, string) ([]string, error)
	IsMemberAdmin(context.Context, string) (bool, error)
	// video
	SetVideo(context.Context, *repository.SetVideoParams) error
	GetPlaylist(context.Context, string) ([]string, error)
	GetVideo(context.Context, string) (repository.Video, error)
	GetPlaylistLength(context.Context, string) (int, error)
	// player
	SetPlayer(context.Context, *repository.SetPlayerParams) error
	// room
	SetCreateRoomSession(context.Context, *repository.SetCreateRoomSessionParams) error
	GetCreateRoomSession(context.Context, string) (repository.CreateRoomSession, error)
	SetJoinRoomSession(context.Context, *repository.SetJoinRoomSessionParams) error
	GetJoinRoomSession(context.Context, string) (repository.JoinRoomSession, error)
}

type iConnRepo interface {
	Add(*websocket.Conn, string) error
	RemoveByMemberID(string) error
	RemoveByConn(*websocket.Conn) error
	GetConn(string) (*websocket.Conn, error)
	GetMemberID(*websocket.Conn) (string, error)
}

type service struct {
	roomRepo        iRoomRepo
	connRepo        iConnRepo
	membersLimit    int
	playlistLimit   int
	updatesInterval time.Duration
}

func NewService(redisRepo iRoomRepo, connRepo iConnRepo, updatesInterval time.Duration, membersLimit, playlistLimit int) service {
	return service{
		roomRepo:        redisRepo,
		connRepo:        connRepo,
		membersLimit:    membersLimit,
		playlistLimit:   playlistLimit,
		updatesInterval: updatesInterval,
	}
}
