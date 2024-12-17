package room

import "context"

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
		s.logger.ErrorContext(ctx, "failed to get member is admin", "error", err)
		return err
	}

	if !isAdmin {
		return ErrPermissionDenied
	}

	return nil
}
