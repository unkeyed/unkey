package ratelimit

import (
	"context"
	"fmt"
	goredis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"time"
)

type redisRateLimiter struct {
	redis  *goredis.Client
	script *goredis.Script
	logger *zap.Logger
}

type RedisConfig struct {
	RedisUrl string
	Logger   *zap.Logger
}

func NewRedis(config RedisConfig) (*redisRateLimiter, error) {
	redisOpts, err := goredis.ParseURL(config.RedisUrl)
	if err != nil {
		return nil, fmt.Errorf("bad redis url: %w", err)
	}

	r := goredis.NewClient(redisOpts)
	err = r.Ping(context.Background()).Err()
	if err != nil {
		return nil, fmt.Errorf("unable to ping redis: %w", err)
	}

	return &redisRateLimiter{
		redis:  r,
		logger: config.Logger,
		script: goredis.NewScript(`
	 local key             = KEYS[1]           -- identifier including prefixes
    local maxTokens       = tonumber(ARGV[1]) -- maximum number of tokens
    local interval        = tonumber(ARGV[2]) -- size of the window in milliseconds
    local refillRate      = tonumber(ARGV[3]) -- how many tokens are refilled after each interval
    local now             = tonumber(ARGV[4]) -- current timestamp in milliseconds
    local requestedTokens = tonumber(ARGV[5]) -- how many tokens are requested for this operation
    local remaining   = 0

    local bucket = redis.call("HMGET", key, "updatedAt", "tokens")

    if bucket[1] == false then
      -- The bucket does not exist yet, so we create it.
      remaining = maxTokens - requestedTokens

      redis.call("HMSET", key, "updatedAt", now, "tokens", remaining)

      return {remaining, now + interval}
    end

    local updatedAt = tonumber(bucket[1])
    local tokens = tonumber(bucket[2])

    if now >= updatedAt + interval then
      local numberOfRefills = math.floor((now - updatedAt)/interval)

      if tokens <= 0 then
        -- No more tokens were left before the refill.
        remaining = math.min(maxTokens, numberOfRefills * refillRate) - requestedTokens
      else
        remaining = math.min(maxTokens, tokens + numberOfRefills * refillRate) - requestedTokens
      end

      local lastRefill = updatedAt + numberOfRefills * interval

      redis.call("HMSET", key, "updatedAt", lastRefill, "tokens", remaining)
      return {remaining, lastRefill + interval}
    end

  remaining = tokens - requestedTokens
  redis.call("HSET", key, "tokens", remaining)
  return {remaining, updatedAt + interval}


		`),
	}, nil

}

func (r *redisRateLimiter) Take(req RatelimitRequest) RatelimitResponse {

	rawResponse, err := r.script.EvalSha(context.Background(), r.redis, []string{
		req.Identifier,
	},
		req.Max,
		req.RefillInterval,
		req.RefillRate,
		time.Now(),
		1,
	).Result()

	if err != nil {
		r.logger.Error("unable to eval sha", zap.Error(err))
		return RatelimitResponse{
			Pass:      false,
			Limit:     -1,
			Remaining: -1,
			Reset:     time.Now().UnixMilli(),
		}
	}

	res, ok := rawResponse.([]int64)
	if !ok {
		return RatelimitResponse{
			Pass:      false,
			Limit:     -1,
			Remaining: -1,
			Reset:     time.Now().UnixMilli(),
		}
	}

	return RatelimitResponse{
		Pass:      res[0] > 0,
		Limit:     req.Max,
		Remaining: res[0],
		Reset:     res[1],
	}

}
