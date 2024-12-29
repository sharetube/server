package room

import "time"

type Member struct {
	Username  string
	Color     string
	AvatarUrl *string
	IsMuted   bool
	IsAdmin   bool
	IsReady   bool
}

type AddMemberToListParams struct {
	MemberId string
	RoomId   string
}

type GetMemberParams struct {
	MemberId string
	RoomId   string
}

type SetMemberParams struct {
	MemberId  string
	Username  string
	Color     string
	AvatarUrl *string
	IsMuted   bool
	IsAdmin   bool
	IsReady   bool
	RoomId    string
}

type RemoveMemberParams struct {
	MemberId string
	RoomId   string
}

type ExpireMemberParams struct {
	MemberId string
	RoomId   string
	ExpireAt time.Time
}

type RemoveMemberFromListParams struct {
	MemberId string
	RoomId   string
}
