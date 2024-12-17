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

func (r *repo) Add(conn *websocket.Conn, memberId string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connList[conn] != "" || r.idList[memberId] != nil {
		return connection.ErrAlreadyExists
	}

	r.connList[conn] = memberId
	r.idList[memberId] = conn

	return nil
}

func (r *repo) RemoveByConn(conn *websocket.Conn) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	memberId, ok := r.connList[conn]
	if !ok {
		return "", connection.ErrNotFound
	}

	delete(r.connList, conn)
	delete(r.idList, memberId)

	return memberId, nil
}

func (r *repo) RemoveByMemberId(memberId string) (*websocket.Conn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, ok := r.idList[memberId]
	if !ok {
		return nil, connection.ErrNotFound
	}

	delete(r.connList, conn)
	delete(r.idList, memberId)

	return conn, nil
}

func (r *repo) GetMemberId(conn *websocket.Conn) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	memberId, ok := r.connList[conn]
	if !ok {
		return "", connection.ErrNotFound
	}

	return memberId, nil
}

func (r *repo) GetConn(memberId string) (*websocket.Conn, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conn, ok := r.idList[memberId]
	if !ok {
		return nil, connection.ErrNotFound
	}

	return conn, nil
}
