package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (c Controller) Mux() http.Handler {
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/room", func(r chi.Router) {
			// r.Get("/{room-id}", c.GetRoom)
			r.Route("/create", func(r chi.Router) {
				r.Get("/validate", c.CreateRoom)
				r.Get("/ws", c.CreateRoom)
			})
			r.Post("/join", c.JoinRoom)
		})
	})

	return r
}
