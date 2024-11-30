package controller

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/gorilla/websocket"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/wsrouter"
)

type Output struct {
	Action string `json:"action"`
	Data   any    `json:"data"`
}

func (c controller) getRoomIDFromCtx(ctx context.Context) string {
	return ctx.Value(roomIDCtxKey).(string)
}

func (c controller) getMemberIDFromCtx(ctx context.Context) string {
	return ctx.Value(memberIDCtxKey).(string)
}

func (c controller) handleGetState(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	roomState, err := c.roomService.GetRoomState(ctx, roomID)
	if err != nil {
		slog.Warn("failed to get room state", "error", err)
		if err := c.writeError(conn, err); err != nil {
			slog.Warn("failed to write error", "error", err)
			return
		}
	}

	if err := conn.WriteJSON(&Output{
		Action: "room_state",
		Data:   roomState,
	}); err != nil {
		slog.Warn("failed to write message", "error", err)
		return
	}
}

func (c controller) handleAddVideo(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) {
	roomID := c.getRoomIDFromCtx(ctx)
	memberID := c.getMemberIDFromCtx(ctx)

	var data struct {
		VideoURL string `json:"video_url"`
	}
	if err := c.unmarshalJSONorError(conn, payload, &data); err != nil {
		return
	}

	addVideoResponse, err := c.roomService.AddVideo(ctx, &room.AddVideoParams{
		MemberID: memberID,
		RoomID:   roomID,
		VideoURL: data.VideoURL,
	})
	if err != nil {
		slog.Warn("failed to add video", "error", err)
		if err := c.writeError(conn, err); err != nil {
			slog.Warn("failed to write error", "error", err)
			return
		}
	}

	if err := c.broadcast(addVideoResponse.Conns, &Output{
		Action: "video_added",
		Data: map[string]any{
			"added_video": addVideoResponse.AddedVideo,
			"playlist":    addVideoResponse.Playlist,
		},
	}); err != nil {
		slog.Warn("failed to broadcast", "error", err)
		return
	}
}
func (c controller) initWSMux() *wsrouter.WSRouter {
	mux := wsrouter.NewWSRouter()
	mux.Handle("GET_STATE", c.handleGetState)
	mux.Handle("ADD_VIDEO", c.handleAddVideo)

	return mux
}

// func (c controller) readMessages(ctx context.Context, conn *websocket.Conn, memberID, roomID string) {
// 	for {
// 		var input Input
// 		if err := conn.ReadJSON(&input); err != nil {
// 			slog.Info("error reading message", "error", err)
// 			conn.Close()
// 			return
// 		}
// 		slog.Info("message recieved", "message", input)

// 		switch input.Type {
// 		case "get_state":
// 		case "remove_member":
// 			var data struct {
// 				MemberID string `json:"member_id"`
// 			}
// 			if err := json.Unmarshal(input.Payload, &data); err != nil {
// 				slog.Warn("failed to unmarshal data", "error", err)
// 				if err := c.writeError(conn, err); err != nil {
// 					slog.Warn("failed to write error", "error", err)
// 					return
// 				}
// 				continue
// 			}

// 			removeVideoResponse, err := c.roomService.RemoveMember(ctx, &room.RemoveMemberParams{
// 				RemovedMemberID: data.MemberID,
// 				SenderID:        memberID,
// 				RoomID:          roomID,
// 			})
// 			if err != nil {
// 				slog.Warn("failed to remove member", "error", err)
// 				if err := c.writeError(conn, err); err != nil {
// 					slog.Warn("failed to write error", "error", err)
// 					return
// 				}
// 				continue
// 			}

// 			if err := c.broadcast(removeVideoResponse.Conns, &Output{
// 				Action: "member_removed",
// 				Data: map[string]any{
// 					"removed_member_id": data.MemberID,
// 					"memberlist":        removeVideoResponse.Memberlist,
// 				},
// 			}); err != nil {
// 				slog.Warn("failed to broadcast", "error", err)
// 				return
// 			}
// 		case "promote_member":
// 			var data struct {
// 				MemberID string `json:"member_id"`
// 			}
// 			if err := json.Unmarshal(input.Payload, &data); err != nil {
// 				slog.Warn("failed to unmarshal data", "error", err)
// 				if err := c.writeError(conn, err); err != nil {
// 					slog.Warn("failed to write error", "error", err)
// 					return
// 				}
// 				continue
// 			}

