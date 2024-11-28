package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (c Controller) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
func (c Controller) Mux() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(c.CORSMiddleware)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/room", func(r chi.Router) {
			// r.Get("/{room-id}", c.GetRoom)
			r.Route("/create", func(r chi.Router) {
				r.Post("/validate", c.ValidateCreateRoom)
				r.Get("/ws", c.CreateRoom)
			})
			r.Route("/{room-id}", func(r chi.Router) {
				r.Route("/join", func(r chi.Router) {
					r.Post("/validate", c.ValidateJoinRoom)
					r.Get("/ws", c.JoinRoom)
				})
			})
		})
	})

	return r
}
