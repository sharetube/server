package service

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/sharetube/server/internal/domain"
)

var (
	ErrRoomNotFound = errors.New("room not found")
)

type RoomService struct {
	rooms           map[string]*Room
	membersLimit    int
	playlistLimit   int
	updatesInterval time.Duration
}

func NewRoomService(updatesInterval time.Duration, membersLimit, playlistLimit int) RoomService {
	return RoomService{
		rooms:           make(map[string]*Room),
		membersLimit:    membersLimit,
		playlistLimit:   playlistLimit,
		updatesInterval: updatesInterval,
	}
}

func (s RoomService) GetRoom(roomID string) (*Room, error) {
	room, ok := s.rooms[roomID]
	if !ok {
		return nil, ErrRoomNotFound
	}

	return room, nil
}

func (s *RoomService) CreateRoom(creator *domain.Member, initialVideoURL string) (string, *Room) {
	roomID := uuid.NewString()
	for s.rooms[roomID] != nil {
		roomID = uuid.NewString()
	}

	room := newRoom(creator, initialVideoURL, s.membersLimit, s.playlistLimit)
	s.rooms[roomID] = room

	go room.SendStateToAllMembersPeriodically(s.updatesInterval)
	go func() {
		room.HandleMessages()
		delete(s.rooms, roomID)
		slog.Info("room closed", "room_id", roomID)
	}()

	return roomID, room
}
