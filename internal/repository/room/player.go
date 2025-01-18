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

type SetVideoEndedParams struct {
	RoomId     string
	VideoEnded bool
}

type ExpireVideoEndedParams struct {
	RoomId   string
	ExpireAt time.Time
}
