package controller

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sharetube/server/pkg/ctxlogger"
)

func (c controller) connectionIdMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = ctxlogger.AppendCtx(ctx, slog.String("connection_id", c.generateTimeBasedId()))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (c controller) GetMux() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(cors.AllowAll().Handler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/ws", func(r chi.Router) {
			r.Route("/room", func(r chi.Router) {
				r.With(c.connectionIdMiddleware).Get("/create", c.createRoom)
				r.With(c.connectionIdMiddleware).Get("/join", c.joinRoom)
			})
		})
	})

	return r
}
