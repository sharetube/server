package room

type Video struct {
	URL string `redis:"url"`
}

type RemoveVideoParams struct {
	VideoId string
	RoomId  string
}

type SetVideoParams struct {
	VideoId string
	RoomId  string
	URL     string
	// Version   int
}

type GetVideoParams struct {
	VideoId string
	RoomId  string
}
