package redis

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/redis/go-redis/v9"
)

type Repo struct {
	rc     *redis.Client
	script string
}

func NewRepo(rc *redis.Client) *Repo {
	return &Repo{
		rc: rc,
		script: rc.ScriptLoad(context.Background(), `
        local key = KEYS[1]
        if redis.call('EXISTS', key) == 0 then
            for i = 1, #ARGV, 2 do
                redis.call('HSET', key, ARGV[i], ARGV[i + 1])
            end
            return 1
        end
        return 0
    `).Val(),
	}
}

func (r Repo) HSetIfNotExists(ctx context.Context, c redis.Scripter, key string, value interface{}) error {
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

	result, err := c.EvalSha(ctx, r.script, []string{key}, args...).Result()
	if err != nil {
		return err
	}

	if result == 0 {
		return redis.Nil
	}

	return nil
}
