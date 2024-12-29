package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type repo struct {
	rc                *redis.Client
	maxScoreScript    string
	maxExpireDuration time.Duration
}

func NewRepo(rc *redis.Client, maxExpireDuration time.Duration) *repo {
	return &repo{
		rc: rc,
		maxScoreScript: rc.ScriptLoad(context.Background(), `
			local maxScore = redis.call('ZREVRANGE', KEYS[1], 0, 0, 'WITHSCORES')
			local nextScore = 1
			if #maxScore > 0 then
				nextScore = tonumber(maxScore[2]) + 1
			end
			redis.call('ZADD', KEYS[1], nextScore, ARGV[1])
			return nextScore
		`).Val(),
		maxExpireDuration: maxExpireDuration,
	}
}
