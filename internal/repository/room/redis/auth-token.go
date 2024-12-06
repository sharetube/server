package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/sharetube/server/internal/repository/room"
)

func (r repo) getAuthTokenKey(authToken string) string {
	return "auth-token:" + authToken
}

func (r repo) SetAuthToken(ctx context.Context, params *room.SetAuthTokenParams) error {
	funcName := "room.redis.SetAuthToken"
	slog.DebugContext(ctx, funcName, "params", params)
	ok, err := r.rc.SetNX(ctx, r.getAuthTokenKey(params.AuthToken), params.MemberID, 10*time.Minute).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if !ok {
		slog.DebugContext(ctx, funcName, "error", room.ErrAuthTokenAlreadyExists)
		return room.ErrAuthTokenAlreadyExists
	}

	return nil
}

func (r repo) GetMemberIDByAuthToken(ctx context.Context, authToken string) (string, error) {
	funcName := "room.redis.GetMemberIDByAuthToken"
	slog.DebugContext(ctx, funcName, "authToken", authToken)
	if authToken == "" {
		slog.DebugContext(ctx, funcName, "error", room.ErrAuthTokenNotFound)
		return "", room.ErrAuthTokenNotFound
	}
	memberID, err := r.rc.Get(ctx, r.getAuthTokenKey(authToken)).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return "", err
	}

	if memberID == "" {
		slog.DebugContext(ctx, funcName, "error", "memberID is empty")
		return "", room.ErrAuthTokenNotFound
	}

	slog.DebugContext(ctx, funcName, "memberID", memberID)
	return memberID, nil
}
