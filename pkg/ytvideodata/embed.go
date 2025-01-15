package ytvideodata

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	ErrVideoNotFound      = fmt.Errorf("video not found")
	ErrVideoNotEmbeddable = fmt.Errorf("video is not embeddable")
)

func getVideoWithEmbed(videoId string) (*VideoData, error) {
	url := fmt.Sprintf("https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=%s", videoId)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return nil, ErrVideoNotFound
		case http.StatusUnauthorized:
			return nil, ErrVideoNotEmbeddable
		default:
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	}

	var result VideoData
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, err
}
