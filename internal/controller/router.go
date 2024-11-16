package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (c Controller) Mux() http.Handler {
	r := chi.NewRouter()

	r.HandleFunc("/ws/create-room", c.CreateRoom)
	r.HandleFunc("/ws/join-room/{room-id}", c.JoinRoom)

	return r
}
