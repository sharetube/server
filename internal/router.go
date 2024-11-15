package internal

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sharetube/server/internal/domain"
)

func GetUsername(w http.ResponseWriter, r *http.Request) string {
	username := r.URL.Query().Get("username")
	if username == "" {
		fmt.Println("username was not provided")
		fmt.Fprint(w, "username was not provided")
		return ""
	}

	return username
}

func GetColor(w http.ResponseWriter, r *http.Request) string {
	color := r.URL.Query().Get("color")
	if color == "" {
		fmt.Println("color was not provided")
		fmt.Fprint(w, "color was not provided")
		return ""
	}

	return color
}

func GetVideoURL(w http.ResponseWriter, r *http.Request) string {
	videoURL := r.URL.Query().Get("video-url")
	if videoURL == "" {
		fmt.Println("video url was not provided")
		fmt.Fprint(w, "video-url was not provided")
		return ""
	}

	return videoURL
}

func (h Handler) Mux() http.Handler {
	r := chi.NewRouter()

	r.HandleFunc("/ws/create-room", func(w http.ResponseWriter, r *http.Request) {
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
		go room.HandleMessages()
		go room.SendStateToAllMembersPeriodically(5 * time.Second)
		room.SendMessageToAllMembers(&domain.Message{
			Type: "svo",
			Data: "zxc",
		})

		fmt.Printf("/ws roomID: https://youtube.com/?room-id=%s\n", roomID)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go room.ReadMessages(conn, &wg)
		wg.Wait()

		fmt.Println("closing connection")
		conn.Close()
	})

	r.HandleFunc("/ws/join-room/{room-id}", func(w http.ResponseWriter, r *http.Request) {
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

		wg := sync.WaitGroup{}
		wg.Add(1)
		go room.ReadMessages(conn, &wg)
		wg.Wait()

		fmt.Println("closing connection")
		conn.Close()
	})

	return r
}
