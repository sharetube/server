package service

import (
	"errors"

	"github.com/google/uuid"
	"github.com/sharetube/server/internal/domain"
)

var (
	ErrRoomNotFound = errors.New("room not found")
)

type RoomService struct {
	rooms map[string]*domain.Room
}

func NewRoomService() RoomService {
	return RoomService{
		rooms: make(map[string]*domain.Room),
	}
}

func (s RoomService) GetRoom(roomID string) (*domain.Room, error) {
	room, ok := s.rooms[roomID]
	if !ok {
		return nil, ErrRoomNotFound
	}

	return room, nil
}

func (s *RoomService) CreateRoom(creator *domain.Member, initialVideoURL string) (string, *domain.Room) {
	roomID := uuid.NewString()
	for s.rooms[roomID] != nil {
		roomID = uuid.NewString()
	}

	room := domain.NewRoom(creator, initialVideoURL)
	s.rooms[roomID] = room
	room.Start(creator.Conn)

	return roomID, room
}
