package room

import (
	"context"
	"log/slog"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository/connection/inmemory"
	roomRedis "github.com/sharetube/server/internal/repository/room/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRoom(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	s, _ := miniredis.Run()
	r := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	roomRepo := roomRedis.NewRepo(r)
	connRepo := inmemory.NewRepo()
	service := NewService(roomRepo, connRepo, 9, 25)

	ctx := context.Background()

	createRoomParams := CreateRoomParams{
		Username:        "usern",
		Color:           "123",
		AvatarURL:       "ava",
		InitialVideoURL: "svosvo",
	}
	createRoomResp, err := service.CreateRoom(ctx, &createRoomParams)
	require.NoError(t, err)
	assert.NotEmpty(t, createRoomResp.RoomID, "room id is empty")
	assert.NotEmpty(t, createRoomResp.AuthToken, "auth token is empty")
	assert.NotEmpty(t, createRoomResp.MemberID, "member id is empty")

	connectMember1Params := ConnectMemberParams{
		Conn:     &websocket.Conn{},
		MemberID: createRoomResp.MemberID,
	}
	err = service.ConnectMember(&connectMember1Params)
	require.NoError(t, err)

	joinRoomParams := JoinRoomParams{
		Username:  "user2",
		Color:     "fff",
		AvatarURL: "",
		RoomID:    createRoomResp.RoomID,
	}
	joinRoomResp, err := service.JoinRoom(ctx, &joinRoomParams)
	require.NoError(t, err)
	assert.NotEmpty(t, joinRoomResp.JoinedMember.ID)
	assert.Equal(t, joinRoomResp.JoinedMember.Username, joinRoomParams.Username, "username is not equal")
	assert.Equal(t, joinRoomResp.JoinedMember.Color, joinRoomParams.Color, "color is not equal")
	assert.Equal(t, joinRoomResp.JoinedMember.AvatarURL, joinRoomParams.AvatarURL, "avatar url is not equal")
	assert.Equal(t, joinRoomResp.JoinedMember.IsAdmin, false, "is admin must be false")
	assert.Equal(t, joinRoomResp.JoinedMember.IsOnline, false, "is online must be false")
	assert.Equal(t, joinRoomResp.JoinedMember.IsMuted, false, "is muted must be false")
	assert.Equal(t, len(joinRoomResp.MemberList), 2, "memberlist must contain 2 members")

	connectMember2Params := ConnectMemberParams{
		Conn:     &websocket.Conn{},
		MemberID: joinRoomResp.JoinedMember.ID,
	}
	err = service.ConnectMember(&connectMember2Params)
	require.NoError(t, err)

	addVideoParams := AddVideoParams{
		SenderID: createRoomResp.MemberID,
		RoomID:   createRoomResp.RoomID,
		VideoURL: "asdasdasd",
	}
	addVideoResp, err := service.AddVideo(ctx, &addVideoParams)
	require.NoError(t, err)
	assert.Equal(t, len(addVideoResp.Playlist), 1, "playlist must contain 1 videos")
	assert.Equal(t, len(addVideoResp.Conns), 2, "conns must contain 2 conns")
	assert.Equal(t, addVideoResp.AddedVideo.URL, addVideoParams.VideoURL, "video url is not equal")
	assert.Equal(t, addVideoResp.AddedVideo.AddedByID, createRoomResp.MemberID, "added by id is not equal")
}
