package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (c controller) corsMiddleware(next http.Handler) http.Handler {
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
func (c controller) Mux() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(c.corsMiddleware)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/room", func(r chi.Router) {
			// r.Get("/{room-id}", c.GetRoom)
			r.Route("/create", func(r chi.Router) {
				r.Post("/validate", c.validateCreateRoom)
				r.Get("/ws", c.createRoom)
			})
			r.Route("/{room-id}", func(r chi.Router) {
				r.Route("/join", func(r chi.Router) {
					r.Post("/validate", c.validateJoinRoom)
					r.Get("/ws", c.joinRoom)
				})
			})
		})
	})

	return r
}
