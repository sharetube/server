package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sharetube/server/internal/repository"
)

const createRoomSessinPrefix = "create-room-session"

func (r Repo) SetCreateRoomSession(ctx context.Context, params *repository.SetCreateRoomSessionParams) error {
	pipe := r.rc.TxPipeline()

	createRoomSession := repository.CreateRoomSession{
		Username:        params.Username,
		Color:           params.Color,
		AvatarURL:       params.AvatarURL,
		InitialVideoURL: params.InitialVideoURL,
	}
	createRoomSessionKey := createRoomSessinPrefix + ":" + params.ID
	r.hSetIfNotExists(ctx, pipe, createRoomSessionKey, createRoomSession)
	pipe.Expire(ctx, createRoomSessionKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func (r Repo) GetCreateRoomSession(ctx context.Context, createRoomSessionID string) (repository.CreateRoomSession, error) {
	var createRoomSession repository.CreateRoomSession
	if err := r.rc.HGetAll(ctx, createRoomSessinPrefix+":"+createRoomSessionID).Scan(&createRoomSession); err != nil {
		return repository.CreateRoomSession{}, err
	}

	if createRoomSession.Username == "" {
		return repository.CreateRoomSession{}, fmt.Errorf("create room session not found")
	}

	slog.Info("create room session", "createRoomSession", createRoomSession)

	return createRoomSession, nil
}
