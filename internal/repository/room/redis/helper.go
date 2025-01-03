package redis

import (
	"context"
	"reflect"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func (r repo) addWithIncrement(ctx context.Context, c redis.Scripter, key string, value interface{}) {
	c.EvalSha(ctx, r.maxScoreScript, []string{key}, value)
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

func (r repo) omitPointers(fields map[string]interface{}) map[string]interface{} {
	omitted := make(map[string]interface{})
	for k, v := range fields {
		if reflect.ValueOf(v).Kind() != reflect.Ptr {
			omitted[k] = v
		}
	}

	return omitted
}

func (r repo) fieldToBool(field string) bool {
	return field == "1"
}

func (r repo) fieldToInt(field string) int {
	i, _ := strconv.Atoi(field)
	return i
}

func (r repo) fieldToFload64(field string) float64 {
	f, _ := strconv.ParseFloat(field, 64)
	return f
}
