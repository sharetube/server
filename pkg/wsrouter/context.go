package wsrouter

import "context"

type ctxKey string

const (
	messageTypeKey ctxKey = "message_type"
)

func GetMessageTypeFromCtx(ctx context.Context) string {
	return ctx.Value(messageTypeKey).(string)
}
