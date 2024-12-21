package controller

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sharetube/server/pkg/ctxlogger"
)

func (c controller) requestIdMw(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = ctxlogger.AppendCtx(ctx, slog.String("request_id", c.generateTimeBasedId()))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (c controller) requestLoggingMw(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.logger.InfoContext(r.Context(), "request",
			"method", r.Method,
			"url", r.URL.String(),
			"remote_addr", r.RemoteAddr,
			"headers", r.Header,
			"body", r.Body,
		)
		next.ServeHTTP(w, r)
	})
}

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
