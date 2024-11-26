package controller

import (
	"log/slog"
	"net/http"

	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/rest"
)

type validateCreateRoom struct {
	Username string `json:"username" validate:"required,max=16"`
	// todo: validate color
	Color           string `json:"color" validate:"required,min=3,max=6"`
	AvatarURL       string `json:"avatar_url" validate:"required"`
	InitialVideoURL string `json:"initial_video_url" validate:"required,len=11"`
}

type validateCreateRoomResponse struct {
	RoomID       string `json:"room_id"`
	MemberID     string `json:"member_id"`
	ConnectToken string `json:"connect_token"`
}

func (c Controller) ValidateCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req validateCreateRoom

	if err := rest.ReadJSON(r, &req); err != nil {
		slog.Info("ValidateCreateRoom", "read json err", err)
		rest.WriteJSON(w, http.StatusUnprocessableEntity, rest.Envelope{"error": err.Error()})
		return
	}

	if validationErrors, ok := c.validate.Validate(req); !ok {
		slog.Info("ValidateCreateRoom", "validate err", validationErrors)
		rest.WriteJSON(w, http.StatusBadRequest, rest.Envelope{"errors": validationErrors})
		return
	}

	resp, err := c.roomService.CreateRoom(r.Context(), &room.CreateRoomParams{
		Username:        req.Username,
		Color:           req.Color,
		AvatarURL:       req.AvatarURL,
		InitialVideoURL: req.InitialVideoURL,
	})
	if err != nil {
		slog.Info("ValidateCreateRoom", "create room err", err)
		rest.WriteJSON(w, http.StatusInternalServerError, rest.Envelope{"error": err.Error()})
		return
	}
	cookie := &http.Cookie{
		Name:     "st_connect_token",
		Value:    resp.ConnectToken,
		Path:     "/",
		Domain:   "127.0.0.1",
		Secure:   false,
		HttpOnly: false,
	}
	http.SetCookie(w, cookie)

	rest.WriteJSON(w, http.StatusOK, rest.Envelope{"data": validateCreateRoomResponse{
		RoomID:       resp.RoomID,
		MemberID:     resp.MemberID,
		ConnectToken: resp.ConnectToken,
	}})
}

func (c Controller) ValidateJoinRoom(w http.ResponseWriter, r *http.Request) {
	rest.WriteJSON(w, http.StatusOK, rest.Envelope{"message": "ok"})
}