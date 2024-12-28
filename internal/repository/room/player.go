package room

import "time"

type Player struct {
	VideoId         string  `redis:"video_id"`
	IsPlaying       bool    `redis:"is_playing"`
	WaitingForReady bool    `redis:"waiting_for_ready"`
	CurrentTime     int     `redis:"current_time"`
	PlaybackRate    float64 `redis:"playback_rate"`
	UpdatedAt       int     `redis:"updated_at"`
}

type SetPlayerParams struct {
	VideoId         string
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

type UpdatePlayerParams struct {
	VideoId         string
	IsPlaying       bool
	CurrentTime     int
	WaitingForReady bool
	PlaybackRate    float64
	UpdatedAt       int
	RoomId          string
}

type UpdatePlayerStateParams struct {
	IsPlaying       bool
	CurrentTime     int
	PlaybackRate    float64
	WaitingForReady bool
	UpdatedAt       int
	RoomId          string
}
