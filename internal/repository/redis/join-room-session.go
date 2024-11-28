package redis

import (
	"context"
	"fmt"
	"time"
)

const joinRoomSessinPrefix = "join-room-session"

type JoinRoomSession struct {
	Username  string `redis:"username"`
	Color     string `redis:"color"`
	AvatarURL string `redis:"avatar_url"`
	RoomID    string `redis:"room_id"`
}

type SetJoinRoomSessionParams struct {
	ID        string
	Username  string
	Color     string
	AvatarURL string
	RoomID    string
}

func (r Repo) SetJoinRoomSession(ctx context.Context, params *SetJoinRoomSessionParams) error {
	pipe := r.rc.TxPipeline()

	joinRoomSession := JoinRoomSession{
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		RoomID:    params.RoomID,
	}
	joinRoomSessionKey := joinRoomSessinPrefix + ":" + params.ID
	if err := r.HSetIfNotExists(ctx, pipe, joinRoomSessionKey, joinRoomSession); err != nil {

	}
	pipe.Expire(ctx, joinRoomSessionKey, 10*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func (r Repo) GetJoinRoomSession(ctx context.Context, id string) (JoinRoomSession, error) {
	var joinRoomSession JoinRoomSession
	if err := r.rc.HGetAll(ctx, joinRoomSessinPrefix+":"+id).Scan(&joinRoomSession); err != nil {
		return JoinRoomSession{}, err
	}

	if joinRoomSession.Username == "" {
		return JoinRoomSession{}, fmt.Errorf("join room session not found")
	}

	return joinRoomSession, nil
}
