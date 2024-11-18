package service

import (
	"errors"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/domain"
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
	playlist *domain.Playlist
	members  *domain.Members
	inputCh  chan Input
	closeCh  chan struct{}
}

func newRoom(creator *domain.Member, initialVideoURL string, membersLimit, playlistLimit int) *Room {
	creator.IsAdmin = true
	return &Room{
		playlist: domain.NewPlaylist(initialVideoURL, creator.ID, playlistLimit),
		members:  domain.NewMembers(creator, membersLimit),
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

func (r *Room) AddMember(member *domain.Member) {
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
