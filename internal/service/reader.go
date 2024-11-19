package service

import (
	"fmt"
	"log/slog"

	"github.com/gorilla/websocket"
)

func (r *Room) ReadMessages(conn *websocket.Conn) {
	for {
		var input Input
		if err := conn.ReadJSON(&input); err != nil {
			r.RemoveMemberByConn(conn)
			conn.Close()
			return
		}
		slog.Info("message recieved", "message", input)
		input.Sender = conn

		r.inputCh <- input
	}
}

/*
Runs in a goroutine and handle messages from inputCh.

Available actions:

- get_state

- add_video

- remove_video

- remove_member

- promote_member

- demote_member
*/
func (r *Room) HandleMessages() {
	for {
		input, more := <-r.inputCh
		if !more {
			slog.Debug("input channel closed")
			return
		}

		switch input.Action {
		case "get_state":
			r.SendMessageToAllMembers(&Message{
				Action: "state",
				Data:   r.GetState(),
			})
		case "remove_member":
			removedMember, err := r.handleRemoveMember(&input)

			if err != nil {
				r.sendError(input.Sender, err)
			} else {
				r.sendMemberLeft(removedMember)
			}
		case "promote_member":
			promotedMember, err := r.handlePromoteMember(&input)

			if err != nil {
				r.sendError(input.Sender, err)
			} else {
				r.sendMemberPromoted(promotedMember)
			}

		case "demote_member":
			demotedMember, err := r.handleDemoteMember(&input)

			if err != nil {
				r.sendError(input.Sender, err)
			} else {
				r.sendMemberDemoted(demotedMember)
			}
		case "add_video":
			video, err := r.handleAddVideo(&input)

			if err != nil {
				r.sendError(input.Sender, err)
			} else {
				r.sendVideoAdded(video)
			}
		case "remove_video":
			video, err := r.handleRemoveVideo(&input)

			if err != nil {
				r.sendError(input.Sender, err)
			} else {
				r.sendVideoRemoved(video)
			}
		default:
			slog.Warn("unknown action", "action", input.Action)
			r.sendError(input.Sender, fmt.Errorf("unknown action: %s", input.Action))
		}
	}
}
