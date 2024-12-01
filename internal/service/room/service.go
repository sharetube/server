package room

import (
	"context"
	"errors"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository"
	"github.com/sharetube/server/pkg/randstr"
)

var (
	ErrPermissionDenied     = errors.New("permission denied")
	ErrMemberNotFound       = errors.New("member not found")
	ErrPlaylistLimitReached = errors.New("playlist limit reached")
	ErrRoomNotFound         = errors.New("room not found")
)

type iRoomRepo interface {
	// member
	SetMember(context.Context, *repository.SetMemberParams) error
	RemoveMember(context.Context, *repository.RemoveMemberParams) error
	GetMember(context.Context, string) (repository.Member, error)
	GetMemberRoomId(context.Context, string) (string, error)
	GetMembersIDs(context.Context, string) ([]string, error)
	IsMemberAdmin(context.Context, string) (bool, error)
	UpdateMemberIsAdmin(context.Context, string, bool) error
	UpdateMemberIsMuted(context.Context, string, bool) error
	UpdateMemberIsOnline(context.Context, string, bool) error
	UpdateMemberUsername(context.Context, string, string) error
	UpdateMemberColor(context.Context, string, string) error
	UpdateMemberAvatarURL(context.Context, string, string) error
	// video
	SetVideo(context.Context, *repository.SetVideoParams) error
	RemoveVideo(context.Context, *repository.RemoveVideoParams) error
	GetVideosIDs(context.Context, string) ([]string, error)
	GetVideo(context.Context, string) (repository.Video, error)
	GetPlaylistLength(context.Context, string) (int, error)
	// player
	SetPlayer(context.Context, *repository.SetPlayerParams) error
	GetPlayer(context.Context, string) (repository.Player, error)
	RemovePlayer(context.Context, string) error
}

type iConnRepo interface {
	Add(*websocket.Conn, string) error
	RemoveByMemberID(string) error
	RemoveByConn(*websocket.Conn) error
	GetConn(string) (*websocket.Conn, error)
	GetMemberID(*websocket.Conn) (string, error)
}

type iGenerator interface {
	GenerateRandomString(length int) string
}

type service struct {
	roomRepo        iRoomRepo
	connRepo        iConnRepo
	generator       iGenerator
	membersLimit    int
	playlistLimit   int
	updatesInterval time.Duration
}

func NewService(redisRepo iRoomRepo, connRepo iConnRepo, updatesInterval time.Duration, membersLimit, playlistLimit int) *service {
	s := service{
		roomRepo:        redisRepo,
		connRepo:        connRepo,
		membersLimit:    membersLimit,
		playlistLimit:   playlistLimit,
		updatesInterval: updatesInterval,
	}

	letterBytes := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	s.generator = randstr.New(letterBytes)

	return &s
}
