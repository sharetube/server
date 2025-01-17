package service

type Video struct {
	Id           int    `json:"id"`
	Url          string `json:"url"`
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	ThumbnailUrl string `json:"thumbnail_url"`
}

type Member struct {
	Id        string  `json:"id"`
	Username  string  `json:"username"`
	Color     string  `json:"color"`
	AvatarUrl *string `json:"avatar_url"`
	IsMuted   bool    `json:"is_muted"`
	IsAdmin   bool    `json:"is_admin"`
	IsReady   bool    `json:"is_ready"`
}

type Playlist struct {
	Videos       []Video `json:"videos"`
	LastVideo    *Video  `json:"last_video"`
	CurrentVideo Video   `json:"current_video"`
	Version      int     `json:"version"`
}

type Player struct {
	IsPlaying    bool    `json:"is_playing"`
	CurrentTime  int     `json:"current_time"`
	PlaybackRate float64 `json:"playback_rate"`
	UpdatedAt    int     `json:"updated_at"`
}

type Room struct {
	Id           string   `json:"id"`
	Player       Player   `json:"player"`
	IsVideoEnded bool     `json:"is_video_ended"`
	Members      []Member `json:"members"`
	Playlist     Playlist `json:"playlist"`
}
