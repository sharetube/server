package room

type Video struct {
	URL       string `redis:"url"`
	AddedById string `redis:"added_by"`
}

type RemoveVideoParams struct {
	VideoId string
	RoomId  string
}

type SetVideoParams struct {
	VideoId   string
	RoomId    string
	URL       string
	AddedById string
	Version   int
}

type GetVideoParams struct {
	VideoId string
	RoomId  string
}
