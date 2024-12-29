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
