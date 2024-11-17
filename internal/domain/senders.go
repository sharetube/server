package domain

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func (r *Room) sendMemberJoined(member *Member) {
	r.SendMessageToAllMembers(&Message{
		Action: "member_joined",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) sendMemberLeft(member *Member) {
	r.SendMessageToAllMembers(&Message{
		Action: "member_left",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) sendMemberPromoted(member *Member) {
	r.SendMessageToAllMembers(&Message{
		Action: "member_promoted",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) sendMemberDemoted(member *Member) {
	r.SendMessageToAllMembers(&Message{
		Action: "member_demoted",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) sendVideoAdded(video *Video) {
	r.SendMessageToAllMembers(&Message{
		Action: "video_added",
		Data: map[string]any{
			"video":           video,
			"playlist":        r.playlist.AsList(),
			"playlist_length": r.playlist.Length(),
		},
	})
}

func (r *Room) sendVideoRemoved(video *Video) {
	r.SendMessageToAllMembers(&Message{
		Action: "video_removed",
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
		r.sendMessageToConn(member.Conn, msg)
	}
}

func (r *Room) sendMessageToConn(conn *websocket.Conn, msg *Message) {
	fmt.Println("sending message to member")
	if err := conn.WriteJSON(msg); err != nil {
		fmt.Println(err)
		r.RemoveMemberByConn(conn)
		conn.Close()
	}
}

func (r *Room) SendStateToAllMembersPeriodically(timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case _, more := <-r.closeCh:
			if !more {
				fmt.Println("ticker stopped")
				return
			}

			continue
		case <-ticker.C:
			// Send update on each tick
			r.SendMessageToAllMembers(&Message{
				Action: "update",
				Data:   r.GetState(),
			})
		}
	}
}

func (r *Room) sendError(conn *websocket.Conn, err error) {
	fmt.Printf("sending error: %s\n", err)
	r.sendMessageToConn(conn, &Message{
		Action: "error",
		Data: map[string]any{
			"message": err.Error(),
		},
	})
}
