package domain

import (
	"fmt"
	"strconv"
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
	playlist *Playlist
	members  *Members
	inputCh  chan Input
	closeCh  chan struct{}
}

func NewRoom(creator *Member, initialVideoURL string) *Room {
	return &Room{
		playlist: NewPlaylist(initialVideoURL, creator.ID, PlaylistLimit),
		members:  NewMembers(creator, MembersLimit),
		inputCh:  make(chan Input),
		closeCh:  make(chan struct{}),
	}
}

func (r Room) GetState() map[string]any {
	return map[string]any{
		"playlist": r.playlist.AsList(),
		"members":  r.members.AsList(),
	}
}

func (r *Room) AddMember(member *Member) error {
	return r.members.Add(member)
}

func (r *Room) RemoveMemberByID(id string) {
	member, err := r.members.RemoveByID(id)
	if err != nil {
		fmt.Printf("remove member by conn: %s\n", err)
		return
	}

	if r.members.Length() == 0 {
		r.closeCh <- struct{}{}
		r.closeCh <- struct{}{}
		return
	}

	r.SendMessageToAllMembers(&Message{
		Type: "member_left",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) RemoveMemberByConn(conn *websocket.Conn) {
	member, err := r.members.RemoveByConn(conn)
	if err != nil {
		fmt.Printf("remove member by conn: %s\n", err)
		return
	}

	if r.members.Length() == 0 {
		r.closeCh <- struct{}{}
		r.closeCh <- struct{}{}
		return
	}

	r.SendMessageToAllMembers(&Message{
		Type: "member_left",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) AddVideo(addedBy *websocket.Conn, url string) (Video, error) {
	member, _, err := r.members.GetByConn(addedBy)
	if err != nil {
		return Video{}, ErrMemberNotFound
	}

	return r.playlist.Add(member.ID, url)
}

func (r *Room) RemoveVideo(videoIndex int) (Video, error) {
	return r.playlist.RemoveByID(videoIndex)
}

func (r *Room) SendError(conn *websocket.Conn, err error) {
	r.SendMessageToConn(conn, &Message{
		Type: "error",
		Data: map[string]any{
			"message": err,
		},
	})
}

func (r *Room) ReadMessages(conn *websocket.Conn) {
	for {
		var input Input
		if err := conn.ReadJSON(&input); err != nil {
			fmt.Println("ReadJson error", err)
			r.RemoveMemberByConn(conn)
			conn.Close()
			return
		}
		input.Sender = conn

		fmt.Printf("Message recieved: %v\n", input)
		r.inputCh <- input
	}
}

func (r *Room) HandleMessages(input Input) {
	for {
		select {
		case <-r.closeCh:
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

func (r *Room) SendMessageToAllMembers(msg *Message) {
	fmt.Println("sending message to all members")
	for _, member := range r.members.AsList() {
		r.SendMessageToConn(member.Conn, msg)
	}
}

func (r *Room) SendMessageToConn(conn *websocket.Conn, msg *Message) {
	fmt.Println("sending message to member")
	if err := conn.WriteJSON(msg); err != nil {
		fmt.Println(err)
		conn.Close()
		r.RemoveMemberByConn(conn)
	}
}

func (r *Room) SendStateToAllMembersPeriodically(timeout time.Duration) {
	for {
		select {
		case <-r.closeCh:
			fmt.Println("stop spam")
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
