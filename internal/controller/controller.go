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

func NewController(roomService service.RoomService) *Controller {
	return &Controller{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		roomService: roomService,
	}
}
