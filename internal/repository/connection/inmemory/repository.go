package inmemory

import (
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/connection"
)

type repo struct {
	connList map[*websocket.Conn]string
	idList   map[string]*websocket.Conn
	mu       sync.RWMutex
	logger   *slog.Logger
}

func NewRepo(logger *slog.Logger) *repo {
	return &repo{
		connList: make(map[*websocket.Conn]string),
		idList:   make(map[string]*websocket.Conn),
		logger:   logger,
	}
}

func (r *repo) Add(conn *websocket.Conn, memberID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connList[conn] != "" || r.idList[memberID] != nil {
		return connection.ErrAlreadyExists
	}

	r.connList[conn] = memberID
	r.idList[memberID] = conn

	return nil
}

func (r *repo) RemoveByConn(conn *websocket.Conn) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	memberID, ok := r.connList[conn]
	if !ok {
		return "", connection.ErrNotFound
	}

	delete(r.connList, conn)
	delete(r.idList, memberID)

	return memberID, nil
}

func (r *repo) RemoveByMemberID(memberID string) (*websocket.Conn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, ok := r.idList[memberID]
	if !ok {
		return nil, connection.ErrNotFound
	}

	delete(r.connList, conn)
	delete(r.idList, memberID)

	return conn, nil
}

func (r *repo) GetMemberID(conn *websocket.Conn) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	memberID, ok := r.connList[conn]
	if !ok {
		return "", connection.ErrNotFound
	}

	return memberID, nil
}

func (r *repo) GetConn(memberID string) (*websocket.Conn, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conn, ok := r.idList[memberID]
	if !ok {
		return nil, connection.ErrNotFound
	}

	return conn, nil
}
