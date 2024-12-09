package room

import (
	"context"
	"errors"

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
	GetMember(context.Context, string) (room.Member, error)
	GetMemberRoomID(context.Context, string) (string, error)
	GetMemberIDs(context.Context, string) ([]string, error)
	IsMemberAdmin(context.Context, string) (bool, error)
	UpdateMemberIsAdmin(ctx context.Context, memberID string, isAdmin bool) error
	UpdateMemberIsMuted(ctx context.Context, memberID string, isMuted bool) error
	UpdateMemberIsOnline(ctx context.Context, memberID string, isOnline bool) error
	UpdateMemberUsername(ctx context.Context, memberID string, username string) error
	UpdateMemberColor(ctx context.Context, memberID string, color string) error
	UpdateMemberAvatarURL(ctx context.Context, memberID string, avatarURL string) error
	// video
	SetVideo(context.Context, *room.SetVideoParams) error
	RemoveVideo(context.Context, *room.RemoveVideoParams) error
	GetVideoIDs(context.Context, string) ([]string, error)
	GetVideo(context.Context, string) (room.Video, error)
	GetPlaylistLength(context.Context, string) (int, error)
	// player
	SetPlayer(context.Context, *room.SetPlayerParams) error
	GetPlayer(context.Context, string) (room.Player, error)
	RemovePlayer(context.Context, string) error
	UpdatePlayerState(context.Context, *room.UpdatePlayerStateParams) error
	UpdatePlayerVideo(ctx context.Context, roomID string, videoURL string) error
	// auth token
	SetAuthToken(context.Context, *room.SetAuthTokenParams) error
	GetMemberIDByAuthToken(context.Context, string) (string, error)
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
}

func NewService(redisRepo iRoomRepo, connRepo iConnRepo, membersLimit, playlistLimit int) *service {
	s := service{
		roomRepo:      redisRepo,
		connRepo:      connRepo,
		membersLimit:  membersLimit,
		playlistLimit: playlistLimit,
	}

	letterBytes := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	s.generator = randstr.New(letterBytes)

	return &s
}
