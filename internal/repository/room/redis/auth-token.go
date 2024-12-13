package redis

import (
	"context"
	"time"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getAuthTokenKey(authToken string) string {
	return "auth-token:" + authToken
}

func (r repo) SetAuthToken(ctx context.Context, params *room.SetAuthTokenParams) error {
	ok, err := r.rc.SetNX(ctx, r.getAuthTokenKey(params.AuthToken), params.MemberID, 10*time.Minute).Result()
	if err != nil {
		return err
	}

	if !ok {
		return room.ErrAuthTokenAlreadyExists
	}

	return nil
}

func (r repo) GetMemberIDByAuthToken(ctx context.Context, authToken string) (string, error) {
	if authToken == "" {
		return "", room.ErrAuthTokenNotFound
	}
	memberID, err := r.rc.Get(ctx, r.getAuthTokenKey(authToken)).Result()
	if err != nil {
		return "", err
	}

	if memberID == "" {
		return "", room.ErrAuthTokenNotFound
	}

	return memberID, nil
}
