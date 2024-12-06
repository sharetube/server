package conn

import (
	"errors"
	"log/slog"
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
	funcName := "inmemory.Add"
	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Debug(funcName, "memberID", memberID)
	if r.connList[conn] != "" || r.idList[memberID] != nil {
		slog.Info(funcName, "error", ErrAlreadyExists)
		return ErrAlreadyExists
	}

	r.connList[conn] = memberID
	r.idList[memberID] = conn

	slog.Debug(funcName, "result", "OK")
	return nil
}

func (r *repo) RemoveByConn(conn *websocket.Conn) error {
	funcName := "inmemory.RemoveByConn"
	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Debug(funcName)
	memberID, ok := r.connList[conn]
	if !ok {
		slog.Info(funcName, "error", ErrNotFound)
		return ErrNotFound
	}
	conn.Close()

	delete(r.connList, conn)
	delete(r.idList, memberID)

	slog.Debug(funcName, "result", memberID)
	return nil
}

func (r *repo) RemoveByMemberID(memberID string) error {
	funcName := "inmemory.RemoveByMemberID"
	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Debug(funcName, "memberID", memberID)
	conn, ok := r.idList[memberID]
	if !ok {
		slog.Info(funcName, "error", ErrNotFound)
		return ErrNotFound
	}
	conn.Close()

	delete(r.connList, conn)
	delete(r.idList, memberID)

	slog.Debug(funcName, "result", "OK")
	return nil
}

func (r *repo) GetMemberID(conn *websocket.Conn) (string, error) {
	funcName := "inmemory.GetMemberID"
	r.mu.RLock()
	defer r.mu.RUnlock()

	slog.Debug(funcName)
	memberID, ok := r.connList[conn]
	if !ok {
		slog.Info(funcName, "error", ErrNotFound)
		return "", ErrNotFound
	}

	slog.Debug(funcName, "result", memberID)
	return memberID, nil
}

func (r *repo) GetConn(memberID string) (*websocket.Conn, error) {
	funcName := "inmemory.GetConn"
	r.mu.RLock()
	defer r.mu.RUnlock()

	slog.Debug(funcName, "memberID", memberID)
	conn, ok := r.idList[memberID]
	if !ok {
		slog.Info(funcName, "error", ErrNotFound)
		return nil, ErrNotFound
	}

	slog.Debug(funcName, "result", "OK")
	return conn, nil
}
