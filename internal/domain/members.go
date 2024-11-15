package domain

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/gorilla/websocket"
)

var (
	ErrMemberNotFound      = errors.New("member not found")
	ErrMemberAlreadyExists = errors.New("member already exists")
	ErrMembersLimitReached = errors.New("members limit reached")
)

type Member struct {
	ID        string          `json:"id"`
	Username  string          `json:"username"`
	Color     string          `json:"color"`
	AvatarURL string          `json:"avatar_url"`
	IsMuted   bool            `json:"is_muted"`
	IsAdmin   bool            `json:"is_admin"`
	Conn      *websocket.Conn `json:"-"`
}

type Members struct {
	list  map[string]*Member
	conns map[*websocket.Conn]*Member
	limit int
}

func NewMembers(creator *Member, limit int) *Members {
	return &Members{
		list: map[string]*Member{
			creator.ID: creator,
		},
		conns: map[*websocket.Conn]*Member{
			creator.Conn: creator,
		},
		limit: limit,
	}
}

func (m Members) AsList() []*Member {
	return slices.Collect(maps.Values(m.list))
}

func (m Members) Length() int {
	return len(m.list)
}

func (m *Members) Add(member *Member) error {
	fmt.Printf("add member: %#v\n", member)
	if m.list[member.ID] != nil {
		return ErrMemberAlreadyExists
	}

	if m.Length() >= m.limit {
		return ErrMembersLimitReached
	}

	m.list[member.ID] = member
	m.conns[member.Conn] = member
	return nil
}

func (m *Members) RemoveByID(id string) (Member, error) {
	fmt.Printf("remove member by id: %#v\n", id)
	member := m.list[id]
	if member == nil {
		return Member{}, ErrMemberNotFound
	}

	delete(m.conns, member.Conn)
	delete(m.list, id)
	return *member, nil
}

func (m *Members) RemoveByConn(conn *websocket.Conn) (Member, error) {
	fmt.Println("remove member by conn")
	member := m.conns[conn]
	if member == nil {
		return Member{}, ErrMemberNotFound
	}

	delete(m.list, member.ID)
	delete(m.conns, conn)
	return *member, nil
}

func (m Members) GetByID(id string) (Member, error) {
	member := m.list[id]
	if member == nil {
		return Member{}, ErrMemberNotFound
	}

	return *member, nil
}

func (m Members) GetByConn(conn *websocket.Conn) (Member, error) {
	member := m.conns[conn]
	if member == nil {
		return Member{}, ErrMemberNotFound
	}

	return *member, nil
}
