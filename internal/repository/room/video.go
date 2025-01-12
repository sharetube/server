package room

import "time"

type Video struct {
	Url string
}

type RemoveVideoParams struct {
	VideoId int
	RoomId  string
}

type ExpireVideoParams struct {
	VideoId  int
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
	VideoId int
	Url     string
}

type SetVideoParams struct {
	RoomId string
	Url    string
	// Version   int
}

type SetLastVideoParams struct {
	VideoId int
	RoomId  string
}

type SetCurrentVideoParams struct {
	VideoId int
	RoomId  string
}

type RemoveVideoFromListParams struct {
	VideoId int
	RoomId  string
}

type ReorderListParams struct {
	VideoIds []int
	RoomId   string
}

type GetVideoParams struct {
	VideoId int
	RoomId  string
}
