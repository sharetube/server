package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/pkg/ctxlogger"
	"github.com/sharetube/server/pkg/wsrouter"
)

func (c controller) wsRequestIdWSMw(next wsrouter.HandlerFunc) wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
		ctx = ctxlogger.AppendCtx(ctx, slog.String("ws_request_id", c.generateTimeBasedId()))
		next(ctx, conn, payload)
	}
}

func (c controller) loggerWSMw(next wsrouter.HandlerFunc) wsrouter.HandlerFunc {
	return func(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
		c.logger.InfoContext(ctx, "request", "type", wsrouter.GetMessageTypeFromCtx(ctx), "payload", payload)
		start := time.Now()

		next(ctx, conn, payload)

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		c.logger.InfoContext(ctx, "returned",
			"duration", time.Since(start).Microseconds(),
			"alloc", memStats.Alloc/1024,
			"total_alloc", memStats.TotalAlloc/1024,
			"sys", memStats.Sys/1024,
			"goroutines", runtime.NumGoroutine(),
		)
	}
}

func (c controller) handleWSNotFound(ctx context.Context, conn *websocket.Conn, typ string) {
	c.logger.InfoContext(ctx, "handler not found", "type", typ)
	c.writeError(ctx, conn, fmt.Errorf("handler for type %s not found", typ))
}

func (c controller) getWSRouter() *wsrouter.WSRouter {
	mux := wsrouter.New(c.logger)

	mux.SetHandlerNotFound(c.handleWSNotFound)

	mux.Use(c.wsRequestIdWSMw)
	mux.Use(c.loggerWSMw)

	// video
	mux.Handle("ALIVE", c.handleAlive)
	mux.Handle("ADD_VIDEO", c.handleAddVideo)
	mux.Handle("REMOVE_VIDEO", c.handleRemoveVideo)
	// mux.Handle("REORDER_PLAYLIST", c.handleRemoveVideo)

	// member
	mux.Handle("PROMOTE_MEMBER", c.handlePromoteMember)
	mux.Handle("REMOVE_MEMBER", c.handleRemoveMember)

	// player
	mux.Handle("UPDATE_PLAYER_STATE", c.handleUpdatePlayerState)
	mux.Handle("UPDATE_PLAYER_VIDEO", c.handleUpdatePlayerVideo)

	// profile
	mux.Handle("UPDATE_PROFILE", c.handleUpdateProfile)
	// mux.Handle("UPDATE_MUTED", c.handleUpdateMuted)
	mux.Handle("UPDATE_READY", c.handleUpdateIsReady)

	return mux
}
