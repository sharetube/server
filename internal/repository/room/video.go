package room

type Video struct {
	URL string `redis:"url"`
}

type RemoveVideoParams struct {
	VideoId string `json:"video_id"`
	RoomId  string `json:"room_id"`
}

type SetVideoParams struct {
	VideoId string `json:"video_id"`
	RoomId  string `json:"room_id"`
	URL     string `json:"url"`
	// Version   int
}

type GetVideoParams struct {
	VideoId string `json:"video_id"`
	RoomId  string `json:"room_id"`
}
