package internal

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type Message struct {
	Username string `json:"username"`
	Body     string `json:"body"`
}

type Member struct {
	ID       string
	Username string
	Color    string
	Conn     *websocket.Conn
}

type Video struct {
	URL     string
	AddedBy string
}

type Room struct {
	ID           string
	Queue        []Video
	Members      map[string]*Member
	MembersConns map[*websocket.Conn]*Member
	CreatorID    string
	Broadcast    chan Message
}

func NewRoom(id, creatorID string) *Room {
	return &Room{
		ID:           id,
		Queue:        []Video{},
		Members:      make(map[string]*Member),
		MembersConns: make(map[*websocket.Conn]*Member),
		CreatorID:    creatorID,
		Broadcast:    make(chan Message),
	}
}

func (r *Room) AddMember(member *Member) error {
	fmt.Printf("add member: %#v\n", member)
	r.Members[member.ID] = member
	r.MembersConns[member.Conn] = member
	return nil
}

func (r *Room) RemoveMember(member *Member) error {
	fmt.Printf("remove member: %#v\n", member)
	delete(r.Members, member.ID)
	return nil
}

func (r *Room) RemoveMemberByConn(conn *websocket.Conn) error {
	fmt.Println("remove member by conn")
	member := r.MembersConns[conn]
	if member == nil {
		return fmt.Errorf("member not found")
	}

	delete(r.Members, member.ID)
	delete(r.MembersConns, conn)

	return nil
}

func (r *Room) HandleMessages() {
	fmt.Println("room handle messages started")
	for {
		msg := <-r.Broadcast

		for memberConn := range r.MembersConns {
			err := memberConn.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
				memberConn.Close()
				r.RemoveMemberByConn(memberConn)
			}
		}
	}
}
