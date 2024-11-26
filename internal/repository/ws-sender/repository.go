package wssender

import (
	"errors"

	"github.com/gorilla/websocket"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

type Repo struct {
	connList map[*websocket.Conn]string
	idList   map[string]*websocket.Conn
}

func NewRepo() *Repo {
	return &Repo{
		connList: make(map[*websocket.Conn]string),
		idList:   make(map[string]*websocket.Conn),
	}
}

func (r *Repo) Add(conn *websocket.Conn, memberID string) error {
	if r.connList[conn] != "" || r.idList[memberID] != nil {
		return ErrAlreadyExists
	}

	r.connList[conn] = memberID
	r.idList[memberID] = conn
	return nil
}

func (r *Repo) RemoveByConn(conn *websocket.Conn) error {
	memberID, ok := r.connList[conn]
	if !ok {
		return ErrNotFound
	}

	delete(r.connList, conn)
	delete(r.idList, memberID)

	return nil
}

func (r *Repo) RemoveByMemberID(memberID string) error {
	conn, ok := r.idList[memberID]
	if !ok {
		return ErrNotFound
	}

	delete(r.connList, conn)
	delete(r.idList, memberID)

	return nil
}

func (r *Repo) GetMemberID(conn *websocket.Conn) (string, error) {
	memberID, ok := r.connList[conn]
	if !ok {
		return "", ErrNotFound
	}

	return memberID, nil
}

func (r *Repo) GetConn(memberID string) (*websocket.Conn, error) {
	conn, ok := r.idList[memberID]
	if !ok {
		return nil, ErrNotFound
	}

	return conn, nil
}
