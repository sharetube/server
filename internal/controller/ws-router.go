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
			ctx = ctxlogger.AppendCtx(ctx, slog.String("ws_message_type", wsrouter.GetMessageTypeFromCtx(ctx)))
			c.logger.InfoContext(ctx, "websocket message received", "payload", payload)

			start := time.Now()

			err := next(ctx, conn, payload)

			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			c.logger.InfoContext(ctx, "websocket message handled",
				"duration", time.Since(start).Microseconds(),
				"alloc", memStats.Alloc/1024,
				"total_alloc", memStats.TotalAlloc/1024,
				"sys", memStats.Sys/1024,
				"goroutines", runtime.NumGoroutine(),
			)

			return err
		}
	}
}

func (c controller) handleError(ctx context.Context, conn *websocket.Conn, err error) error {
	c.logger.InfoContext(ctx, "websocket handler error", "error", err)
	return c.writeError(ctx, conn, err)
}

func (c controller) getWSRouter() *wsrouter.WSRouter {
	mux := wsrouter.New()

	mux.SetErrorHandler(c.handleError)

	mux.Use(c.wsRequestIdWSMw())
	mux.Use(c.loggerWSMw())

	// video
	wsrouter.Handle(mux, "ALIVE", c.handleAlive)
	wsrouter.Handle(mux, "ADD_VIDEO", c.handleAddVideo)
	wsrouter.Handle(mux, "REMOVE_VIDEO", c.handleRemoveVideo)
	// wsrouter.Handle(mux, "REORDER_PLAYLIST", c.handleRemoveVideo)

	// member
	wsrouter.Handle(mux, "PROMOTE_MEMBER", c.handlePromoteMember)
	wsrouter.Handle(mux, "REMOVE_MEMBER", c.handleRemoveMember)

	// player
	wsrouter.Handle(mux, "UPDATE_PLAYER_STATE", c.handleUpdatePlayerState)
	wsrouter.Handle(mux, "UPDATE_PLAYER_VIDEO", c.handleUpdatePlayerVideo)

	// profile
	wsrouter.Handle(mux, "UPDATE_PROFILE", c.handleUpdateProfile)
	// mux.Handle("UPDATE_MUTED", c.handleUpdateMuted)
	wsrouter.Handle(mux, "UPDATE_READY", c.handleUpdateIsReady)

	return mux
}
