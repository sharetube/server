package room

type Video struct {
	URL       string `redis:"url"`
	AddedById string `redis:"added_by"`
	RoomId    string `redis:"room_id"`
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
