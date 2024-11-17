package controller

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service"
)

type Controller struct {
	upgrader    websocket.Upgrader
	roomService service.RoomService
}

func NewController() *Controller {
	return &Controller{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		roomService: service.NewRoomService(),
	}
}
