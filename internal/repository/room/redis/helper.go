package redis

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"

	"github.com/redis/go-redis/v9"
)

func (r repo) addWithIncrement(ctx context.Context, c redis.Scripter, key string, value interface{}) {
	c.EvalSha(ctx, r.maxScoreScript, []string{key}, value)
}

// Same as HSet, but returns error if key already exists (implemented with lua script)
func (r repo) hSetIfNotExists(ctx context.Context, c redis.Scripter, key string, value interface{}) {
	v := reflect.ValueOf(value)
	t := v.Type()

	// Create args slice with capacity for all field-value pairs
	args := make([]interface{}, 0, v.NumField()*2)

	// Iterate through struct fields
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		// Get redis tag or use field name
		redisKey := field.Tag.Get("redis")
		if redisKey == "" {
			redisKey = field.Name
		}

		var strValue string
		if v.Field(i).Kind() == reflect.String {
			strValue = v.Field(i).String()
		} else {
			// Convert field value to string
			fieldValue := v.Field(i).Interface()

			b, _ := json.Marshal(fieldValue)

			strValue = string(b)
		}

		args = append(args, redisKey, string(strValue))

	}

	c.EvalSha(ctx, r.hSetIfNotExistsScript, []string{key}, args...)
}

func (r repo) HSetStruct(ctx context.Context, c redis.Pipeliner, key string, value interface{}) error {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	fields := make(map[string]interface{})
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := t.Field(i).Tag.Get("redis")
		if tag == "" {
			tag = t.Field(i).Name
		}

		// Handle nil pointer fields
		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		// Get the actual value for pointer fields
		if field.Kind() == reflect.Ptr {
			fields[tag] = field.Elem().Interface()
		} else {
			fields[tag] = field.Interface()
		}
	}

	return c.HSet(ctx, key, fields).Err()
}

func (r repo) executePipe(ctx context.Context, pipe redis.Pipeliner) error {
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		slog.InfoContext(ctx, "redis tx err != nil", "error", err)
		for _, cmd := range cmds {
			if err := cmd.Err(); err != nil {
				slog.ErrorContext(ctx, "redis tx error", "error", err)
			}
		}

		return err
	}

	return nil
}
