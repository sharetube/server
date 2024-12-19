package room

type Video struct {
	Id  string `json:"id"`
	URL string `json:"url"`
}

type Member struct {
	Id        string  `json:"id"`
	Username  string  `json:"username"`
	Color     string  `json:"color"`
	AvatarURL *string `json:"avatar_url"`
	IsMuted   bool    `json:"is_muted"`
	IsAdmin   bool    `json:"is_admin"`
	IsReady   bool    `json:"is_ready"`
}

type Playlist struct {
	Videos    []Video `json:"videos"`
	LastVideo *Video  `json:"last_video"`
}

type Player struct {
	VideoURL     string  `json:"video_url"`
	IsPlaying    bool    `json:"is_playing"`
	CurrentTime  int     `json:"current_time"`
	PlaybackRate float64 `json:"playback_rate"`
	UpdatedAt    int     `json:"updated_at"`
}

type PlayerState struct {
	IsPlaying    bool    `json:"is_playing"`
	CurrentTime  int     `json:"current_time"`
	PlaybackRate float64 `json:"playback_rate"`
	UpdatedAt    int     `json:"updated_at"`
}

type Room struct {
	RoomId   string   `json:"room_id"`
	Player   Player   `json:"player"`
	Members  []Member `json:"members"`
	Playlist Playlist `json:"playlist"`
}
