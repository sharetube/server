package domain

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	PlaylistLimit = 25
	MembersLimit  = 9
)

type Input struct {
	Action string          `json:"action"`
	Sender *websocket.Conn `json:"-"`
	Data   *string         `json:"data"`
}

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Room struct {
	Playlist *Playlist `json:"playlist"`
	Members  *Members  `json:"members"`
	inputCh  chan Input
	closeCh  chan struct{}
}

func NewRoom(creator *Member, initialVideoURL string) *Room {
	return &Room{
		Playlist: NewPlaylist(initialVideoURL, creator.ID, PlaylistLimit),
		Members:  NewMembers(creator, MembersLimit),
		inputCh:  make(chan Input),
		closeCh:  make(chan struct{}),
	}
}

func (r Room) GetState() map[string]any {
	return map[string]any{
		"playlist": r.Playlist.AsList(),
		"members":  r.Members.AsList(),
	}
}

func (r *Room) AddMember(member *Member) error {
	return r.Members.Add(member)
}

func (r *Room) RemoveMemberByID(id string) (Member, error) {
	return r.Members.RemoveByID(id)
}

func (r *Room) RemoveMemberByConn(conn *websocket.Conn) (Member, error) {
	return r.Members.RemoveByConn(conn)
}

func (r *Room) AddVideo(addedBy *websocket.Conn, url string) (Video, error) {
	member, err := r.Members.GetByConn(addedBy)
	if err != nil {
		return Video{}, ErrMemberNotFound
	}

	return r.Playlist.Add(member.ID, url)
}

func (r *Room) RemoveVideo(videoIndex int) (Video, error) {
	return r.Playlist.Remove(videoIndex)
}

func (r Room) SendError(conn *websocket.Conn, err error) {
	r.SendMessageToConn(conn, &Message{
		Type: "error",
		Data: map[string]any{
			"message": err,
		},
	})
}

func (r *Room) ReadMessages(conn *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		var input Input
		if err := conn.ReadJSON(&input); err != nil {
			fmt.Println("ReadJson error", err)
			r.RemoveMemberByConn(conn)
			return
		}
		input.Sender = conn

		fmt.Printf("Message recieved: %v\n", input)
		r.ReadMessage(conn, input)
	}
}

// todo: refactor
func (r Room) HandleMessages() {
	fmt.Println("room handle messages started")
	for {
		select {
		case <-r.closeCh:
			fmt.Println("closing hub")
			return
		case input := <-r.inputCh:
			fmt.Printf("message recieved: %#v\n", input)
			var outMsg Message
			switch input.Action {
			case "get_state":
				outMsg = Message{
					Type: "state",
					Data: r.GetState(),
				}
			case "add_video":
				r.AddVideo(nil, *input.Data)
			case "remove_video":
				videoIndex, err := strconv.Atoi(*input.Data)
				if err != nil {
					fmt.Printf("remove %s\n", err)
					r.SendError(input.Sender, err)
				}

				r.RemoveVideo(videoIndex)
			}
			fmt.Printf("message to send: %#v\n", outMsg)
			r.SendMessageToAllMembers(&outMsg)
		}
	}
}

func (r Room) SendMessageToAllMembers(msg *Message) {
	fmt.Println("sending message to all members")
	for memberConn := range r.Members.conns {
		r.SendMessageToConn(memberConn, msg)
	}
}

func (r Room) SendMessageToConn(conn *websocket.Conn, msg *Message) {
	fmt.Println("sending message to member")
	if err := conn.WriteJSON(msg); err != nil {
		fmt.Println(err)
		conn.Close()
		r.RemoveMemberByConn(conn)
	}
}

func (r Room) SendStateToAllMembersPeriodically(timeout time.Duration) {
	for {
		select {
		case <-r.closeCh:
			fmt.Println("stoping spam closed")
			return
		default:
			time.Sleep(timeout)
			r.SendMessageToAllMembers(&Message{
				Type: "update",
				Data: r.GetState(),
			})
		}
	}
}

func (r Room) ReadMessage(conn *websocket.Conn, input Input) {
	r.inputCh <- input
}
