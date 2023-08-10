package server

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tinybird"
	"github.com/unkeyed/unkey/apps/agent/pkg/whitelist"
	"go.uber.org/zap"
)

type VerifyKeyRequest struct {
	Key string `json:"key"`
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
		} else {
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

	api, isCached := s.apiCache.Get(ctx, key.KeyAuthId)
	if !isCached {
		var found bool
		api, found, err = s.db.FindApiByKeyAuthId(ctx, key.KeyAuthId)
		if err != nil {
			return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())
		}
		if !found {
			return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("keyauth %s not found", key.KeyAuthId))
		} else {

			s.apiCache.Set(ctx, key.KeyAuthId, api)
		}
	}

	// ---------------------------------------------------------------------------------------------
	// Preflight checks
	// ---------------------------------------------------------------------------------------------

	if len(api.IpWhitelist) > 0 {
		sourceIp := c.Get("Fly-Client-IP")
		s.logger.Info("checking ip whitelist", zap.String("sourceIp", sourceIp), zap.Strings("whitelist", api.IpWhitelist))

		if !whitelist.Ip(sourceIp, api.IpWhitelist) {
			s.logger.Info("ip denied", zap.String("workspaceId", api.WorkspaceId), zap.String("apiId", api.Id), zap.String("keyId", key.Id), zap.String("sourceIp", sourceIp), zap.Strings("whitelist", api.IpWhitelist))
			return c.JSON(VerifyKeyResponse{
				Valid: false,
				Code:  errors.FORBIDDEN,
			})

		}
	}

	// ---------------------------------------------------------------------------------------------
	// Start validation
	// ---------------------------------------------------------------------------------------------
	logger := s.logger.With(
		zap.String("keyId", key.Id),
		zap.String("apiId", api.Id),
		zap.String("keyAuthId", key.KeyAuthId),
		zap.String("workspaceId", key.WorkspaceId),
	)

	logger.Info("report.key.verifying")

	res := VerifyKeyResponse{
		Valid:   true,
		OwnerId: key.OwnerId,
		Meta:    key.Meta,
	}

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
		s.keyCache.Set(ctx, key.Hash, keyAfterUpdate)
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

	// ---------------------------------------------------------------------------------------------
	// Send usage to tinybird
	// ---------------------------------------------------------------------------------------------

	if s.tinybird != nil {
		defer func() {
			s.tinybird.PublishKeyVerificationEventChannel() <- tinybird.KeyVerificationEvent{
				WorkspaceId: key.WorkspaceId,
				ApiId:       api.Id,
				KeyId:       key.Id,
				Ratelimited: res.Code == errors.RATELIMITED,
				Time:        time.Now().UnixMilli(),
			}
		}()
	}

	return c.JSON(res)
}
