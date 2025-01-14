package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// todo: handle error
func (r repo) addWithIncrement(ctx context.Context, c redis.Cmdable, key string, value interface{}) {
	c.EvalSha(ctx, r.maxScoreScript, []string{key}, value)
}

// todo: handle error
func (r repo) expireKeysWithPrefix(ctx context.Context, c redis.Cmdable, pattern string, expireAt time.Time) {
	c.EvalSha(ctx, r.expireKeysWithPrefixScript, []string{}, pattern, expireAt.Unix())
}

func (r repo) executePipe(ctx context.Context, pipe redis.Pipeliner) error {
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		for _, cmd := range cmds {
			if err := cmd.Err(); err != nil {
				return err
			}
		}

		return err
	}

	return nil
}

func (r repo) fieldToBool(field string) bool {
	return field != "0"
}

func (r repo) fieldToInt(field string) int {
	i, _ := strconv.Atoi(field)
	return i
}

func (r repo) fieldToFload64(field string) float64 {
	f, _ := strconv.ParseFloat(field, 64)
	return f
}
