package room

import (
	"context"
	"fmt"
	"time"

	"github.com/sharetube/server/internal/repository"
)

const joinRoomSessinPrefix = "join-room-session"

func (r Repo) SetJoinRoomSession(ctx context.Context, params *repository.SetJoinRoomSessionParams) error {
	pipe := r.rc.TxPipeline()

	joinRoomSession := repository.JoinRoomSession{
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		RoomID:    params.RoomID,
	}
	joinRoomSessionKey := joinRoomSessinPrefix + ":" + params.ID
	if err := r.hSetIfNotExists(ctx, pipe, joinRoomSessionKey, joinRoomSession); err != nil {

	}
	pipe.Expire(ctx, joinRoomSessionKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func (r Repo) GetJoinRoomSession(ctx context.Context, id string) (repository.JoinRoomSession, error) {
	var joinRoomSession repository.JoinRoomSession
	if err := r.rc.HGetAll(ctx, joinRoomSessinPrefix+":"+id).Scan(&joinRoomSession); err != nil {
		return repository.JoinRoomSession{}, err
	}

	if joinRoomSession.Username == "" {
		return repository.JoinRoomSession{}, fmt.Errorf("join room session not found")
	}

	return joinRoomSession, nil
}
