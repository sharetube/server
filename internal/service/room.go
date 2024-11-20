package service

import (
	"encoding/json"
	"errors"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/domain"
)

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrEmptyData        = errors.New("empty data")
)

type Input struct {
	Action string         `json:"action"`
	Sender *domain.Member `json:"-"`
	Data   []byte         `json:"data"`
}

func (i *Input) UnmarshalJSON(data []byte) error {
	var dataMap map[string]any
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return err
	}

	action, ok := dataMap["action"].(string)
	if !ok {
		return errors.New("invalid action")
	}

	i.Action = action

	dataBytes, err := json.Marshal(dataMap["data"])
	if err != nil {
		return err
	}
	i.Data = dataBytes

	return nil
}

type Message struct {
	Action string `json:"action"`
	Data   any    `json:"data"`
}

type Room struct {
	playlist *domain.Playlist
	members  *domain.Members
	player   *domain.Player
	inputCh  chan Input
	closeCh  chan struct{}
}

func newRoom(creator *domain.Member, initialVideoURL string, membersLimit, playlistLimit int) *Room {
	creator.IsAdmin = true
	return &Room{
		playlist: domain.NewPlaylist(initialVideoURL, creator.ID, playlistLimit),
		members:  domain.NewMembers(creator, membersLimit),
		player:   domain.NewPlayer(initialVideoURL),
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
