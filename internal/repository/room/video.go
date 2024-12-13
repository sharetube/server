package room

type Video struct {
	URL       string `redis:"url"`
	AddedByID string `redis:"added_by"`
	RoomID    string `redis:"room_id"`
}

type RemoveVideoParams struct {
	VideoID string
	RoomID  string
}

type SetVideoParams struct {
	VideoID   string
	RoomID    string
	URL       string
	AddedByID string
	Version   int
}

type GetVideoParams struct {
	VideoID string
	RoomID  string
}
