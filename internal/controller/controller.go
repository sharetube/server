package controller

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/validator"
	"github.com/sharetube/server/pkg/wsrouter"
)

type iRoomService interface {
	CreateRoom(context.Context, *room.CreateRoomParams) (room.CreateRoomResponse, error)
	ConnectMember(*room.ConnectMemberParams) error
	GetRoomState(ctx context.Context, roomID string) (room.RoomState, error)
	JoinRoom(context.Context, *room.JoinRoomParams) (room.JoinRoomResponse, error)
	AddVideo(context.Context, *room.AddVideoParams) (room.AddVideoResponse, error)
	RemoveVideo(context.Context, *room.RemoveVideoParams) (room.RemoveVideoResponse, error)
	RemoveMember(context.Context, *room.RemoveMemberParams) (room.RemoveMemberResponse, error)
	PromoteMember(context.Context, *room.PromoteMemberParams) (room.PromoteMemberResponse, error)
}

type controller struct {
	roomService iRoomService
	upgrader    websocket.Upgrader
	wsmux       *wsrouter.WSRouter
	validate    *validator.Validator
}

func NewController(roomService iRoomService) *controller {
	c := controller{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		roomService: roomService,
		validate:    validator.NewValidator(),
	}
	c.wsmux = c.getWSRouter()

	return &c
}
