package domain

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/gorilla/websocket"
)

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrEmptyData        = errors.New("empty data")
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
	closeCh  chan struct{}
}

func NewRoom(creator *Member, initialVideoURL string, membersLimit, playlistLimit int) *Room {
	creator.IsAdmin = true
	return &Room{
		playlist: NewPlaylist(initialVideoURL, creator.ID, playlistLimit),
		members:  NewMembers(creator, membersLimit),
		inputCh:  make(chan Input),
		closeCh:  make(chan struct{}),
	}
}

func (r Room) GetState() map[string]any {
	return map[string]any{
		"playlist":        r.playlist.AsList(),
		"playlist_length": r.playlist.Length(),
		"members":         r.members.AsList(),
		"members_count":   r.members.Length(),
	}
}

func (r *Room) Close() {
	close(r.inputCh)
	close(r.closeCh)
}

func (r *Room) AddMember(member *Member) {
	member.IsAdmin = false
	if err := r.members.Add(member); err != nil {
		r.sendError(member.Conn, err)
		return
	}

	r.sendMemberJoined(member)
}

func (r *Room) RemoveMemberByConn(conn *websocket.Conn) {
	member, err := r.members.RemoveByConn(conn)
	if err != nil {
		return
	}

	if r.members.Length() == 0 {
		r.Close()
		return
	}

	r.sendMemberLeft(&member)
}

func (r *Room) ReadMessages(conn *websocket.Conn) {
	for {
		var input Input
		if err := conn.ReadJSON(&input); err != nil {
			slog.Warn("error reading message", "error", err)
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
			removedMember, err := func() (*Member, error) {
				if input.Data == nil {
					return nil, ErrEmptyData
				}

				member, _, err := r.members.GetByConn(input.Sender)
				if err != nil {
					return nil, err
				}

				if !member.IsAdmin {
					return nil, ErrPermissionDenied
				}

				removedMember, err := r.members.RemoveByID(*input.Data)
				if err != nil {
					return nil, err
				}

				if r.members.Length() == 0 {
					r.Close()
					return nil, nil
				}

				removedMember.Conn.Close()

				return &removedMember, nil
			}()

			if err != nil {
				r.sendError(input.Sender, err)
			} else {
				r.sendMemberLeft(removedMember)
			}
		case "promote_member":
			promotedMember, err := func() (*Member, error) {
				if input.Data == nil {
					return nil, ErrEmptyData
				}

				member, _, err := r.members.GetByConn(input.Sender)
				if err != nil {
					return nil, err
				}

				if !member.IsAdmin {
					return nil, ErrPermissionDenied
				}

				promotedMember, err := r.members.PromoteMemberByID(*input.Data)
				if err != nil {
					return nil, err
				}

				return &promotedMember, nil
			}()

			if err != nil {
				r.sendError(input.Sender, err)
			} else {
				r.sendMemberPromoted(promotedMember)
			}

		case "demote_member":
			demotedMember, err := func() (*Member, error) {
				if input.Data == nil {
					return nil, ErrEmptyData
				}

				member, _, err := r.members.GetByConn(input.Sender)
				if err != nil {
					return nil, err
				}

				if !member.IsAdmin {
					return nil, ErrPermissionDenied
				}

				demotedMember, err := r.members.DemoteMemberByID(*input.Data)
				if err != nil {
					return nil, err
				}

				return &demotedMember, nil
			}()

			if err != nil {
				r.sendError(input.Sender, err)
			} else {
				r.sendMemberDemoted(demotedMember)
			}
		case "add_video":
			// todo: refactor.
			video, err := func() (*Video, error) {
				member, _, err := r.members.GetByConn(input.Sender)
				if err != nil {
					return nil, err
				}

				if input.Data == nil {
					return nil, ErrEmptyData
				}

				if !member.IsAdmin {
					return nil, ErrPermissionDenied
				}

				video, err := r.playlist.Add(member.ID, *input.Data)
				if err != nil {
					return nil, err
				}

				return &video, nil
			}()

			if err != nil {
				r.sendError(input.Sender, err)
			} else {
				r.sendVideoAdded(video)
			}
		case "remove_video":
			// todo: refactor.
			video, err := func() (*Video, error) {
				member, _, err := r.members.GetByConn(input.Sender)
				if err != nil {
					return nil, err
				}

				if !member.IsAdmin {
					return nil, ErrPermissionDenied
				}

				if input.Data == nil {
					return nil, ErrVideoNotFound
				}

				videoID, err := strconv.Atoi(*input.Data)
				if err != nil {
					return nil, err
				}

				video, err := r.playlist.RemoveByID(videoID)
				if err != nil {
					return nil, err
				}

				return &video, nil
			}()

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
