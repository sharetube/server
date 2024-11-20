package domain

type Player struct {
	CurrentVideoURL string  `json:"current_video_url"`
	IsPlaying       bool    `json:"is_playing"`
	CurrentTime     float64 `json:"current_time"`
	PlaybackRate    float64 `json:"playback_rate"`
}

func NewPlayer(initialVideoURL string) *Player {
	return &Player{
		CurrentVideoURL: initialVideoURL,
		IsPlaying:       false,
		CurrentTime:     0,
		PlaybackRate:    1,
	}
}
