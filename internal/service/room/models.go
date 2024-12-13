package room

type Video struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	AddedByID string `json:"added_by_id"`
}

type Member struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Color     string `json:"color"`
	AvatarURL string `json:"avatar_url"`
	IsMuted   bool   `json:"is_muted"`
	IsAdmin   bool   `json:"is_admin"`
	IsOnline  bool   `json:"is_online"`
}

type Playlist struct {
	Videos        []Video `json:"videos"`
	PreviousVideo *Video  `json:"previous_video"`
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

type RoomState struct {
	RoomID     string   `json:"room_id"`
	Player     Player   `json:"player"`
	MemberList []Member `json:"member_list"`
	Playlist   Playlist `json:"playlist"`
}
