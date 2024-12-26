package room

import "time"

type Video struct {
	URL string `redis:"url"`
}

type RemoveVideoParams struct {
	VideoId string `json:"video_id"`
	RoomId  string `json:"room_id"`
}

type ExpireVideoParams struct {
	VideoId  string    `json:"video_id"`
	RoomId   string    `json:"room_id"`
	ExpireAt time.Time `json:"expire_at"`
}

type ExpireLastVideoParams struct {
	RoomId   string    `json:"room_id"`
	ExpireAt time.Time `json:"expire_at"`
}

type ExpirePlaylistParams struct {
	RoomId   string    `json:"room_id"`
	ExpireAt time.Time `json:"expire_at"`
}

type AddVideoToListParams struct {
	RoomId  string `json:"room_id"`
	VideoId string `json:"video_id"`
	URL     string `json:"url"`
}

type SetVideoParams struct {
	VideoId string `json:"video_id"`
	RoomId  string `json:"room_id"`
	URL     string `json:"url"`
	// Version   int
}

type SetLastVideoParams struct {
	VideoId string `json:"video_id"`
	RoomId  string `json:"room_id"`
}

type SetCurrentVideoParams struct {
	VideoId string `json:"video_id"`
	RoomId  string `json:"room_id"`
}

type RemoveVideoFromListParams struct {
	VideoId string `json:"video_id"`
	RoomId  string `json:"room_id"`
}

type GetVideoParams struct {
	VideoId string `json:"video_id"`
	RoomId  string `json:"room_id"`
}
