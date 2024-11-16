package domain

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func (r *Room) SendMemberJoined(member *Member) {
	r.SendMessageToAllMembers(&Message{
		Type: "member_joined",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) SendMemberLeft(member *Member) {
	r.SendMessageToAllMembers(&Message{
		Type: "member_left",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) SendVideoAdded(video *Video) {
	r.SendMessageToAllMembers(&Message{
		Type: "video_added",
		Data: map[string]any{
			"video":           video,
			"playlist":        r.playlist.AsList(),
			"playlist_length": r.playlist.Length(),
		},
	})
}

func (r *Room) SendVideoRemoved(video *Video) {
	r.SendMessageToAllMembers(&Message{
		Type: "video_removed",
		Data: map[string]any{
			"video":           video,
			"playlist":        r.playlist.AsList(),
			"playlist_length": r.playlist.Length(),
		},
	})
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

func (r *Room) SendError(conn *websocket.Conn, err error) {
	r.SendMessageToConn(conn, &Message{
		Type: "error",
		Data: map[string]any{
			"message": err,
		},
	})
}
