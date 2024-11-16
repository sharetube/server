package domain

import (
	"errors"
	"fmt"

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
	list  []Member
	limit int
}

func NewMembers(creator *Member, limit int) *Members {
	return &Members{
		list:  []Member{*creator},
		limit: limit,
	}
}

func (m Members) Length() int {
	return len(m.list)
}

func (m Members) AsList() []Member {
	return m.list
}

func (m Members) GetByID(id string) (Member, int, error) {
	fmt.Printf("get member by id: %#v\n", id)
	for index, member := range m.list {
		if member.ID == id {
			return member, index, nil
		}
	}

	return Member{}, 0, ErrMemberNotFound
}

func (m Members) GetByConn(conn *websocket.Conn) (Member, int, error) {
	fmt.Println("get member by conn")
	for index, member := range m.list {
		if member.Conn == conn {
			return member, index, nil
		}
	}

	return Member{}, 0, ErrMemberNotFound
}

func (m *Members) Add(member *Member) error {
	fmt.Printf("add member: %#v\n", member)
	if _, _, err := m.GetByID(member.ID); err == nil {
		return ErrMemberAlreadyExists
	}

	if m.Length() >= m.limit {
		return ErrMembersLimitReached
	}

	m.list = append(m.list, *member)
	return nil
}

func (m *Members) RemoveByID(id string) (Member, error) {
	fmt.Printf("remove member by id: %#v\n", id)
	member, index, err := m.GetByID(id)
	if err != nil {
		return Member{}, err
	}

	m.list = append(m.list[:index], m.list[index+1:]...)
	return member, nil
}

func (m *Members) RemoveByConn(conn *websocket.Conn) (Member, error) {
	fmt.Println("remove member by conn")
	member, index, err := m.GetByConn(conn)
	if err != nil {
		return Member{}, err
	}

	m.list = append(m.list[:index], m.list[index+1:]...)
	return member, nil
}
