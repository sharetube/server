package room

import "time"

type Player struct {
	VideoId         int
	IsPlaying       bool
	WaitingForReady bool
	IsEnded         bool
	CurrentTime     int
	PlaybackRate    float64
	UpdatedAt       int
}

type SetPlayerParams struct {
	VideoId         int
	IsPlaying       bool
	WaitingForReady bool
	IsEnded         bool
	CurrentTime     int
	PlaybackRate    float64
	UpdatedAt       int
	RoomId          string
}

type ExpirePlayerParams struct {
	RoomId   string
	ExpireAt time.Time
}
