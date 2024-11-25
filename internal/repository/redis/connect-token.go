package redis

import (
	"context"
	"time"
)

const connectTokenPrefix = "connect-token"

// type CreateMemberParams struct {
// 	ID        string
// 	Username  string
// 	Color     string
// 	AvatarURL string
// 	IsMuted   bool
// 	IsAdmin   bool
// 	IsOnline  bool
// 	RoomID    string
// }

func (r Repo) CreateConnectToken(ctx context.Context, connectTokenID, memberID string) error {
	return r.rc.Set(ctx, connectTokenPrefix+":"+connectTokenID, memberID, 5*time.Minute).Err()
}
