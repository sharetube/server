package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type repo struct {
	rc                         *redis.Client
	maxScoreScript             string
	expireKeysWithPrefixScript string
	// maxExpireDuration          time.Duration
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
		expireKeysWithPrefixScript: rc.ScriptLoad(context.Background(), `
			local pattern = ARGV[1]
			local timestamp = ARGV[2]
			local cursor = "0"
			local count = 0

			repeat
				local result = redis.call('SCAN', cursor, 'MATCH', pattern)
				cursor = result[1]
				local keys = result[2]

				for i, key in ipairs(keys) do
					redis.call('EXPIREAT', key, timestamp)
					count = count + 1
				end
			until cursor == "0"

			return count
		`).Val(),
		// maxExpireDuration: maxExpireDuration,
	}
}
