package service

import (
	"context"
	"errors"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
	"github.com/skewb1k/goutils/randstr"
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
	ExpireMembers(context.Context, *room.ExpireMembersParams) error
	RemoveMemberFromList(context.Context, *room.RemoveMemberFromListParams) error
	GetMember(context.Context, *room.GetMemberParams) (room.Member, error)
	GetMemberIds(context.Context, string) ([]string, error)
	GetMemberIsAdmin(ctx context.Context, roomId string, memberId string) (bool, error)
	GetMemberIsMuted(ctx context.Context, roomId, memberId string) (bool, error)
	UpdateMemberIsAdmin(ctx context.Context, roomId string, memberId string, isAdmin bool) error
	UpdateMemberIsMuted(ctx context.Context, roomId string, memberId string, isMuted bool) error
	UpdateMemberIsReady(ctx context.Context, roomId string, memberId string, isReady bool) error
	UpdateMemberUsername(ctx context.Context, roomId string, memberId string, username string) error
	UpdateMemberColor(ctx context.Context, roomId string, memberId string, color string) error
	UpdateMemberAvatarUrl(ctx context.Context, roomId string, memberId string, avatarUrl *string) error
	// video
	SetVideo(context.Context, *room.SetVideoParams) (int, error)
	RemoveVideo(context.Context, *room.RemoveVideoParams) error
	ExpireVideo(context.Context, *room.ExpireVideoParams) error
	RemoveVideoFromList(context.Context, *room.RemoveVideoFromListParams) error
	SetLastVideo(context.Context, *room.SetLastVideoParams) error
	ExpireLastVideo(context.Context, *room.ExpireLastVideoParams) error
	GetPlaylistVersion(context.Context, string) (int, error)
	IncrPlaylistVersion(context.Context, string) (int, error)
	ExpirePlaylist(context.Context, *room.ExpirePlaylistParams) error
	GetVideoIds(context.Context, string) ([]int, error)
	GetVideo(context.Context, *room.GetVideoParams) (room.Video, error)
	GetVideosLength(context.Context, string) (int, error)
	GetLastVideoId(context.Context, string) (*int, error)
	ReorderList(context.Context, *room.ReorderListParams) error
	AddVideoToList(context.Context, *room.AddVideoToListParams) error
	SetCurrentVideoId(context.Context, *room.SetCurrentVideoParams) error
	GetCurrentVideoId(context.Context, string) (int, error)
	// player
	SetPlayer(context.Context, *room.SetPlayerParams) error
	GetPlayer(context.Context, string) (room.Player, error)
	IsPlayerExists(context.Context, string) (bool, error)
	RemovePlayer(context.Context, string) error
	ExpirePlayer(context.Context, *room.ExpirePlayerParams) error
	SetVideoEnded(context.Context, *room.SetVideoEndedParams) error
	GetVideoEnded(context.Context, string) (bool, error)
	ExpireVideoEnded(context.Context, *room.ExpireVideoEndedParams) error
	UpdatePlayerIsPlaying(ctx context.Context, roomId string, isPlaying bool) error
	UpdatePlayerWaitingForReady(ctx context.Context, roomId string, waitingForReady bool) error
	UpdatePlayerCurrentTime(ctx context.Context, roomId string, currentTime int) error
	UpdatePlayerPlaybackRate(ctx context.Context, roomId string, playbackRate float64) error
	UpdatePlayerUpdatedAt(ctx context.Context, roomId string, updatedAt int) error
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
	roomExp       time.Duration
}

type Config struct {
	MembersLimit  int
	PlaylistLimit int
	Secret        string
	RoomExp       time.Duration
}

func New(redisRepo iRoomRepo, connRepo iConnRepo, cfg *Config) *service {
	letterBytes := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	return &service{
		roomRepo:      redisRepo,
		connRepo:      connRepo,
		membersLimit:  cfg.MembersLimit,
		playlistLimit: cfg.PlaylistLimit,
		secret:        []byte(cfg.Secret),
		generator:     randstr.New(letterBytes),
		roomExp:       cfg.RoomExp,
	}
}
