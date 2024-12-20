package room

type Member struct {
	Username  string  `redis:"username" json:"username"`
	Color     string  `redis:"color" json:"color"`
	AvatarURL *string `redis:"avatar_url" json:"avatar_url"`
	IsMuted   bool    `redis:"is_muted" json:"is_muted"`
	IsAdmin   bool    `redis:"is_admin" json:"is_admin"`
	IsReady   bool    `redis:"is_ready" json:"is_ready"`
}

type AddMemberToListParams struct {
	MemberId string `json:"member_id"`
	RoomId   string `json:"room_id"`
}

type GetMemberParams struct {
	MemberId string `json:"member_id"`
	RoomId   string `json:"room_id"`
}

type SetMemberParams struct {
	MemberId  string  `json:"member_id"`
	Username  string  `json:"username"`
	Color     string  `json:"color"`
	AvatarURL *string `json:"avatar_url"`
	IsMuted   bool    `json:"is_muted"`
	IsAdmin   bool    `json:"is_admin"`
	IsReady   bool    `json:"is_ready"`
	RoomId    string  `json:"room_id"`
}

type RemoveMemberParams struct {
	MemberId string `json:"member_id"`
	RoomId   string `json:"room_id"`
}

type RemoveMemberFromListParams struct {
	MemberId string `json:"member_id"`
	RoomId   string `json:"room_id"`
}
