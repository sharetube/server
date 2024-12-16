package controller

import "context"

type contextKey int

const (
	roomIDCtxKey contextKey = iota
	memberIDCtxKey
)

func (c controller) getRoomIDFromCtx(ctx context.Context) string {
	roomID, ok := ctx.Value(roomIDCtxKey).(string)
	if !ok {
		return ""
	}

	return roomID
}

func (c controller) getMemberIDFromCtx(ctx context.Context) string {
	memberID, ok := ctx.Value(memberIDCtxKey).(string)
	if !ok {
		return ""
	}

	return memberID
}
