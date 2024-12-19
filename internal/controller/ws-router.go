package controller

import (
	"github.com/sharetube/server/pkg/wsrouter"
)

func (c controller) getWSRouter() *wsrouter.WSRouter {
	mux := wsrouter.New()

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
