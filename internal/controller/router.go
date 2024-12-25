package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (c controller) GetMux() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(c.requestIdMw)
	r.Use(c.requestLoggingMw)
	r.Use(cors.AllowAll().Handler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		r.Route("/ws", func(r chi.Router) {
			r.Route("/room", func(r chi.Router) {
				r.Get("/create", c.createRoom)
				r.Route("/{room-id}", func(r chi.Router) {
					r.Get("/join", c.joinRoom)
				})
			})
		})
	})

	return r
}
