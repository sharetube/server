package wssender

import (
	"errors"

	"github.com/gorilla/websocket"
)

type Repo struct {
	list map[*websocket.Conn]string
}

func NewRepo() *Repo {
	return &Repo{
		list: make(map[*websocket.Conn]string),
	}
}

func (r *Repo) Add(conn *websocket.Conn, memberID string) error {
	if r.list[conn] != "" {
		return errors.New("connection already exists")
	}

	r.list[conn] = memberID
	return nil
}

func (r *Repo) Remove(conn *websocket.Conn) {
	delete(r.list, conn)
}

func (r *Repo) GetMemberID(conn *websocket.Conn) (string, error) {
	memberID, ok := r.list[conn]
	if !ok {
		return "", errors.New("connection not found")
	}

	return memberID, nil
}
