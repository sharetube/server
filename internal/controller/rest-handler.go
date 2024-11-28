package controller

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
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

	connectToken, err := c.roomService.CreateRoomCreateSession(r.Context(), &room.CreateRoomCreateSessionParams{
		Username:        req.Username,
		Color:           req.Color,
		AvatarURL:       req.AvatarURL,
		InitialVideoURL: req.InitialVideoURL,
	})
	if err != nil {
		slog.Info("ValidateCreateRoom", "create room session err", err)
		rest.WriteJSON(w, http.StatusInternalServerError, rest.Envelope{"error": err.Error()})
		return
	}

	cookie := &http.Cookie{
		Name:     "st-connect-token",
		Value:    connectToken,
		Path:     "/",
		Domain:   "127.0.0.1",
		Secure:   false,
		HttpOnly: false,
	}
	http.SetCookie(w, cookie)

	rest.WriteJSON(w, http.StatusOK, rest.Envelope{"data": validateCreateRoomResponse{
		ConnectToken: connectToken,
	}})
}

type validateJoinRoom struct {
	Username  string `json:"username" validate:"required,max=16"`
	Color     string `json:"color" validate:"required,min=3,max=6"`
	AvatarURL string `json:"avatar_url" validate:"required"`
}

type validateJoinRoomResponse struct {
	ConnectToken string `json:"connect_token"`
}

func (c Controller) ValidateJoinRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "room-id")
	if roomID == "" {
		slog.Info("ValidateJoinRoom", "room_id is empty", "")
		rest.WriteJSON(w, http.StatusNotFound, rest.Envelope{"error": "room not found"})
		return
	}

	var req validateJoinRoom

	if err := rest.ReadJSON(r, &req); err != nil {
		slog.Info("ValidateJoinRoom", "read json err", err)
		rest.WriteJSON(w, http.StatusUnprocessableEntity, rest.Envelope{"error": err.Error()})
		return
	}

	if validationErrors, ok := c.validate.Validate(req); !ok {
		slog.Info("ValidateJoinRoom", "validate err", validationErrors)
		rest.WriteJSON(w, http.StatusBadRequest, rest.Envelope{"errors": validationErrors})
		return
	}

	connectToken, err := c.roomService.CreateRoomJoinSession(r.Context(), &room.CreateRoomJoinSessionParams{
		Username:  req.Username,
		Color:     req.Color,
		AvatarURL: req.AvatarURL,
		RoomID:    roomID,
	})
	if err != nil {
		slog.Info("ValildateJoinRoom", "err", err)
		rest.WriteJSON(w, http.StatusInternalServerError, rest.Envelope{"error": err.Error()})
		return
	}
	cookie := &http.Cookie{
		Name:     "st-connect-token",
		Value:    connectToken,
		Path:     "/",
		Domain:   "127.0.0.1",
		Secure:   false,
		HttpOnly: false,
	}
	http.SetCookie(w, cookie)

	rest.WriteJSON(w, http.StatusOK, rest.Envelope{"data": validateJoinRoomResponse{
		ConnectToken: connectToken,
	}})
}
