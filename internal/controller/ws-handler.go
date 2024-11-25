package controller

// import (
// 	"fmt"
// 	"log/slog"
// 	"net/http"

// 	"github.com/go-chi/chi/v5"
// )

// func (c Controller) CreateRoom(w http.ResponseWriter, r *http.Request) {
// 	user, err := c.getUser(r)
// 	if err != nil {
// 		slog.Info("CreateRoom:", "error", err)
// 		fmt.Fprint(w, err)
// 		return
// 	}

// 	slog.Debug("CreateRoom: user recieved", "user", user)

// 	videoURL, err := c.getQueryParam(r, "video-url")
// 	if err != nil {
// 		slog.Info("CreateRoom:", "error", err)
// 		fmt.Fprint(w, err)
// 		return
// 	}

// 	headers := http.Header{}
// 	// headers.Add("Set-Cookie", cookieString)
// 	conn, err := c.upgrader.Upgrade(w, r, headers)
// 	if err != nil {
// 		slog.Warn("CreateRoom: failed to upgrade connection", "error", err)
// 		return
// 	}
// 	slog.Debug("CreateRoom: connection established", "user", user)

// 	user.Conn = conn

// 	room := c.roomService.CreateRoom(user, videoURL)

// 	go room.ReadMessages(conn)
// }

// func (c Controller) JoinRoom(w http.ResponseWriter, r *http.Request) {
// 	// for _, c := range r.Cookies() {
// 	// 	fmt.Printf("Cookie: %#v\n", c)
// 	// }

// 	user, err := c.getUser(r)
// 	if err != nil {
// 		slog.Info("JoinRoom:", "error", err)
// 		fmt.Fprint(w, err)
// 		return
// 	}

// 	slog.Debug("JoinRoom: user recieved", "user", user)

// 	roomID := chi.URLParam(r, "room-id")
// 	room, err := c.roomService.GetRoom(roomID)
// 	if err != nil {
// 		slog.Info("JoinRoom: failed to get room", "error", err)
// 		fmt.Fprint(w, err)
// 		return
// 	}

// 	headers := http.Header{}
// 	// headers.Add("Set-Cookie", cookieString)
// 	conn, err := c.upgrader.Upgrade(w, r, headers)
// 	if err != nil {
// 		slog.Warn("JoinRoom: failed to upgrade connection", "error", err)
// 		return
// 	}
// 	slog.Debug("JoinRoom: connection established", "user", user)

// 	user.Conn = conn

// 	room.AddMember(user)

// 	go room.ReadMessages(conn)
// }
