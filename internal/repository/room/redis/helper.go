package redis

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/redis/go-redis/v9"
)

func (r repo) addWithIncrement(ctx context.Context, c redis.Scripter, key string, value interface{}) error {
	_, err := c.EvalSha(ctx, r.maxScoreScript, []string{key}, value).Result()
	return err
}

// Same as HSet, but returns error if key already exists (implemented with lua script)
func (r repo) hSetIfNotExists(ctx context.Context, c redis.Scripter, key string, value interface{}) error {
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

			b, err := json.Marshal(fieldValue)
			if err != nil {
				return err
			}

			strValue = string(b)
		}

		args = append(args, redisKey, string(strValue))

	}

	result, err := c.EvalSha(ctx, r.hSetIfNotExistsScript, []string{key}, args...).Result()
	if err != nil {
		return err
	}

	if result == 0 {
		return redis.Nil
	}

	return nil
}
