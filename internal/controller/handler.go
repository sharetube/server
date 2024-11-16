package controller

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/domain"
	"github.com/sharetube/server/internal/service"
)

type Handler struct {
	upgrader    websocket.Upgrader
	roomService service.RoomService
}

func NewHandler() *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		roomService: service.NewRoomService(),
	}
}

func (h Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	username, err := MustHeader(r, "Username")
	if err != nil {
		fmt.Printf("/ws/create-room: %s\n", err)
		fmt.Fprint(w, err)
		return
	}

	color, err := MustHeader(r, "Color")
	if err != nil {
		fmt.Printf("/ws/create-room: %s\n", err)
		fmt.Fprint(w, err)
		return
	}

	videoURL, err := MustHeader(r, "Video-Url")
	if err != nil {
		fmt.Printf("/ws/create-room: %s\n", err)
		fmt.Fprint(w, err)
		return
	}

	// userID := uuid.NewString()
	// fmt.Printf("/ws userID: %s\n", userID)
	userID := username

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := h.upgrader.Upgrade(w, r, headers)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("conn upgraded")

	member := domain.Member{
		ID:       userID,
		Username: username,
		Color:    color,
		IsAdmin:  true,
		Conn:     conn,
	}

	roomID, room := h.roomService.CreateRoom(&member, videoURL)

	fmt.Printf("/ws roomID: https://youtube.com/?room-id=%s\n", roomID)

	go room.ReadMessages(conn)
}

func (h Handler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	// for _, c := range r.Cookies() {
	// 	fmt.Printf("Cookie: %#v\n", c)
	// }

	username, err := MustHeader(r, "Username")
	if err != nil {
		fmt.Printf("/ws/join-room: %s\n", err)
		fmt.Fprint(w, err)
		return
	}

	color, err := MustHeader(r, "Color")
	if err != nil {
		fmt.Printf("/ws/join-room: %s\n", err)
		fmt.Fprint(w, err)
		return
	}

	// userID := uuid.NewString()
	// fmt.Printf("/ws userID: %s\n", userID)
	userID := username

	roomID := chi.URLParam(r, "room-id")
	room, err := h.roomService.GetRoom(roomID)
	if err != nil {
		fmt.Printf("/ws/join-room: %s\n", err)
		fmt.Fprint(w, err)
		return
	}

	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := h.upgrader.Upgrade(w, r, headers)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("conn upgraded")

	member := domain.Member{
		ID:       userID,
		Username: username,
		Color:    color,
		IsAdmin:  false,
		Conn:     conn,
	}

	room.AddMember(&member)

	go room.ReadMessages(conn)
}
