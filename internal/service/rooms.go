package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sharetube/server/internal/domain"
)

var (
	ErrRoomNotFound = errors.New("room not found")
)

type RoomService struct {
	rooms         map[string]*domain.Room
	membersLimit  int
	playlistLimit int
}

func NewRoomService(membersLimit, playlistLimit int) RoomService {
	return RoomService{
		rooms:         make(map[string]*domain.Room),
		membersLimit:  membersLimit,
		playlistLimit: playlistLimit,
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

	room := domain.NewRoom(creator, initialVideoURL, s.membersLimit, s.playlistLimit)
	s.rooms[roomID] = room
	room.SendMessageToAllMembers(&domain.Message{
		Action: "room_created",
		Data: map[string]any{
			"room_id": roomID,
		},
	})

	go room.SendStateToAllMembersPeriodically(10 * time.Second)
	go func() {
		room.HandleMessages()
		delete(s.rooms, roomID)
		fmt.Println("room deleted")
	}()

	return roomID, room
}
