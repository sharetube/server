package domain

import "time"

type Player struct {
	CurrentVideoURL string  `json:"current_video_url"`
	IsPlaying       bool    `json:"is_playing"`
	CurrentTime     float64 `json:"current_time"`
	PlaybackRate    float64 `json:"playback_rate"`
	UpdatedAt       int64   `json:"updated_at"`
}

func (p Player) CalcCurrentTime() float64 {
	if p.IsPlaying {
		return (float64(time.Now().UnixMilli()-p.UpdatedAt) * p.PlaybackRate) + p.CurrentTime
	}

	return p.CurrentTime
}

func NewPlayer(initialVideoURL string) *Player {
	return &Player{
		CurrentVideoURL: initialVideoURL,
		IsPlaying:       false,
		CurrentTime:     0,
		PlaybackRate:    1,
	}
}
