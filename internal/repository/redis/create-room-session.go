package redis

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CreateRoomSessionRepo struct {
	rc         *redis.Client
	expireTime time.Duration
}

const (
	createRoomSessionPrefix = "create-room-session"
)

func NewCreateRoomSessionRepo(rc *redis.Client, expireTime time.Duration) *CreateRoomSessionRepo {
	return &CreateRoomSessionRepo{
		rc:         rc,
		expireTime: expireTime,
	}
}

func (r CreateRoomSessionRepo) Set(ctx context.Context, value string) (string, error) {
	key := uuid.NewString()
	return key, r.rc.Set(ctx, createRoomSessionPrefix+key, value, r.expireTime).Err()
}
