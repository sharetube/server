package conn

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

type repo struct {
	connList map[*websocket.Conn]string
	idList   map[string]*websocket.Conn
	mu       sync.RWMutex
}

func NewRepo() *repo {
	return &repo{
		connList: make(map[*websocket.Conn]string),
		idList:   make(map[string]*websocket.Conn),
	}
}

func (r *repo) Add(conn *websocket.Conn, memberID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connList[conn] != "" || r.idList[memberID] != nil {
		return ErrAlreadyExists
	}

	r.connList[conn] = memberID
	r.idList[memberID] = conn
	return nil
}

func (r *repo) RemoveByConn(conn *websocket.Conn) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	memberID, ok := r.connList[conn]
	if !ok {
		return ErrNotFound
	}
	conn.Close()

	delete(r.connList, conn)
	delete(r.idList, memberID)

	return nil
}

func (r *repo) RemoveByMemberID(memberID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, ok := r.idList[memberID]
	if !ok {
		return ErrNotFound
	}
	conn.Close()

	delete(r.connList, conn)
	delete(r.idList, memberID)

	return nil
}

func (r *repo) GetMemberID(conn *websocket.Conn) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	memberID, ok := r.connList[conn]
	if !ok {
		return "", ErrNotFound
	}

	return memberID, nil
}

func (r *repo) GetConn(memberID string) (*websocket.Conn, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conn, ok := r.idList[memberID]
	if !ok {
		return nil, ErrNotFound
	}

	return conn, nil
}
