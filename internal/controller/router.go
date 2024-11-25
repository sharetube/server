package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (c Controller) Mux() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/room", func(r chi.Router) {
			// r.Get("/{room-id}", c.GetRoom)
			r.Route("/create", func(r chi.Router) {
				r.Post("/validate", c.ValidateCreateRoom)
				// r.Get("/ws", c.CreateRoom)
			})
			r.Route("/{room-id}", func(r chi.Router) {
				r.Route("/join", func(r chi.Router) {
					r.Post("/validate", c.ValidateJoinRoom)
					// r.Get("/ws", c.JoinRoom)
				})
			})
		})
	})

	return r
}
