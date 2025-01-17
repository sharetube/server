package service

import (
	"context"
	"errors"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gorilla/websocket"
)

type UpdatePlayerStateParams struct {
	VideoId      int             `json:"video_id"`
	IsPlaying    bool            `json:"is_playing"`
	CurrentTime  int             `json:"current_time"`
	PlaybackRate float64         `json:"playback_rate"`
	UpdatedAt    int             `json:"updated_at"`
	SenderId     string          `json:"sender_id"`
	SenderConn   *websocket.Conn `json:"sender_conn"`
	RoomId       string          `json:"room_id"`
}

type UpdatePlayerStateResponse struct {
	Player Player
	Conns  []*websocket.Conn
}

func (s service) UpdatePlayerState(ctx context.Context, params *UpdatePlayerStateParams) (UpdatePlayerStateResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return UpdatePlayerStateResponse{}, err
	}
	//? add validation

	currentVideoId, err := s.roomRepo.GetCurrentVideoId(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get current video id: %w", err)
	}

	if currentVideoId != nil && *currentVideoId != params.VideoId {
		return UpdatePlayerStateResponse{}, errors.New("video id is not equal")
	}

	updated := false
	player, err := s.roomRepo.GetPlayer(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get player: %w", err)
	}

	// todo: wrap in transaction
	if player.CurrentTime != params.CurrentTime {
		updated = true
		if err := s.roomRepo.UpdatePlayerCurrentTime(ctx, params.RoomId, params.CurrentTime); err != nil {
			return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player current time: %w", err)
		}
	}

	if player.IsPlaying != params.IsPlaying {
		updated = true
		if err := s.roomRepo.UpdatePlayerIsPlaying(ctx, params.RoomId, params.IsPlaying); err != nil {
			return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player is playing: %w", err)
		}
	}

	if player.PlaybackRate != params.PlaybackRate {
		updated = true
		if err := s.roomRepo.UpdatePlayerPlaybackRate(ctx, params.RoomId, params.PlaybackRate); err != nil {
			return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player playback rate: %w", err)
		}
	}

	// if player.IsEnded != params.IsEnded {
	// 	updated = true
	// 	if err := s.roomRepo.UpdatePlayerIsEnded(ctx, params.RoomId, params.IsEnded); err != nil {
	// 		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player is ended: %w", err)
	// 	}
	// }

	if !updated {
		return UpdatePlayerStateResponse{
			Player: Player{
				IsPlaying:    player.IsPlaying,
				CurrentTime:  player.CurrentTime,
				PlaybackRate: player.PlaybackRate,
				UpdatedAt:    player.UpdatedAt,
			},
			Conns: []*websocket.Conn{params.SenderConn},
		}, nil
	}

	if err := s.roomRepo.UpdatePlayerUpdatedAt(ctx, params.RoomId, params.UpdatedAt); err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player updated at: %w", err)
	}

	if player.WaitingForReady {
		if err := s.roomRepo.UpdatePlayerWaitingForReady(ctx, params.RoomId, false); err != nil {
			return UpdatePlayerStateResponse{}, fmt.Errorf("failed to update player waiting for ready: %w", err)
		}
	}

	memberIds, err := s.roomRepo.GetMemberIds(ctx, params.RoomId)
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get member ids: %w", err)
	}

	for i, memberId := range memberIds {
		if memberId == params.SenderId {
			memberIds = append(memberIds[:i], memberIds[i+1:]...)
			break
		}
	}

	conns, err := s.getConnsFromMemberIds(ctx, memberIds)
	if err != nil {
		return UpdatePlayerStateResponse{}, fmt.Errorf("failed to get conns from member ids: %w", err)
	}

	return UpdatePlayerStateResponse{
		Player: Player{
			IsPlaying:    params.IsPlaying,
			CurrentTime:  params.CurrentTime,
			PlaybackRate: params.PlaybackRate,
			UpdatedAt:    params.UpdatedAt,
		},
		Conns: conns,
	}, nil
}

type UpdatePlayerVideoParams struct {
	VideoId   int    `json:"video_id"`
	UpdatedAt int    `json:"updated_at"`
	SenderId  string `json:"sender_id"`
	RoomId    string `json:"room_id"`
}

type UpdatePlayerVideoResponse struct {
	Player   Player
	Members  []Member
	Conns    []*websocket.Conn
	Playlist Playlist
}

func (s service) UpdatePlayerVideo(ctx context.Context, params *UpdatePlayerVideoParams) (UpdatePlayerVideoResponse, error) {
	if err := s.checkIfMemberAdmin(ctx, params.RoomId, params.SenderId); err != nil {
		return UpdatePlayerVideoResponse{}, err
	}

	if err := validation.ValidateStructWithContext(ctx, params,
		validation.Field(&params.VideoId, VideoIdRule...),
	); err != nil {
		return UpdatePlayerVideoResponse{}, err
	}

	updatePlayerVideoRes, err := s.updatePlayerVideo(ctx, params.RoomId, params.VideoId, params.UpdatedAt)
	if err != nil {
		return UpdatePlayerVideoResponse{}, err
	}

	return UpdatePlayerVideoResponse{
		Members:  updatePlayerVideoRes.Members,
		Player:   updatePlayerVideoRes.Player,
		Playlist: updatePlayerVideoRes.Playlist,
		Conns:    updatePlayerVideoRes.Conns,
	}, nil
}
