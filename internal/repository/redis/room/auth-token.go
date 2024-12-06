package room

import (
	"context"
	"log/slog"
	"time"

	"github.com/sharetube/server/internal/repository"
)

func (r repo) getAuthTokenKey(authToken string) string {
	return "auth-token:" + authToken
}

func (r repo) SetAuthToken(ctx context.Context, params *repository.SetAuthTokenParams) error {
	funcName := "RedisRepo:SetAuthToken"
	slog.DebugContext(ctx, funcName, "params", params)
	ok, err := r.rc.SetNX(ctx, r.getAuthTokenKey(params.AuthToken), params.MemberID, 10*time.Minute).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return err
	}

	if !ok {
		slog.DebugContext(ctx, funcName, "error", repository.ErrAuthTokenAlreadyExists)
		return repository.ErrAuthTokenAlreadyExists
	}

	return nil
}

func (r repo) GetMemberIDByAuthToken(ctx context.Context, authToken string) (string, error) {
	funcName := "RedisRepo:GetMemberIDByAuthToken"
	slog.DebugContext(ctx, funcName, "authToken", authToken)
	if authToken == "" {
		slog.DebugContext(ctx, funcName, "error", repository.ErrAuthTokenNotFound)
		return "", repository.ErrAuthTokenNotFound
	}
	memberID, err := r.rc.Get(ctx, r.getAuthTokenKey(authToken)).Result()
	if err != nil {
		slog.ErrorContext(ctx, funcName, "error", err)
		return "", err
	}

	if memberID == "" {
		slog.DebugContext(ctx, funcName, "error", "memberID is empty")
		return "", repository.ErrAuthTokenNotFound
	}

	slog.DebugContext(ctx, funcName, "memberID", memberID)
	return memberID, nil
}
