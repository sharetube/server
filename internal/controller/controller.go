package controller

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/validator"
	"github.com/sharetube/server/pkg/wsrouter"
)

type iRoomService interface {
	CreateRoom(context.Context, *room.CreateRoomParams) (*room.CreateRoomResponse, error)
	ConnectMember(context.Context, *room.ConnectMemberParams) error
	DisconnectMember(context.Context, *room.DisconnectMemberParams) (room.DisconnectMemberResponse, error)
	GetRoom(context.Context, string) (room.Room, error)
	UpdatePlayerState(context.Context, *room.UpdatePlayerStateParams) (room.UpdatePlayerStateResponse, error)
	UpdatePlayerVideo(context.Context, *room.UpdatePlayerVideoParams) (room.UpdatePlayerVideoResponse, error)
	JoinRoom(context.Context, *room.JoinRoomParams) (room.JoinRoomResponse, error)
	AddVideo(context.Context, *room.AddVideoParams) (room.AddVideoResponse, error)
	RemoveVideo(context.Context, *room.RemoveVideoParams) (room.RemoveVideoResponse, error)
	RemoveMember(context.Context, *room.RemoveMemberParams) (room.RemoveMemberResponse, error)
	PromoteMember(context.Context, *room.PromoteMemberParams) (room.PromoteMemberResponse, error)
	UpdateProfile(context.Context, *room.UpdateProfileParams) (room.UpdateProfileResponse, error)
}

type controller struct {
	roomService iRoomService
	upgrader    websocket.Upgrader
	wsmux       *wsrouter.WSRouter
	validate    *validator.Validator
	logger      *slog.Logger
}

func NewController(roomService iRoomService, logger *slog.Logger) *controller {
	c := controller{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		roomService: roomService,
		validate:    validator.NewValidator(),
		logger:      logger,
	}
	c.wsmux = c.getWSRouter()

	return &c
}
