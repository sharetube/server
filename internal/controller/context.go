package controller

import "context"

type contextKey int

const (
	roomIDCtxKey contextKey = iota
	memberIDCtxKey
)

func (c controller) getRoomIDFromCtx(ctx context.Context) string {
	return ctx.Value(roomIDCtxKey).(string)
}

func (c controller) getMemberIDFromCtx(ctx context.Context) string {
	return ctx.Value(memberIDCtxKey).(string)
}
