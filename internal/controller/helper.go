package controller

import (
	"fmt"
	"net/http"
)

func GetUsername(w http.ResponseWriter, r *http.Request) string {
	username := r.URL.Query().Get("username")
	if username == "" {
		fmt.Println("username was not provided")
		fmt.Fprint(w, "username was not provided")
		return ""
	}

	return username
}

func GetColor(w http.ResponseWriter, r *http.Request) string {
	color := r.URL.Query().Get("color")
	if color == "" {
		fmt.Println("color was not provided")
		fmt.Fprint(w, "color was not provided")
		return ""
	}

	return color
}

func GetVideoURL(w http.ResponseWriter, r *http.Request) string {
	videoURL := r.URL.Query().Get("video-url")
	if videoURL == "" {
		fmt.Println("video url was not provided")
		fmt.Fprint(w, "video-url was not provided")
		return ""
	}

	return videoURL
}
