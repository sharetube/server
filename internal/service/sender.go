package service

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/domain"
)

func (r *Room) sendMemberJoined(member *domain.Member) {
	r.SendMessageToAllMembers(&Message{
		Action: "member_joined",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) sendMemberLeft(member *domain.Member) {
	r.SendMessageToAllMembers(&Message{
		Action: "member_left",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) sendMemberPromoted(member *domain.Member) {
	r.SendMessageToAllMembers(&Message{
		Action: "member_promoted",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) sendMemberDemoted(member *domain.Member) {
	r.SendMessageToAllMembers(&Message{
		Action: "member_demoted",
		Data: map[string]any{
			"member":        member,
			"members":       r.members.AsList(),
			"members_count": r.members.Length(),
		},
	})
}

func (r *Room) sendVideoAdded(video *domain.Video) {
	r.SendMessageToAllMembers(&Message{
		Action: "video_added",
		Data: map[string]any{
			"video":           video,
			"playlist":        r.playlist.AsList(),
			"playlist_length": r.playlist.Length(),
		},
	})
}

func (r *Room) sendVideoRemoved(video *domain.Video) {
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
	// slog.Debug("sending message to all members", "message", msg)
	for _, member := range r.members.AsList() {
		r.sendMessageToConn(member.Conn, msg)
	}
}

func (r *Room) sendMessageToConn(conn *websocket.Conn, msg *Message) {
	// slog.Debug("sending message to conn", "message", msg)
	if err := conn.WriteJSON(msg); err != nil {
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
				slog.Debug("ticker stopped")
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
	slog.Info("sending error", "error", err)
	r.sendMessageToConn(conn, &Message{
		Action: "error",
		Data: map[string]any{
			"message": err.Error(),
		},
	})
}
