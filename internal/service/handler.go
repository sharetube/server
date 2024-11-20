package service

import (
	"encoding/json"
	"log/slog"

	"github.com/sharetube/server/internal/domain"
)

func (r *Room) handleRemoveMember(input *Input) (*domain.Member, error) {
	// if input.Data == nil {
	// 	return nil, ErrEmptyData
	// }

	if !input.Sender.IsAdmin {
		return nil, ErrPermissionDenied
	}

	var memberID string
	if err := json.Unmarshal(input.Data, &memberID); err != nil {
		return nil, err
	}

	removedMember, err := r.members.RemoveByID(memberID)
	if err != nil {
		return nil, err
	}

	if r.members.Length() == 0 {
		r.Close()
		return nil, nil
	}

	removedMember.Conn.Close()

	return &removedMember, nil
}

func (r *Room) handlePromoteMember(input *Input) (*domain.Member, error) {
	// if input.Data == nil {
	// 	return nil, ErrEmptyData
	// }

	if !input.Sender.IsAdmin {
		return nil, ErrPermissionDenied
	}

	var memberID string
	if err := json.Unmarshal(input.Data, &memberID); err != nil {
		return nil, err
	}

	promotedMember, err := r.members.PromoteMemberByID(memberID)
	if err != nil {
		return nil, err
	}

	return &promotedMember, nil
}

func (r *Room) handleDemoteMember(input *Input) (*domain.Member, error) {
	// if input.Data == nil {
	// 	return nil, ErrEmptyData
	// }

	if !input.Sender.IsAdmin {
		return nil, ErrPermissionDenied
	}

	var memberID string
	if err := json.Unmarshal(input.Data, &memberID); err != nil {
		return nil, err
	}

	demotedMember, err := r.members.DemoteMemberByID(memberID)
	if err != nil {
		return nil, err
	}

	return &demotedMember, nil
}

func (r *Room) handleAddVideo(input *Input) (*domain.Video, error) {
	// if input.Data == nil {
	// 	return nil, ErrEmptyData
	// }

	if !input.Sender.IsAdmin {
		return nil, ErrPermissionDenied
	}

	var videoURL string
	if err := json.Unmarshal(input.Data, &videoURL); err != nil {
		return nil, err
	}

	video, err := r.playlist.Add(input.Sender.ID, videoURL)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (r *Room) handleRemoveVideo(input *Input) (*domain.Video, error) {
	if !input.Sender.IsAdmin {
		return nil, ErrPermissionDenied
	}

	// if input.Data == nil {
	// 	return nil, ErrEmptyData
	// }

	var videoID int
	if err := json.Unmarshal(input.Data, &videoID); err != nil {
		return nil, err
	}

	video, err := r.playlist.RemoveByID(videoID)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (r *Room) handlePlayerUpdated(input *Input) (*domain.Player, error) {
	// if input.Data == nil {
	// 	return nil, ErrEmptyData
	// }

	var udpatedPlayer struct {
		IsPlaying    bool    `json:"is_playing"`
		CurrentTime  float64 `json:"current_time"`
		PlaybackRate float64 `json:"playback_rate"`
	}
	if err := json.Unmarshal(input.Data, &udpatedPlayer); err != nil {
		return nil, err
	}
	slog.Debug("player updated", "player", udpatedPlayer)

	r.player.IsPlaying = udpatedPlayer.IsPlaying
	r.player.CurrentTime = udpatedPlayer.CurrentTime
	r.player.PlaybackRate = udpatedPlayer.PlaybackRate

	return r.player, nil
}
