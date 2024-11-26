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
	GetMemberIDByConnectToken(context.Context, string) (string, error)
	ConnectMember(context.Context, *websocket.Conn, string) error
	AddVideo(context.Context, *room.AddVideoParams) (room.AddVideoResponse, error)
}

type Controller struct {
	roomService iRoomService
	upgrader    websocket.Upgrader
	validate    *validator.Validator
}

func NewController(roomService iRoomService) *Controller {
	return &Controller{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		roomService: roomService,
		validate:    validator.NewValidator(),
	}
}
