package room

import (
	"context"
	"log/slog"
	"time"

	"github.com/sharetube/server/internal/repository"
)

const authTokenPrefix = "auth-token"

func (r repo) SetAuthToken(ctx context.Context, params *repository.SetAuthTokenParams) error {
	slog.Debug("SetAuthToken", "params", params)
	ok, err := r.rc.SetNX(ctx, authTokenPrefix+":"+params.AuthToken, params.MemberID, 10*time.Minute).Result()
	if err != nil {
		slog.Info("SetAuthToken", "error", err)
		return err
	}

	if !ok {
		return repository.ErrAuthTokenAlreadyExists
	}

	return nil
}

func (r repo) GetMemberIDByAuthToken(ctx context.Context, authToken string) (string, error) {
	if authToken == "" {
		return "", repository.ErrAuthTokenNotFound
	}
	memberID, err := r.rc.Get(ctx, authTokenPrefix+":"+authToken).Result()
	if err != nil {
		return "", err
	}

	if memberID == "" {
		return "", repository.ErrAuthTokenNotFound
	}

	return memberID, nil
}
