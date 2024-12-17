package room

type Member struct {
	Username  string  `redis:"username"`
	Color     string  `redis:"color"`
	AvatarURL *string `redis:"avatar_url"`
	IsMuted   bool    `redis:"is_muted"`
	IsAdmin   bool    `redis:"is_admin"`
	IsOnline  bool    `redis:"is_online"`
	RoomID    string  `redis:"room_id"`
}

type AddMemberToListParams struct {
	MemberID string
	RoomID   string
}

type GetMemberParams struct {
	MemberID string
	RoomID   string
}

type SetMemberParams struct {
	MemberID  string
	Username  string
	Color     string
	AvatarURL *string
	IsMuted   bool
	IsAdmin   bool
	IsOnline  bool
	RoomID    string
}

type RemoveMemberParams struct {
	MemberID string
	RoomID   string
}

type RemoveMemberFromListParams struct {
	MemberID string
	RoomID   string
}
