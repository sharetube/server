package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (c Controller) Mux() http.Handler {
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.HandleFunc("/ws/create-room", c.CreateRoom)
		r.HandleFunc("/ws/join-room/{room-id}", c.JoinRoom)
	})

	return r
}
