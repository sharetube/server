package room

type Video struct {
	ID        string
	URL       string
	AddedByID string
}

type Member struct {
	ID        string
	Username  string
	Color     string
	AvatarURL string
	IsMuted   bool
	IsAdmin   bool
	IsOnline  bool
}
