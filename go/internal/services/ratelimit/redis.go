package ratelimit

import (
	"context"
	"fmt"
	"strconv"
)

func (s *service) redisIncr(ctx context.Context, bucketKey string, sequence int64, cost int64) (int64, error) {
	return s.redis.IncrBy(ctx, fmt.Sprintf("ratelimit:v1:%s:window:%d", bucketKey, sequence), cost).Result()
}
func (s *service) redisGet(ctx context.Context, bucketKey string, sequence int64) (int64, error) {
	res, err := s.redis.Get(ctx, fmt.Sprintf("ratelimit:v1:%s:window:%d", bucketKey, sequence)).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(res, 10, 64)
}
