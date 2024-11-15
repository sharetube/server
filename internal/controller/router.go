package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h Handler) Mux() http.Handler {
	r := chi.NewRouter()

	r.HandleFunc("/ws/create-room", h.CreateRoom)
	r.HandleFunc("/ws/join-room/{room-id}", h.JoinRoom)

	return r
}
