package ratelimit

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/go-redis/redis/v8"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type redisRateLimiter struct {
	redis  *goredis.Client
	script *goredis.Script
	logger logging.Logger
}

type RedisConfig struct {
	RedisUrl string
	Logger   logging.Logger
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

			local bucket = redis.call("HMGET", key, "updatedAt", "tokens")
			if bucket[1] == false then
				-- The bucket does not exist yet, so we create it.
				local remaining = maxTokens - requestedTokens

				redis.call("HMSET", key, "updatedAt", now, "tokens", remaining)
				redis.call("PEXPIRE", key, 2 * interval)

				return {remaining, now + interval}
			end

			local lastUpdatedAt = tonumber(bucket[1])
			local newUpdatedAt = now
			local tokens = tonumber(bucket[2])
			local reset = lastUpdatedAt + interval
			local remaining = tokens - requestedTokens

			-- if we have entered a new interval, we need to refill the bucket
			if now >= lastUpdatedAt + interval then
				local numberOfRefills = math.floor((now - lastUpdatedAt)/interval)

				remaining = math.min(maxTokens, math.max(0,tokens) + numberOfRefills * refillRate) - requestedTokens
			
				newUpdatedAt = lastUpdatedAt + numberOfRefills * interval
				reset = lastUpdatedAt + interval 
			end

			-- return early if we are already out of tokens
			if remaining < 0 then
				return {remaining, reset}
			end

			redis.call("HMSET", key, "updatedAt", newUpdatedAt, "tokens", remaining)
			redis.call("PEXPIRE", key, 2 * interval) -- extend the expiration

			return {remaining, reset}


		`),
	}, nil

}

func (r *redisRateLimiter) Take(req RatelimitRequest) RatelimitResponse {

	rawResponse, err := r.script.Run(context.Background(), r.redis, []string{
		req.Identifier,
	},
		req.Max,
		req.RefillInterval,
		req.RefillRate,
		time.Now().UnixMilli(),
		1,
	).Result()

	if err != nil {
		r.logger.Error().Err(err).Msg("unable to run script")
		return RatelimitResponse{
			Pass:      false,
			Limit:     -1,
			Remaining: -1,
			Reset:     time.Now().UnixMilli(),
		}
	}

	iArrCast, ok := rawResponse.([]interface{})
	if !ok {
		r.logger.Error().Msgf("unable to cast script response: %#v\n", rawResponse)
		return RatelimitResponse{
			Pass:      false,
			Limit:     -1,
			Remaining: -1,
			Reset:     time.Now().UnixMilli(),
		}
	}

	remaining, ok := iArrCast[0].(int64)
	if !ok {
		r.logger.Error().Msgf("unable to cast 'remaining' to int64 response: %#v\n", iArrCast[0])

		return RatelimitResponse{
			Pass:      false,
			Limit:     -1,
			Remaining: -1,
			Reset:     time.Now().UnixMilli(),
		}

	}

	reset, ok := iArrCast[1].(int64)
	if !ok {
		r.logger.Error().Msgf("unable to cast 'reset' to int64 response: %#v\n", iArrCast[1])
		return RatelimitResponse{
			Pass:      false,
			Limit:     -1,
			Remaining: -1,
			Reset:     time.Now().UnixMilli(),
		}
	}
	pass := remaining >= 0
	if remaining < 0 {
		remaining = 0
	}

	return RatelimitResponse{
		Pass:      pass,
		Limit:     req.Max,
		Remaining: int32(remaining),
		Reset:     reset,
	}

}
