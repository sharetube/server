package room

import "time"

type Player struct {
	IsPlaying       bool
	WaitingForReady bool
	CurrentTime     int
	PlaybackRate    float64
	UpdatedAt       int
}

type SetPlayerParams struct {
	IsPlaying       bool
	WaitingForReady bool
	CurrentTime     int
	PlaybackRate    float64
	UpdatedAt       int
	RoomId          string
}

type ExpirePlayerParams struct {
	RoomId   string
	ExpireAt time.Time
}

type SetIsVideoEndedParams struct {
	RoomId       string
	IsVideoEnded bool
}

type ExpireIsVideoEndedParams struct {
	RoomId   string
	ExpireAt time.Time
}
