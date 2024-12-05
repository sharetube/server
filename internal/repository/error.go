package repository

import "errors"

var (
	ErrMemberNotFound         = errors.New("member not found")
	ErrAuthTokenAlreadyExists = errors.New("auth token already exists")
	ErrPlayerNotFound         = errors.New("player not found")
	ErrVideoNotFound          = errors.New("video not found")
	ErrAuthTokenNotFound      = errors.New("auth token not found")
)
