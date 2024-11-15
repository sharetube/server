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
	username := GetUsername(w, r)
	fmt.Printf("/ws username: %s\n", username)

	color := GetColor(w, r)
	fmt.Printf("/ws color: %s\n", color)

	videoURL := GetVideoURL(w, r)
	fmt.Printf("/ws videoURL: %s\n", color)

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

	username := GetUsername(w, r)
	fmt.Printf("/ws/join-room username: %s\n", username)

	color := GetColor(w, r)
	fmt.Printf("/ws/join-room color: %s\n", color)

	// userID := uuid.NewString()
	// fmt.Printf("/ws userID: %s\n", userID)
	userID := username

	roomID := chi.URLParam(r, "room-id")
	room, err := h.roomService.GetRoom(roomID)
	if err != nil {
		fmt.Println("room id was not provided")
		fmt.Fprint(w, "room-id was not provided")
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

	if err := room.AddMember(&member); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("/ws roomID: https://youtube.com/?room-id=%s\n", roomID)

	go room.ReadMessages(conn)
}
