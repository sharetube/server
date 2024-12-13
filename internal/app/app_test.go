package app

import (
	"context"
	"log/slog"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/sharetube/server/internal/repository/connection/inmemory"
	roomRedis "github.com/sharetube/server/internal/repository/room/redis"
	"github.com/sharetube/server/internal/service/room"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRoom(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	s, _ := miniredis.Run()
	r := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	roomRepo := roomRedis.NewRepo(r, slog.Default())
	connRepo := inmemory.NewRepo(slog.Default())
	service := room.NewService(roomRepo, connRepo, 9, 25, slog.Default())

	ctx := context.Background()

	// create room
	createRoomParams := room.CreateRoomParams{
		Username:        "user1",
		Color:           "123",
		AvatarURL:       "some-avatar",
		InitialVideoURL: "some-video-id",
	}
	createRoomResp, err := service.CreateRoom(ctx, &createRoomParams)
	require.NoError(t, err)
	assert.NotEmpty(t, createRoomResp.RoomID, "room id is empty")
	assert.NotEmpty(t, createRoomResp.AuthToken, "auth token is empty")
	assert.NotEmpty(t, createRoomResp.MemberID, "member id is empty")

	connectMember1Params := room.ConnectMemberParams{
		Conn:     &websocket.Conn{},
		MemberID: createRoomResp.MemberID,
	}
	err = service.ConnectMember(ctx, &connectMember1Params)
	require.NoError(t, err)
	t.Log("room created")

	// member join room
	joinRoomParams := room.JoinRoomParams{
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

	connectMember2Params := room.ConnectMemberParams{
		Conn:     &websocket.Conn{},
		MemberID: joinRoomResp.JoinedMember.ID,
	}
	err = service.ConnectMember(ctx, &connectMember2Params)
	require.NoError(t, err)
	t.Log("member joined")

	// member 1 add video
	addVideoParams := room.AddVideoParams{
		SenderID: createRoomResp.MemberID,
		RoomID:   createRoomResp.RoomID,
		VideoURL: "asdasdasd",
	}
	addVideoResp, err := service.AddVideo(ctx, &addVideoParams)
	require.NoError(t, err)
	assert.Equal(t, len(addVideoResp.Videos), 1, "playlist must contain 1 videos")
	assert.Equal(t, len(addVideoResp.Conns), 2, "conns must contain 2 conns")
	assert.Equal(t, addVideoResp.AddedVideo.URL, addVideoParams.VideoURL, "video url is not equal")
	assert.Equal(t, addVideoResp.AddedVideo.AddedByID, createRoomResp.MemberID, "added by id is not equal")
	t.Log("video added")

	// member 1 disconnect
	disconnectMemberResp, err := service.DisconnectMember(ctx, &room.DisconnectMemberParams{
		MemberID: joinRoomResp.JoinedMember.ID,
		RoomID:   joinRoomParams.RoomID,
	})
	require.NoError(t, err)
	assert.Equal(t, disconnectMemberResp.IsRoomDeleted, false, "room must be not deleted")
	assert.Equal(t, len(disconnectMemberResp.Memberlist), 1, "memberlist must contain 1 member")
	assert.Equal(t, disconnectMemberResp.Memberlist[0].ID, createRoomResp.MemberID, "member id is not equal")
	t.Log("member 1 disconnected")

	t.Log(r.Keys(ctx, "*").Val())
}
