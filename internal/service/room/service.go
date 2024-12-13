package room

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
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
	SetMember(context.Context, *room.SetMemberParams) error
	AddMemberToList(context.Context, *room.AddMemberToListParams) error
	RemoveMember(context.Context, *room.RemoveMemberParams) error
	RemoveMemberFromList(context.Context, *room.RemoveMemberFromListParams) error
	GetMember(context.Context, *room.GetMemberParams) (room.Member, error)
	GetMemberIDs(context.Context, string) ([]string, error)
	GetMemberIsAdmin(ctx context.Context, roomID string, memberID string) (bool, error)
	UpdateMemberIsAdmin(ctx context.Context, roomID string, memberID string, isAdmin bool) error
	UpdateMemberIsMuted(ctx context.Context, roomID string, memberID string, isMuted bool) error
	UpdateMemberIsOnline(ctx context.Context, roomID string, memberID string, isOnline bool) error
	UpdateMemberUsername(ctx context.Context, roomID string, memberID string, username string) error
	UpdateMemberColor(ctx context.Context, roomID string, memberID string, color string) error
	UpdateMemberAvatarURL(ctx context.Context, roomID string, memberID string, avatarURL string) error
	// video
	SetVideo(context.Context, *room.SetVideoParams) error
	RemoveVideo(context.Context, *room.RemoveVideoParams) error
	GetVideoIDs(context.Context, string) ([]string, error)
	GetVideo(context.Context, *room.GetVideoParams) (room.Video, error)
	GetVideosLength(context.Context, string) (int, error)
	GetPreviousVideoID(context.Context, string) (string, error)
	// player
	SetPlayer(context.Context, *room.SetPlayerParams) error
	GetPlayer(context.Context, string) (room.Player, error)
	IsPlayerExists(context.Context, string) (bool, error)
	RemovePlayer(context.Context, string) error
	UpdatePlayer(context.Context, *room.UpdatePlayerParams) error
	UpdatePlayerState(context.Context, *room.UpdatePlayerStateParams) error
}

type iConnRepo interface {
	Add(*websocket.Conn, string) error
	RemoveByMemberID(string) (*websocket.Conn, error)
	RemoveByConn(*websocket.Conn) (string, error)
	GetConn(string) (*websocket.Conn, error)
	GetMemberID(*websocket.Conn) (string, error)
}

type iGenerator interface {
	GenerateRandomString(length int) string
}

type service struct {
	roomRepo      iRoomRepo
	connRepo      iConnRepo
	generator     iGenerator
	membersLimit  int
	playlistLimit int
	secret        []byte
	logger        *slog.Logger
}

func NewService(redisRepo iRoomRepo, connRepo iConnRepo, membersLimit, playlistLimit int, secret string, logger *slog.Logger) *service {
	s := service{
		roomRepo:      redisRepo,
		connRepo:      connRepo,
		membersLimit:  membersLimit,
		playlistLimit: playlistLimit,
		secret:        []byte(secret),
		logger:        logger,
	}

	letterBytes := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	s.generator = randstr.New(letterBytes)

	return &s
}
