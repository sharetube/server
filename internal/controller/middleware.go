package controller

import (
	"log/slog"
	"net/http"

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
