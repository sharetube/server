package internal

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type Handler struct {
	upgrader    websocket.Upgrader
	roomService RoomService
}

func NewHandler() *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		roomService: NewRoomService(),
	}
}

// func (h Handler) ReadMessages(conn *websocket.Conn, hub *hub, wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	for {
// 		var input Input
// 		err := conn.ReadJSON(&input)
// 		if err != nil {
// 			fmt.Println("ReadJson error", err)
// 			hub.RemoveMemberByConn(conn)
// 			return
// 		}
// 		input.Sender = conn

// 		fmt.Printf("Message recieved: %v\n", input)
// 		hub.ReadMessage(conn, input)
// 	}
// }

// func (h Handler) JoinRoomAndListen(w http.ResponseWriter, r *http.Request, hub *hub, member *models.Member) {
// 	headers := http.Header{}
// 	// headers.Add("Set-Cookie", cookieString)
// 	conn, err := h.upgrader.Upgrade(w, r, headers)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	fmt.Println("conn upgraded")

// 	member.Conn = conn
// 	hub.AddMember(conn, member)

// 	wg := sync.WaitGroup{}
// 	wg.Add(1)
// 	go h.ReadMessages(conn, hub, &wg)
// 	wg.Wait()

// 	hub.SendStateToMemberByConn(conn)

// 	fmt.Println("closing connection")
// 	conn.Close()
// }
