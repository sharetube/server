package repository

type CreateRoomSession struct {
	Username        string `redis:"username"`
	Color           string `redis:"color"`
	AvatarURL       string `redis:"avatar_url"`
	InitialVideoURL string `redis:"initial_video_url"`
}

type JoinRoomSession struct {
	Username  string `redis:"username"`
	Color     string `redis:"color"`
	AvatarURL string `redis:"avatar_url"`
	RoomID    string `redis:"room_id"`
}

type Member struct {
	Username  string `redis:"username"`
	Color     string `redis:"color"`
	AvatarURL string `redis:"avatar_url"`
	IsMuted   bool   `redis:"is_muted"`
	IsAdmin   bool   `redis:"is_admin"`
	IsOnline  bool   `redis:"is_online"`
	RoomID    string `redis:"room_id"`
}

type Video struct {
	URL       string `redis:"url"`
	AddedByID string `redis:"added_by"`
	RoomID    string `redis:"room_id"`
}

type Player struct {
	CurrentVideoURL string  `redis:"current_video_url"`
	IsPlaying       bool    `redis:"is_playing"`
	CurrentTime     float64 `redis:"current_time"`
	PlaybackRate    float64 `redis:"playback_rate"`
	UpdatedAt       int64   `redis:"updated_at"`
}
