package controller

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/validator"
)

type iRoomService interface {
	CreateRoom(context.Context, *room.CreateRoomParams) (room.CreateRoomResponse, error)
	ConnectMember(*room.ConnectMemberParams) error
	GetRoomState(context.Context, string) (room.RoomState, error)
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
