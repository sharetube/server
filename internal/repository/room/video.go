package room

import "time"

type Video struct {
	Url string
}

type RemoveVideoParams struct {
	VideoId string
	RoomId  string
}

type ExpireVideoParams struct {
	VideoId  string
	RoomId   string
	ExpireAt time.Time
}

type ExpireLastVideoParams struct {
	RoomId   string
	ExpireAt time.Time
}

type ExpirePlaylistParams struct {
	RoomId   string
	ExpireAt time.Time
}

type AddVideoToListParams struct {
	RoomId  string
	VideoId string
	Url     string
}

type SetVideoParams struct {
	VideoId string
	RoomId  string
	Url     string
	// Version   int
}

type SetLastVideoParams struct {
	VideoId string
	RoomId  string
}

type SetCurrentVideoParams struct {
	VideoId string
	RoomId  string
}

type RemoveVideoFromListParams struct {
	VideoId string
	RoomId  string
}

type ReorderListParams struct {
	VideoIds []string
	RoomId   string
}

type GetVideoParams struct {
	VideoId string
	RoomId  string
}
