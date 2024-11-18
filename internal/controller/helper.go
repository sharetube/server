package controller

import (
	"fmt"
	"net/http"

	"github.com/sharetube/server/internal/domain"
)

const (
	headerPrefix = "St-"
)

func (c Controller) getHeader(r *http.Request, key string) (string, error) {
	value := r.Header.Get(headerPrefix + key)
	if value == "" {
		return "", fmt.Errorf("header %s was not provided", key)
	}

	return value, nil
}

func (c Controller) getUser(r *http.Request) (*domain.Member, error) {
	username, err := c.getHeader(r, "Username")
	if err != nil {
		return nil, err
	}

	color, err := c.getHeader(r, "Color")
	if err != nil {
		return nil, err
	}

	avatarURL, err := c.getHeader(r, "Avatar-Url")
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
