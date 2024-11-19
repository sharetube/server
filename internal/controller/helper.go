package controller

import (
	"fmt"
	"net/http"

	"github.com/sharetube/server/internal/domain"
)

func (c Controller) getQueryParam(r *http.Request, key string) (string, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return "", fmt.Errorf("param %s was not provided", key)
	}

	return value, nil
}

func (c Controller) getUser(r *http.Request) (*domain.Member, error) {
	username, err := c.getQueryParam(r, "username")
	if err != nil {
		return nil, err
	}

	color, err := c.getQueryParam(r, "color")
	if err != nil {
		return nil, err
	}

	avatarURL, err := c.getQueryParam(r, "avatar-url")
	if err != nil {
		return nil, err
	}

	// userID := uuid.NewString()
	userID := username

	return &domain.Member{
		ID:        userID,
		Username:  username,
		Color:     color,
		AvatarURL: avatarURL,
	}, nil
}
