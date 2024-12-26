package room

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

func (s service) getConnsFromMemberIds(ctx context.Context, memberIds []string) ([]*websocket.Conn, error) {
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

func (s service) getConnsByRoomId(ctx context.Context, roomId string) ([]*websocket.Conn, error) {
	memberIds, err := s.roomRepo.GetMemberIds(ctx, roomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get member ids: %w", err)
	}

	return s.getConnsFromMemberIds(ctx, memberIds)
}

type CreateRoomParams struct {
	Username        string
	Color           string
	AvatarURL       *string
	InitialVideoURL string
}

type CreateRoomResponse struct {
	RoomId       string
	JoinedMember Member
	JWT          string
}

func (s service) CreateRoom(ctx context.Context, params *CreateRoomParams) (*CreateRoomResponse, error) {
	roomId := s.generator.GenerateRandomString(8)

	memberId := uuid.NewString()
	setMemberParams := room.SetMemberParams{
		MemberId:  memberId,
		Username:  params.Username,
		Color:     params.Color,
		AvatarURL: params.AvatarURL,
		IsMuted:   false,
		IsAdmin:   true,
		IsReady:   false,
		RoomId:    roomId,
	}
	if err := s.roomRepo.SetMember(ctx, &setMemberParams); err != nil {
		return nil, fmt.Errorf("failed to set member: %w", err)
	}

	jwt, err := s.generateJWT(memberId)
	if err != nil {
		return nil, fmt.Errorf("failed to generate jwt: %w", err)
	}

	videoId := uuid.NewString()
	if err := s.roomRepo.SetVideo(ctx, &room.SetVideoParams{
		VideoId: videoId,
		RoomId:  roomId,
		URL:     params.InitialVideoURL,
	}); err != nil {
		return nil, fmt.Errorf("failed to set video: %w", err)
	}

	if err := s.roomRepo.SetPlayer(ctx, &room.SetPlayerParams{
		VideoId:      videoId,
		IsPlaying:    s.getDefaultPlayerIsPlaying(),
		CurrentTime:  s.getDefaultPlayerCurrentTime(),
		PlaybackRate: s.getDefaultPlayerPlaybackRate(),
		UpdatedAt:    int(time.Now().Unix()),
		RoomId:       roomId,
	}); err != nil {
		return nil, fmt.Errorf("failed to set player: %w", err)
	}

	return &CreateRoomResponse{
		JWT:    jwt,
		RoomId: roomId,
		JoinedMember: Member{
			Id:        memberId,
			Username:  setMemberParams.Username,
			Color:     setMemberParams.Color,
			AvatarURL: setMemberParams.AvatarURL,
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
		AvatarURL: member.AvatarURL,
		IsMuted:   member.IsMuted,
		IsAdmin:   member.IsAdmin,
		IsReady:   member.IsReady,
	}, nil
}

type JoinRoomParams struct {
	JWT       string
	Username  string
	Color     string
	AvatarURL *string
	RoomId    string
}

type JoinRoomResponse struct {
	JWT          string
	JoinedMember Member
	Members      []Member
	Conns        []*websocket.Conn
}

func (s service) JoinRoom(ctx context.Context, params *JoinRoomParams) (JoinRoomResponse, error) {
	conns, err := s.getConnsByRoomId(ctx, params.RoomId)
	if err != nil {
		return JoinRoomResponse{}, err
	}

	jwt := params.JWT
	member, err := s.getMemberByJWT(ctx, params.RoomId, params.JWT)
	if err != nil {
		return JoinRoomResponse{}, err
	}

	if member == nil {
		// check if room exists
		_, err := s.roomRepo.GetPlayerVideoId(ctx, params.RoomId)
		if err != nil {
			return JoinRoomResponse{}, errors.New("room not found")
		}

		// member not found, creating new one
		memberId := uuid.NewString()
		setMemberParams := room.SetMemberParams{
			MemberId:  memberId,
			Username:  params.Username,
			Color:     params.Color,
			AvatarURL: params.AvatarURL,
			IsMuted:   false,
			IsAdmin:   false,
			IsReady:   false,
			RoomId:    params.RoomId,
		}
		if err := s.roomRepo.SetMember(ctx, &setMemberParams); err != nil {
			return JoinRoomResponse{}, fmt.Errorf("failed to set member: %w", err)
		}

		member = &Member{
			Id:        memberId,
			Username:  params.Username,
			Color:     params.Color,
			AvatarURL: params.AvatarURL,
			IsMuted:   false,
			IsAdmin:   false,
			IsReady:   false,
		}

		jwt, err = s.generateJWT(member.Id)
		if err != nil {
			return JoinRoomResponse{}, fmt.Errorf("failed to generate jwt: %w", err)
		}
	} else {
		// member found, updating
		if err := s.roomRepo.AddMemberToList(ctx, &room.AddMemberToListParams{
			RoomId:   params.RoomId,
			MemberId: member.Id,
		}); err != nil {
			return JoinRoomResponse{}, fmt.Errorf("failed to add member to list: %w", err)
		}

		if member.Username != params.Username {
			if err := s.roomRepo.UpdateMemberUsername(ctx, params.RoomId, member.Id, params.Username); err != nil {
				return JoinRoomResponse{}, fmt.Errorf("failed to update member username: %w", err)
			}
			member.Username = params.Username
		}

		if member.Color != params.Color {
			if err := s.roomRepo.UpdateMemberColor(ctx, params.RoomId, member.Id, params.Color); err != nil {
				return JoinRoomResponse{}, fmt.Errorf("failed to update member color: %w", err)
			}
			member.Color = params.Color
		}

		if member.AvatarURL != params.AvatarURL {
			if err := s.roomRepo.UpdateMemberAvatarURL(ctx, params.RoomId, member.Id, params.AvatarURL); err != nil {
				return JoinRoomResponse{}, fmt.Errorf("failed to update member avatar URL: %w", err)
			}
			member.AvatarURL = params.AvatarURL
		}
	}

	members, err := s.getMembers(ctx, params.RoomId)
	if err != nil {
		return JoinRoomResponse{}, err
	}
	return JoinRoomResponse{
		JWT:          jwt,
		Conns:        conns,
		Members:      members,
		JoinedMember: *member,
	}, nil
}

func (s service) GetRoom(ctx context.Context, roomId string) (Room, error) {
	player, err := s.roomRepo.GetPlayer(ctx, roomId)
	if err != nil {
		return Room{}, fmt.Errorf("failed to get player: %w", err)
	}

	currentVideo, err := s.roomRepo.GetVideo(ctx, &room.GetVideoParams{
		VideoId: player.VideoId,
		RoomId:  roomId,
	})
	if err != nil {
		return Room{}, fmt.Errorf("failed to get current video: %w", err)
	}

	members, err := s.getMembers(ctx, roomId)
	if err != nil {
		return Room{}, err
	}

	playlist, err := s.getPlaylist(ctx, roomId)
	if err != nil {
		return Room{}, err
	}

	return Room{
		RoomId: roomId,
		Player: Player{
			VideoURL:     currentVideo.URL,
			IsPlaying:    player.IsPlaying,
			CurrentTime:  player.CurrentTime,
			PlaybackRate: player.PlaybackRate,
			UpdatedAt:    player.UpdatedAt,
		},
		Members:  members,
		Playlist: playlist,
	}, nil
}
