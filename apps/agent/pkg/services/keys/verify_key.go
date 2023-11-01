package keys

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"github.com/unkeyed/unkey/apps/agent/pkg/whitelist"
	"go.opentelemetry.io/otel/trace"
)

const (
	NOT_FOUND          = "NOT_FOUND"
	FORBIDDEN          = "FORBIDDEN"
	KEY_USAGE_EXCEEDED = "KEY_USAGE_EXCEEDED"
	RATELIMITED        = "RATELIMITED"
)

func (s *keyService) VerifyKey(ctx context.Context, req *authenticationv1.VerifyKeyRequest) (*authenticationv1.VerifyKeyResponse, error) {
	if req.Key == "" {
		return nil, errors.New(errors.ErrBadRequest, fmt.Errorf("key is required"))
	}

	s.logger.Debug().Str("key", req.Key).Msg("verifying key")
	keyHash := hash.Sha256(req.Key)
	key, hit := s.keyCache.Get(ctx, keyHash)
	if hit == cache.Null {
		return &authenticationv1.VerifyKeyResponse{
			Valid: false,
			Code:  NOT_FOUND,
		}, nil
	}

	if hit == cache.Miss {
		s.logger.Debug().Str("key", req.Key).Msg("key not found in cache, fetching from db")
		var found bool
		var err error
		key, found, err = s.db.FindKeyByHash(ctx, keyHash)
		if err != nil {
			return nil, errors.New(errors.ErrInternalServerError, err)
		}
		if !found {
			s.keyCache.SetNull(ctx, keyHash)
			return nil, errors.New(errors.ErrNotFound, fmt.Errorf("key not found"))
		} else if key.Remaining == nil {
			s.keyCache.Set(ctx, keyHash, key)
		}
	}

	if key.DeletedAt != nil {
		return &authenticationv1.VerifyKeyResponse{
			Valid: false,
			Code:  NOT_FOUND,
		}, nil
	}

	if key.Expires != nil && time.UnixMilli(key.GetExpires()).Before(time.Now()) {
		s.keyCache.Remove(ctx, keyHash)
		err := s.db.SoftDeleteKey(ctx, key.KeyId)
		if err != nil {
			return nil, errors.New(errors.ErrInternalServerError, err)
		}

		return &authenticationv1.VerifyKeyResponse{
			Valid: false,
			Code:  NOT_FOUND,
		}, nil

	}

	// ---------------------------------------------------------------------------------------------
	// Get the api from either cache or db
	// ---------------------------------------------------------------------------------------------
	api, found, err := cache.WithCache(s.apiCache, s.db.FindApiByKeyAuthId)(ctx, key.KeyAuthId)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, err)
	}
	if !found {
		return nil, errors.New(errors.ErrNotFound, fmt.Errorf("api not found"))
	}

	// ---------------------------------------------------------------------------------------------
	// Preflight checks
	// ---------------------------------------------------------------------------------------------

	if len(api.IpWhitelist) > 0 {
		var ipSpan trace.Span
		ctx, ipSpan = s.tracer.Start(ctx, "server.verifyKey.checkIpWhitelist")
		s.logger.Debug().Str("sourceIp", req.SourceIp).Strs("whitelist", api.IpWhitelist).Msg("checking ip whitelist")
		if !whitelist.Ip(req.SourceIp, api.IpWhitelist) {
			s.logger.Info().Str("workspaceId", api.WorkspaceId).Str("apiId", api.ApiId).Str("keyId", key.KeyId).Str("sourceIp", req.SourceIp).Strs("whitelist", api.IpWhitelist).Msg("ip denied")
			ipSpan.End()
			return &authenticationv1.VerifyKeyResponse{
				Valid: false,
				Code:  FORBIDDEN,
			}, nil

		}
		ipSpan.End()
	}

	// ---------------------------------------------------------------------------------------------
	// Start validation
	// ---------------------------------------------------------------------------------------------
	if s.metrics != nil {
		var sp trace.Span
		ctx, sp = s.tracer.Start(ctx, "server.verifyKey.ReportKeyVerification")
		s.metrics.ReportKeyVerification(metrics.KeyVerificationReport{
			WorkspaceId: key.WorkspaceId,
			ApiId:       api.ApiId,
			KeyId:       key.KeyId,
			KeyAuthId:   key.KeyAuthId,
		})
		sp.End()
	}
	res := &authenticationv1.VerifyKeyResponse{
		Valid:     true,
		OwnerId:   key.OwnerId,
		IsRootKey: key.ForWorkspaceId != nil,
	}
	// key.AuthorizedWorkspaceId must always be the user's workspace id, not ours
	if res.IsRootKey {
		res.AuthorizedWorkspaceId = key.GetForWorkspaceId()
	} else {
		res.AuthorizedWorkspaceId = key.GetWorkspaceId()
	}

	if key.Meta != nil {
		err = json.Unmarshal([]byte(key.GetMeta()), &res.Meta)
		if err != nil {
			return nil, errors.New(errors.ErrInternalServerError, err)
		}
	}

	// ---------------------------------------------------------------------------------------------
	// Send usage to tinybird
	// ---------------------------------------------------------------------------------------------

	defer func() {
		var denied analytics.DeniedReason
		switch res.Code {
		case KEY_USAGE_EXCEEDED:
			denied = analytics.DeniedUsageExceeded
		case RATELIMITED:
			denied = analytics.DeniedRateLimited
		}

		s.analytics.PublishKeyVerificationEvent(ctx, analytics.KeyVerificationEvent{
			WorkspaceId:       key.WorkspaceId,
			ApiId:             api.ApiId,
			KeyId:             key.KeyId,
			Denied:            denied,
			Time:              time.Now().UnixMilli(),
			Region:            req.Region,
			EdgeRegion:        req.GetEdgeRegion(),
			UserAgent:         req.GetUserAgent(),
			IpAddress:         req.SourceIp,
			RequestedResource: req.GetResource(),
		})
	}()

	if key.Expires != nil {
		res.Expires = key.Expires
	}

	if key.Remaining != nil {
		var sp trace.Span
		ctx, sp = s.tracer.Start(ctx, "server.verifyKey.CheckRemainingKeyUsage")
		if *key.Remaining <= 0 {
			res.Valid = false
			res.Code = KEY_USAGE_EXCEEDED
			zero := int32(0)
			res.Remaining = &zero
			return res, nil
		}

		beforeKeyUpdate := time.Now()
		keyAfterUpdate, err := s.db.DecrementRemainingKeyUsage(ctx, key.KeyId)
		s.metrics.ReportDatabaseLatency(metrics.DatabaseLatencyReport{
			Query:   "DecrementRemainingKeyUsage",
			Latency: time.Since(beforeKeyUpdate).Milliseconds(),
		})
		if err != nil {
			sp.End()
			return nil, errors.New(errors.ErrInternalServerError, err)
		}
		if *keyAfterUpdate.Remaining < 0 {
			res.Valid = false
			res.Remaining = util.Pointer(int32(0))
			res.Code = KEY_USAGE_EXCEEDED
			sp.End()
			return res, nil
		} else {
			res.Remaining = keyAfterUpdate.Remaining
		}
		sp.End()

	}

	if key.Ratelimit != nil {
		var sp trace.Span
		ctx, sp = s.tracer.Start(ctx, "server.verifyKey.CheckRatelimit")
		var limiter ratelimit.Ratelimiter
		switch key.Ratelimit.Type {
		case authenticationv1.RatelimitType_RATELIMIT_TYPE_FAST:
			limiter = s.memoryRatelimit
		case authenticationv1.RatelimitType_RATELIMIT_TYPE_CONSISTENT:
			limiter = s.consitentRatelimit
		}
		if limiter != nil {
			r := limiter.Take(ratelimit.RatelimitRequest{
				Identifier:     key.Hash,
				Max:            key.Ratelimit.Limit,
				RefillRate:     key.Ratelimit.RefillRate,
				RefillInterval: key.Ratelimit.RefillInterval,
			})
			res.Ratelimit = &authenticationv1.RatelimitResponse{
				Limit:     r.Limit,
				Remaining: r.Remaining,
				ResetAt:   r.Reset,
			}
			res.Valid = r.Pass
			if !r.Pass {
				res.Code = RATELIMITED
			}
		}
		sp.End()
	}

	return res, nil
}
