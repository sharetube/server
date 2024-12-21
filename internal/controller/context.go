package controller

import "context"

type contextKey int

const (
	roomIdCtxKey contextKey = iota
	memberIdCtxKey
	requestIdCtxKey
)

func (c controller) getRoomIdFromCtx(ctx context.Context) string {
	roomId, ok := ctx.Value(roomIdCtxKey).(string)
	if !ok {
		return ""
	}

	return roomId
}

func (c controller) getMemberIdFromCtx(ctx context.Context) string {
	memberId, ok := ctx.Value(memberIdCtxKey).(string)
	if !ok {
		return ""
	}

	return memberId
}
