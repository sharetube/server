package service

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

func (s service) getDefaultPlayerIsEnded() bool {
	return false
}

func (s service) getDefaultPlayerWaitingForReady() bool {
	return false
}

func (s service) getDefaultMemberIsMuted() bool {
	return false
}

func (s service) getDefaultMemberIsReady() bool {
	return false
}

func (s service) checkIfMemberAdmin(ctx context.Context, roomId, memberId string) error {
	isAdmin, err := s.roomRepo.GetMemberIsAdmin(ctx, roomId, memberId)
	if err != nil {
		return fmt.Errorf("failed to get member is admin: %w", err)
	}

	if !isAdmin {
		return ErrPermissionDenied
	}

	return nil
}
