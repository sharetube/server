package room

import (
	"fmt"
	"log/slog"

	"github.com/gorilla/websocket"
)

func (r *Room) ReadMessages(conn *websocket.Conn) {
	for {
		var input Input
		if err := conn.ReadJSON(&input); err != nil {
			slog.Debug("error reading message", "error", err)
			r.sendError(conn, fmt.Errorf("error reading message: %w", err))
			r.RemoveMemberByConn(conn)
			conn.Close()
			return

		}
		slog.Info("message recieved", "message", input)

		member, _, err := r.members.GetByConn(conn)
		if err != nil {
			slog.Warn("error getting member", "error", err)
			return
		}
		input.Sender = &member

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
				r.sendError(input.Sender.Conn, err)
			} else {
				r.sendMemberLeft(removedMember)
			}
		case "promote_member":
			promotedMember, err := r.handlePromoteMember(&input)

			if err != nil {
				r.sendError(input.Sender.Conn, err)
			} else {
				r.sendMemberPromoted(promotedMember)
			}

		case "demote_member":
			demotedMember, err := r.handleDemoteMember(&input)

			if err != nil {
				r.sendError(input.Sender.Conn, err)
			} else {
				r.sendMemberDemoted(demotedMember)
			}
		case "add_video":
			video, err := r.handleAddVideo(&input)

			if err != nil {
				r.sendError(input.Sender.Conn, err)
			} else {
				r.sendVideoAdded(video)
			}
		case "remove_video":
			video, err := r.handleRemoveVideo(&input)

			if err != nil {
				r.sendError(input.Sender.Conn, err)
			} else {
				r.sendVideoRemoved(video)
			}
		case "player_updated":
			player, err := r.handlePlayerUpdated(&input)
			if err != nil {
				r.sendError(input.Sender.Conn, err)
			} else {
				r.sendPlayerUpdated(player)
			}
		default:
			slog.Warn("unknown action", "action", input.Action)
			r.sendError(input.Sender.Conn, fmt.Errorf("unknown action: %s", input.Action))
		}
	}
}
