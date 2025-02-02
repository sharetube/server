package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
	"github.com/sharetube/server/pkg/ytvideodata"
)

func (s service) getConnsFromMemberIds(_ context.Context, memberIds []string) ([]*websocket.Conn, error) {
	conns := make([]*websocket.Conn, 0, len(memberIds))
	for _, memberId := range memberIds {
		conn, err := s.connRepo.GetConn(memberId)
		if err != nil {
			return nil, fmt.Errorf("failed to get conn: %w", err)
		}
		conns = append(conns, conn)
	}

	return conns, nil
}

func (s service) getConns(ctx context.Context, roomId string) ([]*websocket.Conn, error) {
	memberIds, err := s.roomRepo.GetMemberIds(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get member ids: %w", err)
	}

	return s.getConnsFromMemberIds(ctx, memberIds)
}

type CreateRoomParams struct {
	Username        string  `json:"username"`
	Color           string  `json:"color"`
	AvatarUrl       *string `json:"avatar_url"`
	InitialVideoUrl string  `json:"initial_video_url"`
}

type CreateRoomResponse struct {
	RoomId       string
	JoinedMember Member
	JWT          string
}

func (s service) CreateRoom(ctx context.Context, params *CreateRoomParams) (*CreateRoomResponse, error) {
	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.Username, UsernameRule...),
		validation.Field(&params.Color, ColorRule...),
		validation.Field(&params.AvatarUrl, AvatarUrlRule...),
		validation.Field(&params.InitialVideoUrl, VideoUrlRule...),
	); err != nil {
		return nil, err
	}

	videoData, err := ytvideodata.Get(params.InitialVideoUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get video data: %w", err)
	}

	roomId := s.generator.GenerateRandomString(8)

	memberId := uuid.NewString()
	setMemberParams := room.SetMemberParams{
		MemberId:  memberId,
		Username:  params.Username,
		Color:     params.Color,
		AvatarUrl: params.AvatarUrl,
		IsMuted:   s.getDefaultMemberIsMuted(),
		IsAdmin:   true,
		IsReady:   s.getDefaultMemberIsReady(),
		RoomId:    roomId,
	}
	if err := s.roomRepo.SetMember(ctx, &setMemberParams); err != nil {
		return nil, fmt.Errorf("failed to set member: %w", err)
	}

	if err := s.roomRepo.AddMemberToList(ctx, &room.AddMemberToListParams{
		MemberId: memberId,
		RoomId:   roomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to add member to list: %w", err)
	}

	jwt, err := s.generateJWT(memberId)
	if err != nil {
		return nil, fmt.Errorf("failed to generate jwt: %w", err)
	}

	videoId, err := s.roomRepo.SetVideo(ctx, &room.SetVideoParams{
		RoomId:       roomId,
		Url:          params.InitialVideoUrl,
		Title:        videoData.Title,
		ThumbnailUrl: videoData.ThumbnailUrl,
		AuthorName:   videoData.AuthorName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set video: %w", err)
	}

	if err := s.roomRepo.SetCurrentVideoId(ctx, &room.SetCurrentVideoParams{
		VideoId: videoId,
		RoomId:  roomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to set current video id: %w", err)
	}

	if err := s.roomRepo.SetPlayer(ctx, &room.SetPlayerParams{
		IsPlaying:       s.getDefaultPlayerIsPlaying(),
		WaitingForReady: s.getDefaultPlayerWaitingForReady(),
		CurrentTime:     s.getDefaultPlayerCurrentTime(),
		PlaybackRate:    s.getDefaultPlayerPlaybackRate(),
		UpdatedAt:       int(time.Now().UnixMicro()),
		RoomId:          roomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to set player: %w", err)
	}

	if err := s.roomRepo.SetVideoEnded(ctx, &room.SetVideoEndedParams{
		VideoEnded: false,
		RoomId:     roomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to set video ended: %w", err)
	}

	return &CreateRoomResponse{
		JWT:    jwt,
		RoomId: roomId,
		JoinedMember: Member{
			Id:        memberId,
			Username:  setMemberParams.Username,
			Color:     setMemberParams.Color,
			AvatarUrl: setMemberParams.AvatarUrl,
			IsMuted:   setMemberParams.IsMuted,
			IsAdmin:   setMemberParams.IsAdmin,
			IsReady:   setMemberParams.IsReady,
		},
	}, nil
}

func (s service) getMemberByJWT(ctx context.Context, roomId, jwt string) (*Member, error) {
	if jwt == "" {
		return nil, nil
	}

	claims, err := s.parseJWT(jwt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jwt: %w", err)
	}

	// todo: add validation

	member, err := s.roomRepo.GetMember(ctx, &room.GetMemberParams{
		RoomId:   roomId,
		MemberId: claims.MemberId,
	})
	if err != nil {
		if errors.Is(err, room.ErrMemberNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	return &Member{
		Id:        claims.MemberId,
		Username:  member.Username,
		Color:     member.Color,
		AvatarUrl: member.AvatarUrl,
		IsMuted:   member.IsMuted,
		IsAdmin:   member.IsAdmin,
		IsReady:   member.IsReady,
	}, nil
}

type JoinRoomParams struct {
	JWT       string  `json:"jwt"`
	Username  string  `json:"username"`
	Color     string  `json:"color"`
	AvatarUrl *string `json:"avatar_url"`
	RoomId    string  `json:"room_id"`
}

type JoinRoomResponse struct {
	JWT          string
	JoinedMember Member
	Members      []Member
	Conns        []*websocket.Conn
}

func (s service) JoinRoom(ctx context.Context, params *JoinRoomParams) (*JoinRoomResponse, error) {
	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.Username, UsernameRule...),
		validation.Field(&params.Color, ColorRule...),
		validation.Field(&params.AvatarUrl, AvatarUrlRule...),
		validation.Field(&params.RoomId, RoomIdRule...),
	); err != nil {
		return nil, err
	}

	conns, err := s.getConns(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns: %w", err)
	}

	if len(conns) >= s.membersLimit {
		return nil, errors.New("room is full")
	}

	jwt := params.JWT
	member, err := s.getMemberByJWT(ctx, params.RoomId, params.JWT)
	if err != nil {
		return nil, fmt.Errorf("failed to get member by jwt: %w", err)
	}

	if member == nil {
		//? check if room exists
		_, err := s.roomRepo.GetCurrentVideoId(ctx, params.RoomId)
		if err != nil {
			return nil, errors.New("room not found")
		}

		// member not found, creating new one
		memberId := uuid.NewString()
		setMemberParams := room.SetMemberParams{
			MemberId:  memberId,
			Username:  params.Username,
			Color:     params.Color,
			AvatarUrl: params.AvatarUrl,
			IsMuted:   false,
			IsAdmin:   false,
			IsReady:   false,
			RoomId:    params.RoomId,
		}
		if err := s.roomRepo.SetMember(ctx, &setMemberParams); err != nil {
			return nil, fmt.Errorf("failed to set member: %w", err)
		}

		if err := s.roomRepo.AddMemberToList(ctx, &room.AddMemberToListParams{
			RoomId:   params.RoomId,
			MemberId: memberId,
		}); err != nil {
			return nil, fmt.Errorf("failed to add member to list: %w", err)
		}

		member = &Member{
			Id:        memberId,
			Username:  params.Username,
			Color:     params.Color,
			AvatarUrl: params.AvatarUrl,
			IsMuted:   false,
			IsAdmin:   false,
			IsReady:   false,
		}

		jwt, err = s.generateJWT(member.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to generate jwt: %w", err)
		}
	} else {
		// member found, updating
		if err := s.roomRepo.AddMemberToList(ctx, &room.AddMemberToListParams{
			RoomId:   params.RoomId,
			MemberId: member.Id,
		}); err != nil {
			return nil, fmt.Errorf("failed to add member to list: %w", err)
		}

		if member.Username != params.Username {
			if err := s.roomRepo.UpdateMemberUsername(ctx, params.RoomId, member.Id, params.Username); err != nil {
				return nil, fmt.Errorf("failed to update member username: %w", err)
			}
			member.Username = params.Username
		}

		if member.Color != params.Color {
			if err := s.roomRepo.UpdateMemberColor(ctx, params.RoomId, member.Id, params.Color); err != nil {
				return nil, fmt.Errorf("failed to update member color: %w", err)
			}
			member.Color = params.Color
		}

		if member.AvatarUrl != params.AvatarUrl {
			if err := s.roomRepo.UpdateMemberAvatarUrl(ctx, params.RoomId, member.Id, params.AvatarUrl); err != nil {
				return nil, fmt.Errorf("failed to update member avatar URL: %w", err)
			}
			member.AvatarUrl = params.AvatarUrl
		}
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	return &JoinRoomResponse{
		JWT:          jwt,
		Conns:        conns,
		Members:      members,
		JoinedMember: *member,
	}, nil
}

func (s service) GetRoom(ctx context.Context, roomId string) (*Room, error) {
	members, err := s.getMembers(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	playlist, err := s.getPlaylist(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	player, err := s.getPlayer(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	return &Room{
		Id:       roomId,
		Player:   *player,
		Members:  members,
		Playlist: *playlist,
	}, nil
}
