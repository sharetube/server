package room

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getConnsByRoomID(ctx context.Context, roomID string) ([]*websocket.Conn, error) {
	memberIDs, err := s.roomRepo.GetMemberIDs(ctx, roomID)
	if err != nil {
		return nil, err
	}

	conns := make([]*websocket.Conn, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		conn, err := s.connRepo.GetConn(memberID)
		if err != nil {
			return nil, err
		}

		conns = append(conns, conn)
	}

	return conns, nil
}

func (s service) deleteRoom(ctx context.Context, roomID string) error {
	s.roomRepo.RemovePlayer(ctx, roomID)
	videoIDs, err := s.roomRepo.GetVideoIDs(ctx, roomID)
	if err != nil {
		return err
	}

	for _, videoID := range videoIDs {
		if err := s.roomRepo.RemoveVideo(ctx, &room.RemoveVideoParams{
			VideoID: videoID,
			RoomID:  roomID,
		}); err != nil {
			return err
		}
	}

	return nil
}
