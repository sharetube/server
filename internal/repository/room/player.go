package room

type Player struct {
	VideoURL     string  `redis:"video_url"`
	IsPlaying    bool    `redis:"is_playing"`
	CurrentTime  int     `redis:"current_time"`
	PlaybackRate float64 `redis:"playback_rate"`
	UpdatedAt    int     `redis:"updated_at"`
}

type SetPlayerParams struct {
	CurrentVideoURL string  `json:"current_video_url"`
	IsPlaying       bool    `json:"is_playing"`
	CurrentTime     int     `json:"current_time"`
	PlaybackRate    float64 `json:"playback_rate"`
	UpdatedAt       int     `json:"updated_at"`
	RoomId          string  `json:"room_id"`
}

type UpdatePlayerParams struct {
	VideoURL     string  `json:"video_url"`
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