// 			removeVideoResponse, err := c.roomService.PromoteMember(ctx, &room.PromoteMemberParams{
// 				PromotedMemberID: data.MemberID,
// 				SenderID:         memberID,
// 				RoomID:           roomID,
// 			})
// 			if err != nil {
// 				slog.Warn("failed to promote member", "error", err)
// 				if err := c.writeError(conn, err); err != nil {
// 					slog.Warn("failed to write error", "error", err)
// 					return
// 				}
// 				continue
// 			}

// 			if err := c.broadcast(removeVideoResponse.Conns, &Output{
// 				Action: "member_promoted",
// 				Data: map[string]any{
// 					"promoted_member_id": data.MemberID,
// 				},
// 			}); err != nil {
// 				slog.Warn("failed to broadcast", "error", err)
// 				return
// 			}
// 		// case "demote_member":
// 		// 	demotedMember, err := r.handleDemoteMember(&input)

// 		// 	if err != nil {
// 		// 		r.sendError(input.Sender.Conn, err)
// 		// 	} else {
// 		// 		r.sendMemberDemoted(demotedMember)
// 		// 	}
// 		case "add_video":
// 			var data struct {
// 				VideoURL string `json:"video_url"`
// 			}
// 			if err := json.Unmarshal(input.Payload, &data); err != nil {
// 				slog.Warn("failed to unmarshal data", "error", err)
// 				if err := c.writeError(conn, err); err != nil {
// 					slog.Warn("failed to write error", "error", err)
// 					return
// 				}
// 				continue
// 			}

// 			addVideoResponse, err := c.roomService.AddVideo(ctx, &room.AddVideoParams{
// 				MemberID: memberID,
// 				VideoURL: data.VideoURL,
// 			})
// 			if err != nil {
// 				slog.Warn("failed to add video", "error", err)
// 				if err := c.writeError(conn, err); err != nil {
// 					slog.Warn("failed to write error", "error", err)
// 					return
// 				}
// 				continue
// 			}

// 			if err := c.broadcast(addVideoResponse.Conns, &Output{
// 				Action: "video_added",
// 				Data: map[string]any{
// 					"added_video": addVideoResponse.AddedVideo,
// 					"playlist":    addVideoResponse.Playlist,
// 				},
// 			}); err != nil {
// 				slog.Warn("failed to broadcast", "error", err)
// 				return
// 			}
// 		case "remove_video":
// 			var data struct {
// 				VideoID string `json:"video_id"`
// 			}
// 			if err := json.Unmarshal(input.Payload, &data); err != nil {
// 				slog.Warn("failed to unmarshal data", "error", err)
// 				if err := c.writeError(conn, err); err != nil {
// 					slog.Warn("failed to write error", "error", err)
// 					return
// 				}
// 				continue
// 			}

// 			addVideoResponse, err := c.roomService.RemoveVideo(ctx, &room.RemoveVideoParams{
// 				SenderID: memberID,
// 				VideoID:  data.VideoID,
// 				RoomID:   roomID,
// 			})
// 			if err != nil {
// 				slog.Warn("failed to remove video", "error", err)
// 				if err := c.writeError(conn, err); err != nil {
// 					slog.Warn("failed to write error", "error", err)
// 					return
// 				}
// 				continue
// 			}

// 			if err := c.broadcast(addVideoResponse.Conns, &Output{
// 				Action: "video_removed",
// 				Data: map[string]any{
// 					"removed_video": data.VideoID,
// 					"playlist":      addVideoResponse.Playlist,
// 				},
// 			}); err != nil {
// 				slog.Warn("failed to broadcast", "error", err)
// 				return
// 			}
// 		default:
// 			slog.Warn("unknown action", "action", input.Type)
// 			if err := c.writeError(conn, errors.New("unknown action")); err != nil {
// 				slog.Warn("failed to write error", "error", err)
// 				return
// 			}
// 		}
// 	}
// }
