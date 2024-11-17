package domain

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gorilla/websocket"
)

var (
	ErrNoVideoUrlProvided = errors.New("no video url provided")
	ErrNoMemberIDProvided = errors.New("no member id provided")
	ErrPermissionDenied   = errors.New("permission denied")
)

// todo: add to config.
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

func (r *Room) Close() {
	close(r.inputCh)
	close(r.closeCh)
}

func (r *Room) AddMember(member *Member) {
	if err := r.members.Add(member); err != nil {
		r.SendError(member.Conn, err)
		return
	}

	r.SendMemberJoined(member)
}

func (r *Room) RemoveMemberByConn(conn *websocket.Conn) {
	member, err := r.members.RemoveByConn(conn)
	if err != nil {
		fmt.Printf("remove member by conn: %s\n", err)
		return
	}

	if r.members.Length() == 0 {
		r.Close()
		return
	}

	r.SendMemberLeft(&member)
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
*/
func (r *Room) HandleMessages() {
	for {
		input, more := <-r.inputCh
		if !more {
			fmt.Println("message handling stopped")
			return
		}

		fmt.Printf("message recieved: %+v\n", input)
		switch input.Action {
		case "get_state":
			fmt.Println("get state")
			r.SendMessageToAllMembers(&Message{
				Action: "state",
				Data:   r.GetState(),
			})
		case "remove_member":
			if input.Data == nil {
				r.SendError(input.Sender, ErrNoVideoUrlProvided)
				break
			}

			removedMember, err := func() (*Member, error) {
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
				r.SendError(input.Sender, err)
			} else {
				r.SendMemberLeft(removedMember)
			}
		case "add_video":
			// todo: refactor.
			video, err := func() (*Video, error) {
				member, _, err := r.members.GetByConn(input.Sender)
				if err != nil {
					return nil, err
				}

				if input.Data == nil {
					return nil, ErrNoVideoUrlProvided
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
				r.SendError(input.Sender, err)
			} else {
				r.SendVideoAdded(video)
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
					return nil, ErrNoVideoUrlProvided
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
				r.SendError(input.Sender, err)
			} else {
				r.SendVideoRemoved(video)
			}
		default:
			fmt.Printf("unknown action: %s\n", input.Action)
			r.SendError(input.Sender, fmt.Errorf("unknown action: %s", input.Action))
		}
	}
}
