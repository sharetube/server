package controller

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/pkg/wsrouter"
)

func (c controller) handleWSError(ctx context.Context, conn *websocket.Conn, err error) error {
	c.logger.InfoContext(ctx, "websocket handler error", "error", err)
	return c.writeError(ctx, conn, err)
}

func (c controller) getWSRouter() *wsrouter.WSRouter {
	mux := wsrouter.New()

	mux.SetErrorHandler(c.handleWSError)

	mux.Use(c.wsRequestIdWSMw())
	mux.Use(c.loggerWSMw())

	// video
	wsrouter.Handle(mux, "ALIVE", c.handleAlive)
	wsrouter.Handle(mux, "ADD_VIDEO", c.handleAddVideo)
	wsrouter.Handle(mux, "REMOVE_VIDEO", c.handleRemoveVideo)
	wsrouter.Handle(mux, "REORDER_PLAYLIST", c.handleReorderPlaylist)

	// member
	wsrouter.Handle(mux, "PROMOTE_MEMBER", c.handlePromoteMember)
	wsrouter.Handle(mux, "REMOVE_MEMBER", c.handleRemoveMember)

	// player
	wsrouter.Handle(mux, "UPDATE_PLAYER_STATE", c.handleUpdatePlayerState)
	wsrouter.Handle(mux, "UPDATE_PLAYER_VIDEO", c.handleUpdatePlayerVideo)
	wsrouter.Handle(mux, "END_VIDEO", c.handleEndVideo)

	// profile
	wsrouter.Handle(mux, "UPDATE_PROFILE", c.handleUpdateProfile)
	wsrouter.Handle(mux, "UPDATE_MUTED", c.handleUpdateIsMuted)
	wsrouter.Handle(mux, "UPDATE_READY", c.handleUpdateIsReady)

	return mux
}
