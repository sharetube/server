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
	RemoveMemberFromList(context.Context, *room.RemoveMemberFromListParams) error
	GetMember(context.Context, *room.GetMemberParams) (room.Member, error)
	GetMemberIds(context.Context, string) ([]string, error)
	GetMemberIsAdmin(ctx context.Context, roomId string, memberId string) (bool, error)
	UpdateMemberIsAdmin(ctx context.Context, roomId string, memberId string, isAdmin bool) error
	UpdateMemberIsMuted(ctx context.Context, roomId string, memberId string, isMuted bool) error
	UpdateMemberIsReady(ctx context.Context, roomId string, memberId string, isReady bool) error
	UpdateMemberUsername(ctx context.Context, roomId string, memberId string, username string) error
	UpdateMemberColor(ctx context.Context, roomId string, memberId string, color string) error
	UpdateMemberAvatarURL(ctx context.Context, roomId string, memberId string, avatarURL *string) error
	// video
	SetVideo(context.Context, *room.SetVideoParams) error
	RemoveVideo(context.Context, *room.RemoveVideoParams) error
	GetVideoIds(context.Context, string) ([]string, error)
	GetVideo(context.Context, *room.GetVideoParams) (room.Video, error)
	GetVideosLength(context.Context, string) (int, error)
	GetLastVideoId(context.Context, string) (*string, error)
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
	RemoveByMemberId(string) (*websocket.Conn, error)
	RemoveByConn(*websocket.Conn) (string, error)
	GetConn(string) (*websocket.Conn, error)
	GetMemberId(*websocket.Conn) (string, error)
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
}

func NewService(redisRepo iRoomRepo, connRepo iConnRepo, membersLimit, playlistLimit int, secret string) *service {
	letterBytes := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	return &service{
		roomRepo:      redisRepo,
		connRepo:      connRepo,
		membersLimit:  membersLimit,
		playlistLimit: playlistLimit,
		secret:        []byte(secret),
		generator:     randstr.New(letterBytes),
	}
}
