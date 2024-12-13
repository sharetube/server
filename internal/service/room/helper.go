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

func (s service) checkIfMemberAdmin(ctx context.Context, roomID, memberID string) error {
	isAdmin, err := s.roomRepo.GetMemberIsAdmin(ctx, roomID, memberID)
	if err != nil {
		return err
	}

	if !isAdmin {
		return ErrPermissionDenied
	}

	return nil
}
