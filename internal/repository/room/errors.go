package room

import "errors"

var (
	ErrMemberNotFound     = errors.New("member not found")
	ErrMemberListNotFound = errors.New("memberlist not found")
	ErrTokenAlreadyExists = errors.New("auth token already exists")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrPlaylistNotFound   = errors.New("playlist not found")
	ErrVideoNotFound      = errors.New("video not found")
	ErrLastVideoNotFound  = errors.New("last video not found")
	ErrTokenNotFound      = errors.New("auth token not found")
)
