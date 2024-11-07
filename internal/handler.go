package internal

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Handler struct {
	rooms    map[string]*Room
	upgrader websocket.Upgrader
}

func NewHandler() *Handler {
	return &Handler{
		rooms: make(map[string]*Room),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h Handler) JoinToRoomAndListen(w http.ResponseWriter, r *http.Request, room *Room, userID, username, color string) {
	headers := http.Header{}
	// headers.Add("Set-Cookie", cookieString)
	conn, err := h.upgrader.Upgrade(w, r, headers)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	fmt.Println("conn upgraded")

	member := Member{
		ID:       userID,
		Username: username,
		Color:    color,
		Conn:     conn,
	}
	room.AddMember(&member)

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Println("ReadJson error", err)
			if err := room.RemoveMemberByConn(conn); err != nil {
				fmt.Printf("remove member by conn error: %s\n", err)
				return
			}
			return
		}

		fmt.Printf("Message recieved: %v\n", msg)
		room.Broadcast <- msg
	}
}

func (h Handler) Mux() http.Handler {
	r := chi.NewRouter()

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userID := uuid.NewString()
		fmt.Printf("/ws userID: %s\n", userID)

		roomID := uuid.NewString()
		fmt.Printf("/ws roomID: %s\n", roomID)

		room := NewRoom(roomID, userID)
		h.rooms[roomID] = room
		fmt.Printf("/ws room created: %#v\n", room)

		go room.HandleMessages()

		h.JoinToRoomAndListen(w, r, room, userID, "some-username", "some-color")
	})

	r.HandleFunc("/ws/{room-id}", func(w http.ResponseWriter, r *http.Request) {
		// for _, c := range r.Cookies() {
		// 	fmt.Printf("Cookie: %#v\n", c)
		// }

		roomID := chi.URLParam(r, "room-id")
		if roomID == "" {
			fmt.Fprint(w, "room-id was not provided")
			return
		}

		userID := uuid.NewString()
		fmt.Printf("/ws/room userID: %s\n", userID)

		room := h.rooms[roomID]
		fmt.Printf("/ws/room roomID: %s\n", roomID)
		if room == nil {
			fmt.Fprint(w, "room not found")
			return
		}

		h.JoinToRoomAndListen(w, r, room, userID, "joined-username", "joined-color")
	})

	return r
}
