package room

import (
	"context"
	"log/slog"

	"github.com/gorilla/websocket"
)

func (s service) getConnsByRoomID(ctx context.Context, roomID string) ([]*websocket.Conn, error) {
	slog.Debug("Service getConnsByRoomID", "roomID", roomID)
	memberIDs, err := s.roomRepo.GetMembersIDs(ctx, roomID)
	if err != nil {
		slog.Info("failed to get member ids", "err", err)
		return nil, err
	}

	conns := make([]*websocket.Conn, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		conn, err := s.connRepo.GetConn(memberID)
		if err != nil {
			slog.Info("failed to get conn", "err", err)
			return nil, err
		}

		conns = append(conns, conn)
	}

	return conns, nil
}
