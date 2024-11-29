package controller

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/validator"
)

type iRoomService interface {
	CreateCreateRoomSession(context.Context, *room.CreateRoomCreateSessionParams) (string, error)
	CreateJoinRoomSession(context.Context, *room.CreateRoomJoinSessionParams) (string, error)
	CreateRoom(context.Context, *room.CreateRoomParams) (room.CreateRoomResponse, error)
	JoinRoom(context.Context, *room.JoinRoomParams) (room.JoinRoomResponse, error)
	AddVideo(context.Context, *room.AddVideoParams) (room.AddVideoResponse, error)
	RemoveMember(context.Context, *room.RemoveMemberParams) (room.RemoveMemberResponse, error)
}

type controller struct {
	roomService iRoomService
	upgrader    websocket.Upgrader
	validate    *validator.Validator
}

func NewController(roomService iRoomService) *controller {
	return &controller{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		roomService: roomService,
		validate:    validator.NewValidator(),
	}
}
