package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type repo struct {
	rc                    *redis.Client
	hSetIfNotExistsScript string
	maxScoreScript        string
	expireDuration        time.Duration
}

func NewRepo(rc *redis.Client, expireDuration time.Duration) *repo {
	return &repo{
		rc: rc,
		hSetIfNotExistsScript: rc.ScriptLoad(context.Background(), `
			local key = KEYS[1]
			if redis.call('EXISTS', key) == 0 then
				for i = 1, #ARGV, 2 do
					redis.call('HSET', key, ARGV[i], ARGV[i + 1])
				end
				return 1
			end
			return 0
		`).Val(),
		maxScoreScript: rc.ScriptLoad(context.Background(), `
			local maxScore = redis.call('ZREVRANGE', KEYS[1], 0, 0, 'WITHSCORES')
			local nextScore = 1
			if #maxScore > 0 then
				nextScore = tonumber(maxScore[2]) + 1
			end
			redis.call('ZADD', KEYS[1], nextScore, ARGV[1])
			return nextScore
		`).Val(),
		expireDuration: expireDuration,
	}
}
