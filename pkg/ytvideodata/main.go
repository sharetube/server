package ytvideodata

import (
	"errors"
)

type VideoData struct {
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	ThumbnailUrl string `json:"thumbnail_url"`
}

func Get(videoUrl string) (*VideoData, error) {
	videoData, err := getVideoWithEmbed(videoUrl)
	if err != nil {
		if !errors.Is(err, ErrVideoNotEmbeddable) {
			return nil, err
		}

		videoData, err = getFromPage(videoUrl)
		if err != nil {
			return nil, err
		}
	}

	return videoData, nil
}
