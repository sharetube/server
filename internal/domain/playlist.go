package domain

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

var (
	ErrVideoNotFound        = errors.New("video not found")
	ErrPlaylistLimitReached = errors.New("playlist limit reached")
)

type Video struct {
	ID        int    `json:"id"`
	URL       string `json:"url"`
	AddedByID string `json:"added_by"`
	WasPlayed bool   `json:"was_played"`
}

// todo: implement pagination
type Playlist struct {
	videos        map[int]*Video
	previousVideo *Video
	lastIndex     int
	limit         int
}

func NewPlaylist(initialVideoURL, addedBy string, limit int) *Playlist {
	return &Playlist{
		videos: map[int]*Video{
			1: {
				ID:        1,
				URL:       initialVideoURL,
				AddedByID: addedBy,
				WasPlayed: false,
			},
		},
		lastIndex: 2,
		limit:     limit,
	}
}

func (p Playlist) AsList() []*Video {
	return slices.Collect(maps.Values(p.videos))
}

func (p Playlist) Length() int {
	return len(p.videos)
}

func (p *Playlist) Add(addedBy, url string) (Video, error) {
	fmt.Printf("add video: %s, %s\n", addedBy, url)
	if p.Length() >= p.limit {
		return Video{}, ErrPlaylistLimitReached
	}

	p.lastIndex++
	video := Video{
		ID:        p.lastIndex,
		URL:       url,
		AddedByID: addedBy,
	}
	p.videos[p.lastIndex] = &video

	return video, nil
}

func (p *Playlist) Remove(videoIndex int) (Video, error) {
	fmt.Printf("remove video: %d\n", videoIndex)
	video := p.videos[videoIndex]
	if video == nil {
		return Video{}, ErrVideoNotFound
	}

	delete(p.videos, videoIndex)
	return *video, nil
}
