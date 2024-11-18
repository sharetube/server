package service

import (
	"strconv"

	"github.com/sharetube/server/internal/domain"
)

func (r *Room) handleRemoveMember(input *Input) (*domain.Member, error) {
	if input.Data == nil {
		return nil, ErrEmptyData
	}

	member, _, err := r.members.GetByConn(input.Sender)
	if err != nil {
		return nil, err
	}

	if !member.IsAdmin {
		return nil, ErrPermissionDenied
	}

	removedMember, err := r.members.RemoveByID(*input.Data)
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
	if input.Data == nil {
		return nil, ErrEmptyData
	}

	member, _, err := r.members.GetByConn(input.Sender)
	if err != nil {
		return nil, err
	}

	if !member.IsAdmin {
		return nil, ErrPermissionDenied
	}

	promotedMember, err := r.members.PromoteMemberByID(*input.Data)
	if err != nil {
		return nil, err
	}

	return &promotedMember, nil
}

func (r *Room) handleDemoteMember(input *Input) (*domain.Member, error) {
	if input.Data == nil {
		return nil, ErrEmptyData
	}

	member, _, err := r.members.GetByConn(input.Sender)
	if err != nil {
		return nil, err
	}

	if !member.IsAdmin {
		return nil, ErrPermissionDenied
	}

	demotedMember, err := r.members.DemoteMemberByID(*input.Data)
	if err != nil {
		return nil, err
	}

	return &demotedMember, nil
}

func (r *Room) handleAddVideo(input *Input) (*domain.Video, error) {
	member, _, err := r.members.GetByConn(input.Sender)
	if err != nil {
		return nil, err
	}

	if input.Data == nil {
		return nil, ErrEmptyData
	}

	if !member.IsAdmin {
		return nil, ErrPermissionDenied
	}

	video, err := r.playlist.Add(member.ID, *input.Data)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (r *Room) handleRemoveVideo(input *Input) (*domain.Video, error) {
	member, _, err := r.members.GetByConn(input.Sender)
	if err != nil {
		return nil, err
	}

	if !member.IsAdmin {
		return nil, ErrPermissionDenied
	}

	if input.Data == nil {
		return nil, ErrEmptyData
	}

	videoID, err := strconv.Atoi(*input.Data)
	if err != nil {
		return nil, err
	}

	video, err := r.playlist.RemoveByID(videoID)
	if err != nil {
		return nil, err
	}

	return &video, nil
}
