package room

import (
	"context"
	"fmt"
)

func (s service) getDefaultPlayerPlaybackRate() float64 {
	return 1.0
}

func (s service) getDefaultPlayerCurrentTime() int {
	return 0
}

func (s service) getDefaultPlayerIsPlaying() bool {
	return false
}

func (s service) checkIfMemberAdmin(ctx context.Context, roomId, memberId string) error {
	isAdmin, err := s.roomRepo.GetMemberIsAdmin(ctx, roomId, memberId)
	if err != nil {
		return fmt.Errorf("failed to get member is admin", "error", err)
	}

	if !isAdmin {
		return ErrPermissionDenied
	}

	return nil
}
