package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/api/pkg/tinybird"
	"github.com/unkeyed/unkey/apps/api/pkg/whitelist"
	"go.uber.org/zap"
)

type VerifyKeyRequest struct {
	Key string `json:"key"`
}

// part of the response
type ratelimitResponse struct {
	Limit     int64 `json:"limit"`
	Remaining int64 `json:"remaining"`
	Reset     int64 `json:"reset"`
}

type VerifyKeyResponse struct {
	Valid     bool               `json:"valid"`
	OwnerId   string             `json:"ownerId,omitempty"`
	Meta      map[string]any     `json:"meta,omitempty"`
	Expires   int64              `json:"expires,omitempty"`
	Remaining *int64             `json:"remaining,omitempty"`
	Ratelimit *ratelimitResponse `json:"ratelimit,omitempty"`
	Code      string             `json:"code,omitempty"`
}

type VerifyKeyErrorResponse struct {
	ErrorResponse
	Valid     bool               `json:"valid"`
	Ratelimit *ratelimitResponse `json:"ratelimit,omitempty"`
}

func (s *Server) verifyKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.verifyKey")
	defer span.End()

	req := VerifyKeyRequest{}
	err := c.BodyParser(&req)
	if err != nil {
		return c.Status(400).JSON(VerifyKeyErrorResponse{
			Valid: false,
			ErrorResponse: ErrorResponse{
				Code:  BAD_REQUEST,
				Error: err.Error(),
			},
		})
	}

	err = s.validator.Struct(req)
	if err != nil {
		return c.Status(400).JSON(VerifyKeyErrorResponse{
			Valid: false,
			ErrorResponse: ErrorResponse{
				Code:  BAD_REQUEST,
				Error: err.Error(),
			},
		})
	}

	// ---------------------------------------------------------------------------------------------
	// Get the key from either cache or db
	// ---------------------------------------------------------------------------------------------
	hash, err := getKeyHash(req.Key)
	if err != nil {
		return err
	}

	key, isCached := s.keyCache.Get(ctx, hash)

	if !isCached {
		key, err = s.db.GetKeyByHash(ctx, hash)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return c.Status(http.StatusNotFound).JSON(VerifyKeyErrorResponse{
					Valid: false,
					ErrorResponse: ErrorResponse{
						Code:  NOT_FOUND,
						Error: "key not found",
					},
				})
			}

			return c.Status(500).JSON(VerifyKeyErrorResponse{
				Valid: false,
				ErrorResponse: ErrorResponse{
					Code:  INTERNAL_SERVER_ERROR,
					Error: err.Error(),
				},
			})
		}
		s.keyCache.Set(ctx, hash, key)
	}
	if !key.Expires.IsZero() && key.Expires.Before(time.Now()) {
		s.keyCache.Remove(ctx, hash)
		err := s.db.DeleteKey(ctx, key.Id)
		if err != nil {
			return c.Status(500).JSON(VerifyKeyErrorResponse{
				Valid: false,
				ErrorResponse: ErrorResponse{
					Code:  INTERNAL_SERVER_ERROR,
					Error: "key not found",
				},
			})
		}
		return c.Status(404).JSON(VerifyKeyErrorResponse{
			Valid: false,
			ErrorResponse: ErrorResponse{
				Code:  NOT_FOUND,
				Error: "key not found",
			},
		})
	}

	// ---------------------------------------------------------------------------------------------
	// Get the api from either cache or db
	// ---------------------------------------------------------------------------------------------

	api, isCached := s.apiCache.Get(ctx, key.KeyAuthId)
	if !isCached {
		keyAuth, err := s.db.GetKeyAuth(ctx, key.KeyAuthId)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return c.Status(http.StatusNotFound).JSON(VerifyKeyErrorResponse{
					Valid: false,
					ErrorResponse: ErrorResponse{
						Code:  NOT_FOUND,
						Error: fmt.Sprintf("keyAuth not found: %s", key.KeyAuthId),
					},
				})
			}

			return c.Status(500).JSON(VerifyKeyErrorResponse{
				Valid: false,
				ErrorResponse: ErrorResponse{
					Code:  INTERNAL_SERVER_ERROR,
					Error: err.Error(),
				},
			})
		}

		api, err = s.db.GetApiByKeyAuthId(ctx, keyAuth.Id)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return c.Status(http.StatusNotFound).JSON(VerifyKeyErrorResponse{
					Valid: false,
					ErrorResponse: ErrorResponse{
						Code:  NOT_FOUND,
						Error: fmt.Sprintf("api not found: %s", keyAuth.Id),
					},
				})
			}

			return c.Status(500).JSON(VerifyKeyErrorResponse{
				Valid: false,
				ErrorResponse: ErrorResponse{
					Code:  INTERNAL_SERVER_ERROR,
					Error: err.Error(),
				},
			})
		}
		s.apiCache.Set(ctx, key.KeyAuthId, api)
	}

	// ---------------------------------------------------------------------------------------------
	// Preflight checks
	// ---------------------------------------------------------------------------------------------

	if len(api.IpWhitelist) > 0 {
		sourceIp := c.Get("Fly-Client-IP")
		s.logger.Info("checking ip whitelist", zap.String("sourceIp", sourceIp), zap.Strings("whitelist", api.IpWhitelist))

		if !whitelist.Ip(sourceIp, api.IpWhitelist) {
			s.logger.Info("ip denied", zap.String("workspaceId", api.WorkspaceId), zap.String("apiId", api.Id), zap.String("keyId", key.Id), zap.String("sourceIp", sourceIp), zap.Strings("whitelist", api.IpWhitelist))
			return c.Status(http.StatusForbidden).JSON(ErrorResponse{
				Code: FORBIDDEN,
			})
		}
	}

	// ---------------------------------------------------------------------------------------------
	// Start validation
	// ---------------------------------------------------------------------------------------------
	logger := s.logger.With(
		zap.String("keyId", key.Id),
		zap.String("keyAuthId", key.KeyAuthId),
		zap.String("workspaceId", key.WorkspaceId),
	)

	logger.Info("report.key.verifying")

	res := VerifyKeyResponse{
		Valid:   true,
		OwnerId: key.OwnerId,
		Meta:    key.Meta,
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
				Ratelimited: res.Code == RATELIMITED,
				Time:        time.Now().UnixMilli(),
			}
		}()
	}

	if !key.Expires.IsZero() {
		res.Expires = key.Expires.UnixMilli()
	}

	if key.Remaining.Enabled {
		if key.Remaining.Remaining <= 0 {
			res.Valid = false
			res.Code = USAGE_EXCEEDED
			zero := int64(0)
			res.Remaining = &zero
			return c.JSON(res)
		}

		remainingAfter, err := s.db.DecrementRemainingKeyUsage(ctx, key.Id)
		if err != nil {
			return c.Status(500).JSON(VerifyKeyErrorResponse{
				Valid: false,
				ErrorResponse: ErrorResponse{
					Code:  INTERNAL_SERVER_ERROR,
					Error: err.Error(),
				},
			})
		}
		key.Remaining.Remaining = remainingAfter
		res.Remaining = &remainingAfter
		s.keyCache.Set(ctx, key.Hash, key)
		if remainingAfter < 0 {
			res.Valid = false
			res.Code = USAGE_EXCEEDED
			return c.JSON(res)
		}

	}

	if key.Ratelimit != nil {
		var limiter ratelimit.Ratelimiter
		switch key.Ratelimit.Type {
		case "fast":
			limiter = s.ratelimit
			break
		case "consistent":
			limiter = s.globalRatelimit
			break
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
				res.Code = RATELIMITED
			}
		}
	}

	return c.JSON(res)
}
