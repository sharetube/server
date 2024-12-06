package room

type SetCreateRoomSessionParams struct {
	ID              string
	Username        string
	Color           string
	AvatarURL       string
	InitialVideoURL string
}

type SetJoinRoomSessionParams struct {
	ID        string
	Username  string
	Color     string
	AvatarURL string
	RoomID    string
}

type SetMemberParams struct {
	MemberID  string
	Username  string
	Color     string
	AvatarURL string
	IsMuted   bool
	IsAdmin   bool
	IsOnline  bool
	RoomID    string
}

type RemoveMemberParams struct {
	MemberID string
	RoomID   string
}

type SetPlayerParams struct {
	CurrentVideoURL string
	IsPlaying       bool
	CurrentTime     float64
	PlaybackRate    float64
	UpdatedAt       int64
	RoomID          string
}

type SetVideoParams struct {
	VideoID   string
	RoomID    string
	URL       string
	AddedByID string
	Version   int
}

type UpdatePlayerStateParams struct {
	IsPlaying    bool
	CurrentTime  float64
	PlaybackRate float64
	UpdatedAt    int64
	RoomID       string
}

type RemoveVideoParams struct {
	VideoID string
	RoomID  string
}

type SetAuthTokenParams struct {
	AuthToken string
	MemberID  string
}

type AddMemberToListParams struct {
	MemberID string
	RoomID   string
}
