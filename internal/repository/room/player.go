package room

import "time"

type Player struct {
	VideoId   string `redis:"video_id"`
	IsPlaying bool   `redis:"is_playing"`
	// SupposedIsPlaying bool    `redis:"supposed_is_playing"`
	CurrentTime  int     `redis:"current_time"`
	PlaybackRate float64 `redis:"playback_rate"`
	UpdatedAt    int     `redis:"updated_at"`
}

type SetPlayerParams struct {
	VideoId      string
	IsPlaying    bool
	CurrentTime  int
	PlaybackRate float64
	UpdatedAt    int
	RoomId       string
}

type ExpirePlayerParams struct {
	RoomId   string
	ExpireAt time.Time
}

type UpdatePlayerParams struct {
	VideoId      string
	IsPlaying    bool
	CurrentTime  int
	PlaybackRate float64
	UpdatedAt    int
	RoomId       string
}

type UpdatePlayerStateParams struct {
	IsPlaying    bool
	CurrentTime  int
	PlaybackRate float64
	UpdatedAt    int
	RoomId       string
}
