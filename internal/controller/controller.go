package controller

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service"
	"github.com/sharetube/server/pkg/validator"
	"github.com/sharetube/server/pkg/wsrouter"
)

type iRoomService interface {
	CreateRoom(context.Context, *service.CreateRoomParams) (*service.CreateRoomResponse, error)
	ConnectMember(context.Context, *service.ConnectMemberParams) error
	DisconnectMember(context.Context, *service.DisconnectMemberParams) (service.DisconnectMemberResponse, error)
	GetRoom(context.Context, string) (service.Room, error)
	UpdatePlayerState(context.Context, *service.UpdatePlayerStateParams) (service.UpdatePlayerStateResponse, error)
	UpdatePlayerVideo(context.Context, *service.UpdatePlayerVideoParams) (service.UpdatePlayerVideoResponse, error)
	JoinRoom(context.Context, *service.JoinRoomParams) (service.JoinRoomResponse, error)
	AddVideo(context.Context, *service.AddVideoParams) (service.AddVideoResponse, error)
	RemoveVideo(context.Context, *service.RemoveVideoParams) (service.RemoveVideoResponse, error)
	RemoveMember(context.Context, *service.RemoveMemberParams) (service.RemoveMemberResponse, error)
	PromoteMember(context.Context, *service.PromoteMemberParams) (service.PromoteMemberResponse, error)
	UpdateProfile(context.Context, *service.UpdateProfileParams) (service.UpdateProfileResponse, error)
	UpdateIsReady(context.Context, *service.UpdateIsReadyParams) (service.UpdateIsReadyResponse, error)
	UpdateIsMuted(context.Context, *service.UpdateIsMutedParams) (service.UpdateIsMutedResponse, error)
	ReorderPlaylist(context.Context, *service.ReorderPlaylistParams) (service.ReorderPlaylistResponse, error)
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
