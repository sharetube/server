package redisclient

import (
	"context"
	"fmt"

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
		DB:       0,
	})

	if err := r.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return r, nil
}
