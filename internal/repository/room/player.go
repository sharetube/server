package room

type Player struct {
	VideoId           string  `redis:"video_id"`
	IsPlaying         bool    `redis:"is_playing"`
	SupposedIsPlaying bool    `redis:"supposed_is_playing"`
	CurrentTime       int     `redis:"current_time"`
	PlaybackRate      float64 `redis:"playback_rate"`
	UpdatedAt         int     `redis:"updated_at"`
}

type SetPlayerParams struct {
	VideoId      string  `json:"video_id"`
	IsPlaying    bool    `json:"is_playing"`
	CurrentTime  int     `json:"current_time"`
	PlaybackRate float64 `json:"playback_rate"`
	UpdatedAt    int     `json:"updated_at"`
	RoomId       string  `json:"room_id"`
}

type UpdatePlayerParams struct {
	VideoId      string  `json:"video_id"`
	IsPlaying    bool    `json:"is_playing"`
	CurrentTime  int     `json:"current_time"`
	PlaybackRate float64 `json:"playback_rate"`
	UpdatedAt    int     `json:"updated_at"`
	RoomId       string  `json:"room_id"`
}

type UpdatePlayerStateParams struct {
	IsPlaying    bool    `json:"is_playing"`
	CurrentTime  int     `json:"current_time"`
	PlaybackRate float64 `json:"playback_rate"`
	UpdatedAt    int     `json:"updated_at"`
	RoomId       string  `json:"room_id"`
}
