package domain

import (
	"fmt"
	"strconv"

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
	Action string `json:"action"`
	Data   any    `json:"data"`
}

type Room struct {
	playlist *Playlist
	members  *Members
	inputCh  chan Input
}

func NewRoom(creator *Member, initialVideoURL string) *Room {
	return &Room{
		playlist: NewPlaylist(initialVideoURL, creator.ID, PlaylistLimit),
		members:  NewMembers(creator, MembersLimit),
		inputCh:  make(chan Input),
	}
}

func (r *Room) Close() {
	close(r.inputCh)
}

func (r Room) GetState() map[string]any {
	return map[string]any{
		"playlist": r.playlist.AsList(),
		"members":  r.members.AsList(),
	}
}

func (r *Room) AddMember(member *Member) {
	if err := r.members.Add(member); err != nil {
		r.SendError(member.Conn, err)
		return
	}

	r.SendMemberJoined(member)
}

func (r *Room) RemoveMemberByID(id string) {
	member, err := r.members.RemoveByID(id)
	if err != nil {
		fmt.Printf("remove member by id: %s\n", err)
		return
	}

	if r.members.Length() == 0 {
		close(r.inputCh)
		return
	}

	r.SendMemberLeft(&member)
}

func (r *Room) RemoveMemberByConn(conn *websocket.Conn) {
	member, err := r.members.RemoveByConn(conn)
	if err != nil {
		fmt.Printf("remove member by conn: %s\n", err)
		return
	}

	if r.members.Length() == 0 {
		close(r.inputCh)
		return
	}

	r.SendMemberLeft(&member)
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
		input, more := <-r.inputCh
		if !more {
			return
		}

		fmt.Printf("message recieved: %#v\n", input)
		switch input.Action {
		case "get_state":
			r.SendMessageToAllMembers(&Message{
				Action: "state",
				Data:   r.GetState(),
			})
		case "add_video":
			video, err := r.AddVideo(input.Sender, *input.Data)
			if err != nil {
				fmt.Printf("add video %s\n", err)
				r.SendError(input.Sender, err)
			}

			r.SendVideoAdded(&video)
		case "remove_video":
			videoIndex, err := strconv.Atoi(*input.Data)
			if err != nil {
				fmt.Printf("remove video error %s\n", err)
				r.SendError(input.Sender, err)
			}

			video, err := r.RemoveVideo(videoIndex)
			if err != nil {
				r.SendError(input.Sender, err)
			}

			r.SendVideoRemoved(&video)
		}
	}
}
