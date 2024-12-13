package room

type Player struct {
	VideoURL     string  `redis:"video_url"`
	IsPlaying    bool    `redis:"is_playing"`
	CurrentTime  int     `redis:"current_time"`
	PlaybackRate float64 `redis:"playback_rate"`
	UpdatedAt    int     `redis:"updated_at"`
}

type SetPlayerParams struct {
	CurrentVideoURL string
	IsPlaying       bool
	CurrentTime     int
	PlaybackRate    float64
	UpdatedAt       int
	RoomID          string
}

type UpdatePlayerParams struct {
	VideoURL     string
	IsPlaying    bool
	CurrentTime  int
	PlaybackRate float64
	UpdatedAt    int
	RoomID       string
}

type UpdatePlayerStateParams struct {
	IsPlaying    bool
	CurrentTime  int
	PlaybackRate float64
	UpdatedAt    int
	RoomID       string
}
