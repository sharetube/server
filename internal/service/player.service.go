package service

import (
	"context"
	"errors"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/repository/room"
)

type UpdatePlayerStateParams struct {
	VideoId       int             `json:"video_id"`
	IsPlaying     bool            `json:"is_playing"`
	CurrentTime   int             `json:"current_time"`
	PlaybackRate  float64         `json:"playback_rate"`
	UpdatedAt     int             `json:"updated_at"`
	PlayerVersion int             `json:"player_version"`
	SenderId      string          `json:"sender_id"`
	SenderConn    *websocket.Conn `json:"sender_conn"`
	RoomId        string          `json:"room_id"`
}

type UpdatePlayerStateResponse struct {
	Player Player
	Conns  []*websocket.Conn
}

func (s service) UpdatePlayerState(ctx context.Context, params *UpdatePlayerStateParams) (*UpdatePlayerStateResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return nil, err
	}
	//? add validation

	playerVersion, err := s.roomRepo.GetPlayerVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player version: %w", err)
	}

	if playerVersion != params.PlayerVersion {
		return nil, ErrPlayerVersionMismatch
	}

	currentVideoId, err := s.roomRepo.GetCurrentVideoId(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get current video id: %w", err)
	}

	if currentVideoId != params.VideoId {
		return nil, errors.New("video id is not equal")
	}

	updated := false
	player, err := s.roomRepo.GetPlayer(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	// todo: wrap in transaction
	if player.CurrentTime != params.CurrentTime {
		updated = true
		if err := s.roomRepo.UpdatePlayerCurrentTime(ctx, params.RoomId, params.CurrentTime); err != nil {
			return nil, fmt.Errorf("failed to update player current time: %w", err)
		}
	}

	if player.IsPlaying != params.IsPlaying {
		updated = true
		if err := s.roomRepo.UpdatePlayerIsPlaying(ctx, params.RoomId, params.IsPlaying); err != nil {
			return nil, fmt.Errorf("failed to update player is playing: %w", err)
		}
	}

	if player.PlaybackRate != params.PlaybackRate {
		updated = true
		if err := s.roomRepo.UpdatePlayerPlaybackRate(ctx, params.RoomId, params.PlaybackRate); err != nil {
			return nil, fmt.Errorf("failed to update player playback rate: %w", err)
		}
	}

	videoEnded, err := s.roomRepo.GetVideoEnded(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get video ended: %w", err)
	}

	if videoEnded {
		if err := s.roomRepo.SetVideoEnded(ctx, &room.SetVideoEndedParams{
			RoomId:     params.RoomId,
			VideoEnded: videoEnded,
		}); err != nil {
			return nil, fmt.Errorf("failed to update video ended: %w", err)
		}
	}

	if !updated {
		return &UpdatePlayerStateResponse{
			Player: Player{
				State: PlayerState{
					IsPlaying:    player.IsPlaying,
					CurrentTime:  player.CurrentTime,
					PlaybackRate: player.PlaybackRate,
					UpdatedAt:    player.UpdatedAt,
				},
				IsEnded: false,
				Version: playerVersion,
			},
			Conns: []*websocket.Conn{params.SenderConn},
		}, nil
	}

	if err := s.roomRepo.UpdatePlayerUpdatedAt(ctx, params.RoomId, params.UpdatedAt); err != nil {
		return nil, fmt.Errorf("failed to update player updated at: %w", err)
	}

	if player.WaitingForReady {
		if err := s.roomRepo.UpdatePlayerWaitingForReady(ctx, params.RoomId, false); err != nil {
			return nil, fmt.Errorf("failed to update player waiting for ready: %w", err)
		}
	}

	memberIds, err := s.roomRepo.GetMemberIds(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get member ids: %w", err)
	}

	for i, memberId := range memberIds {
		if memberId == params.SenderId {
			memberIds = append(memberIds[:i], memberIds[i+1:]...)
			break
		}
	}

	conns, err := s.getConnsFromMemberIds(ctx, memberIds)
	if err != nil {
		return nil, fmt.Errorf("failed to get conns from member ids: %w", err)
	}

	playerVersion, err = s.roomRepo.IncrPlayerVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to incr player version: %w", err)
	}

	return &UpdatePlayerStateResponse{
		Player: Player{
			State: PlayerState{
				IsPlaying:    params.IsPlaying,
				CurrentTime:  params.CurrentTime,
				PlaybackRate: params.PlaybackRate,
				UpdatedAt:    params.UpdatedAt,
			},
			IsEnded: false,
			Version: playerVersion,
		},
		Conns: conns,
	}, nil
}

type UpdatePlayerVideoParams struct {
	VideoId         int    `json:"video_id"`
	UpdatedAt       int    `json:"updated_at"`
	SenderId        string `json:"sender_id"`
	RoomId          string `json:"room_id"`
	PlayerVersion   int    `json:"player_version"`
	PlaylistVersion int    `json:"playlist_version"`
}

type UpdatePlayerVideoResponse struct {
	Player   Player
	Members  []Member
	Conns    []*websocket.Conn
	Playlist Playlist
}

func (s service) UpdatePlayerVideo(ctx context.Context, params *UpdatePlayerVideoParams) (*UpdatePlayerVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return nil, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoId, VideoIdRule...),
	); err != nil {
		return nil, err
	}

	playerVersion, err := s.roomRepo.GetPlayerVersion(ctx, params.RoomId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player version: %w", err)
	}

	if playerVersion != params.PlayerVersion {
		return nil, errors.New("player version is not equal")
	}

	updatePlayerVideoRes, err := s.updatePlayerVideo(ctx, params.RoomId, params.VideoId, params.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &UpdatePlayerVideoResponse{
		Members:  updatePlayerVideoRes.Members,
		Player:   updatePlayerVideoRes.Player,
		Playlist: updatePlayerVideoRes.Playlist,
		Conns:    updatePlayerVideoRes.Conns,
	}, nil
}
