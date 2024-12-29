package controller

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/pkg/ctxlogger"
	"github.com/sharetube/server/pkg/wsrouter"
)

func (c controller) wsRequestIdWSMw() wsrouter.Middleware {
	return func(next wsrouter.HandlerFunc[any]) wsrouter.HandlerFunc[any] {
		return func(ctx context.Context, conn *websocket.Conn, payload any) error {
			ctx = ctxlogger.AppendCtx(ctx, slog.String("ws_request_id", c.generateTimeBasedId()))
			return next(ctx, conn, payload)
		}
	}
}

func (c controller) loggerWSMw() wsrouter.Middleware {
	return func(next wsrouter.HandlerFunc[any]) wsrouter.HandlerFunc[any] {
		return func(ctx context.Context, conn *websocket.Conn, payload any) error {
			ctx = ctxlogger.AppendCtx(ctx, slog.String("message_type", wsrouter.GetMessageTypeFromCtx(ctx)))
			c.logger.InfoContext(ctx, "websocket message received", "payload", payload)

			start := time.Now()

			err := next(ctx, conn, payload)

			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			c.logger.InfoContext(ctx, "websocket message handled",
				"processing_time_us", time.Since(start).Microseconds(),
				"alloc", memStats.Alloc/1024,
				"total_alloc", memStats.TotalAlloc/1024,
				"sys", memStats.Sys/1024,
				"goroutines", runtime.NumGoroutine(),
			)

			return err
		}
	}
}
