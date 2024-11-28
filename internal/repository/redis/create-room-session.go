package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const createRoomSessinPrefix = "create-room-session"

type CreateRoomSession struct {
	Username        string `redis:"username"`
	Color           string `redis:"color"`
	AvatarURL       string `redis:"avatar_url"`
	InitialVideoURL string `redis:"initial_video_url"`
}

type SetCreateRoomSessionParams struct {
	ID              string
	Username        string
	Color           string
	AvatarURL       string
	InitialVideoURL string
}

func (r Repo) SetCreateRoomSession(ctx context.Context, params *SetCreateRoomSessionParams) error {
	pipe := r.rc.TxPipeline()

	createRoomSession := CreateRoomSession{
		Username:        params.Username,
		Color:           params.Color,
		AvatarURL:       params.AvatarURL,
		InitialVideoURL: params.InitialVideoURL,
	}
	createRoomSessionKey := createRoomSessinPrefix + ":" + params.ID
	r.HSetIfNotExists(ctx, pipe, createRoomSessionKey, createRoomSession)
	pipe.Expire(ctx, createRoomSessionKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func (r Repo) GetCreateRoomSession(ctx context.Context, createRoomSessionID string) (CreateRoomSession, error) {
	var createRoomSession CreateRoomSession
	if err := r.rc.HGetAll(ctx, createRoomSessinPrefix+":"+createRoomSessionID).Scan(&createRoomSession); err != nil {
		return CreateRoomSession{}, err
	}

	if createRoomSession.Username == "" {
		return CreateRoomSession{}, fmt.Errorf("create room session not found")
	}

	slog.Info("create room session", "createRoomSession", createRoomSession)

	return createRoomSession, nil
}
