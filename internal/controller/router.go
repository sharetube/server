package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (c controller) Mux() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.AllowAll().Handler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/room", func(r chi.Router) {
			// r.Get("/{room-id}", c.GetRoom)
			r.Route("/create", func(r chi.Router) {
				// r.Post("/validate", c.validateCreateRoom)
				r.Get("/ws", c.createRoom)
			})
			r.Route("/{room-id}", func(r chi.Router) {
				r.Route("/join", func(r chi.Router) {
					// r.Post("/validate", c.validateJoinRoom)
					r.Get("/ws", c.joinRoom)
				})
			})
		})
	})

	return r
}
