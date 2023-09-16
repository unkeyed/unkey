package server

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/whitelist"
)

type VerifyKeyRequest struct {
	Key string `json:"key" validate:"required"`
	X   struct {
		Resource string `json:"resource,omitempty"`
	} `json:"x,omitempty"`
}

// part of the response
type ratelimitResponse struct {
	Limit     int32 `json:"limit"`
	Remaining int32 `json:"remaining"`
	Reset     int64 `json:"reset"`
}

type VerifyKeyResponse struct {
	Valid     bool               `json:"valid"`
	OwnerId   string             `json:"ownerId,omitempty"`
	Meta      map[string]any     `json:"meta,omitempty"`
	Expires   int64              `json:"expires,omitempty"`
	Remaining *int32             `json:"remaining,omitempty"`
	Ratelimit *ratelimitResponse `json:"ratelimit,omitempty"`
	Code      string             `json:"code,omitempty"`
	Error     string             `json:"error,omitempty"`
}

func (s *Server) verifyKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.verifyKey")
	defer span.End()

	req := VerifyKeyRequest{}
	err := c.BodyParser(&req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, err.Error())
	}

	err = s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, err.Error())

	}

	// ---------------------------------------------------------------------------------------------
	// Get the key from either cache or db
	// ---------------------------------------------------------------------------------------------
	hash, err := getKeyHash(req.Key)
	if err != nil {
		return c.JSON(VerifyKeyResponse{
			Valid: false,
			Code:  errors.NOT_FOUND,
		})
	}
	key, isCached := s.keyCache.Get(ctx, hash)
	if !isCached {
		var found bool
		key, found, err = s.db.FindKeyByHash(ctx, hash)
		if err != nil {
			return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())
		}
		if !found {
			return c.JSON(VerifyKeyResponse{
				Valid: false,
				Code:  errors.NOT_FOUND,
			})
		} else if key.Remaining == nil {
			s.keyCache.Set(ctx, hash, key)
		}
	}
	if !key.Expires.IsZero() && key.Expires.Before(time.Now()) {
		s.keyCache.Remove(ctx, hash)
		err := s.db.DeleteKey(ctx, key.Id)
		if err != nil {
			return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())
		}

		return c.JSON(VerifyKeyResponse{
			Valid: false,
			Code:  errors.NOT_FOUND,
		})

	}

	// ---------------------------------------------------------------------------------------------
	// Get the api from either cache or db
	// ---------------------------------------------------------------------------------------------
	api, found, err := withCache(s.apiCache, s.db.FindApiByKeyAuthId)(ctx, key.KeyAuthId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("keyauth %s not found", key.KeyAuthId))
	}

	// ---------------------------------------------------------------------------------------------
	// Preflight checks
	// ---------------------------------------------------------------------------------------------

	if len(api.IpWhitelist) > 0 {
		sourceIp := c.Get("Fly-Client-IP")
		s.logger.Debug().Str("sourceIp", sourceIp).Strs("whitelist", api.IpWhitelist).Msg("checking ip whitelist")

		if !whitelist.Ip(sourceIp, api.IpWhitelist) {
			s.logger.Info().Str("workspaceId", api.WorkspaceId).Str("apiId", api.Id).Str("keyId", key.Id).Str("sourceIp", sourceIp).Strs("whitelist", api.IpWhitelist).Msg("ip denied")
			return c.JSON(VerifyKeyResponse{
				Valid: false,
				Code:  errors.FORBIDDEN,
			})

		}
	}

	// ---------------------------------------------------------------------------------------------
	// Start validation
	// ---------------------------------------------------------------------------------------------

	if s.metrics != nil {
		s.metrics.ReportKeyVerification(metrics.KeyVerificationReport{
			WorkspaceId: key.WorkspaceId,
			ApiId:       api.Id,
			KeyId:       key.Id,
			KeyAuthId:   key.KeyAuthId,
		})
	}

	res := VerifyKeyResponse{
		Valid:   true,
		OwnerId: key.OwnerId,
		Meta:    key.Meta,
	}

	// ---------------------------------------------------------------------------------------------
	// Send usage to tinybird
	// ---------------------------------------------------------------------------------------------

	defer func() {
		var denied analytics.DeniedReason
		switch res.Code {
		case errors.KEY_USAGE_EXCEEDED:
			denied = analytics.DeniedUsageExceeded
		case errors.RATELIMITED:
			denied = analytics.DeniedRateLimited
		}

		s.analytics.PublishKeyVerificationEvent(ctx, analytics.KeyVerificationEvent{
			WorkspaceId:       key.WorkspaceId,
			ApiId:             api.Id,
			KeyId:             key.Id,
			Denied:            denied,
			Time:              time.Now().UnixMilli(),
			Region:            s.region,
			EdgeRegion:        c.Get("Fly-Region"),
			UserAgent:         c.Get("User-Agent"),
			IpAddress:         c.Get("Fly-Client-IP"),
			RequestedResource: req.X.Resource,
		})
	}()

	if !key.Expires.IsZero() {
		res.Expires = key.Expires.UnixMilli()
	}

	if key.Remaining != nil {
		if *key.Remaining <= 0 {
			res.Valid = false
			res.Code = errors.KEY_USAGE_EXCEEDED
			zero := int32(0)
			res.Remaining = &zero
			return c.JSON(res)
		}

		keyAfterUpdate, err := s.db.DecrementRemainingKeyUsage(ctx, key.Id)
		if err != nil {
			return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())
		}
		if *keyAfterUpdate.Remaining < 0 {
			res.Valid = false
			zero := int32(0)
			res.Remaining = &zero
			res.Code = errors.KEY_USAGE_EXCEEDED
			return c.JSON(res)
		} else {
			res.Remaining = keyAfterUpdate.Remaining
		}

	}

	if key.Ratelimit != nil {
		var limiter ratelimit.Ratelimiter
		switch key.Ratelimit.Type {
		case "fast":
			limiter = s.ratelimit
		case "consistent":
			limiter = s.globalRatelimit
		}
		if limiter != nil {
			r := limiter.Take(ratelimit.RatelimitRequest{
				Identifier:     key.Hash,
				Max:            key.Ratelimit.Limit,
				RefillRate:     key.Ratelimit.RefillRate,
				RefillInterval: key.Ratelimit.RefillInterval,
			})
			res.Ratelimit = &ratelimitResponse{
				Limit:     r.Limit,
				Remaining: r.Remaining,
				Reset:     r.Reset,
			}
			res.Valid = r.Pass
			if !r.Pass {
				res.Code = errors.RATELIMITED
			}
		}
	}

	return c.JSON(res)
}
