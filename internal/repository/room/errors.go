package room

import "errors"

var (
	ErrMemberNotFound     = errors.New("member not found")
	ErrTokenAlreadyExists = errors.New("auth token already exists")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrVideoNotFound      = errors.New("video not found")
	ErrTokenNotFound      = errors.New("auth token not found")
)
