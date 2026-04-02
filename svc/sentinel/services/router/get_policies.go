package router

import (
	"context"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/sentinel/engine"
)

func (s *service) GetPolicies(ctx context.Context, deployment db.Deployment) ([]*sentinelv1.Policy, error) {
	policies, hit, err := s.policyCache.SWR(ctx, deployment.ID, func(ctx context.Context) ([]*sentinelv1.Policy, error) {
		return engine.ParseMiddleware(deployment.SentinelConfig)
	}, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	})

	if err != nil {
		return nil, err
	}

	if hit == cache.Null {
		return nil, nil
	}

	return policies, nil
}
