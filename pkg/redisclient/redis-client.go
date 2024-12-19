package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Host     string
	Port     int
	Password string
}

func NewRedisClient(cfg *Config) (*redis.Client, error) {
	r := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
	})

	if err := r.Set(context.Background(), "key", "value", time.Second).Err(); err != nil {
		return nil, err
	}

	return r, nil
}
