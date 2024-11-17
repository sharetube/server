package domain

import (
	"errors"
	"fmt"
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
	list          []Video
	previousVideo *Video
	lastID        int
	limit         int
}

func NewPlaylist(initialVideoURL, addedBy string, limit int) *Playlist {
	return &Playlist{
		list: []Video{
			{
				ID:        1,
				URL:       initialVideoURL,
				AddedByID: addedBy,
				WasPlayed: false,
			},
		},
		lastID: 1,
		limit:  limit,
	}
}

func (p Playlist) AsList() []Video {
	return p.list
}

func (p Playlist) Length() int {
	return len(p.list)
}

func (p Playlist) GetByID(id int) (Video, int, error) {
	fmt.Printf("get video by id: %#v\n", id)
	for index, video := range p.list {
		if video.ID == id {
			return video, index, nil
		}
	}

	return Video{}, 0, fmt.Errorf("get video by id: %w", ErrVideoNotFound)
}

func (p *Playlist) Add(addedBy, url string) (Video, error) {
	fmt.Printf("add video: %s, %s\n", addedBy, url)
	if p.Length() >= p.limit {
		return Video{}, fmt.Errorf("add video: %w", ErrPlaylistLimitReached)
	}

	p.lastID++
	video := Video{
		ID:        p.lastID,
		URL:       url,
		AddedByID: addedBy,
	}
	p.list = append(p.list, video)

	return video, nil
}

func (p *Playlist) RemoveByID(id int) (Video, error) {
	fmt.Printf("remove video by id: %#v\n", id)
	member, index, err := p.GetByID(id)
	if err != nil {
		return Video{}, fmt.Errorf("remove video by id: %w", err)
	}

	fmt.Printf("removed video index: %v\n", index)
	p.list = append(p.list[:index], p.list[index+1:]...)
	fmt.Printf("video list: %v\n", p.list)
	return member, nil
}
