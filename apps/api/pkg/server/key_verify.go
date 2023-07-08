package server

import (
	"errors"
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
	Remaining int64              `json:"remaining,omitempty"`
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

	api, isCached := s.apiCache.Get(ctx, key.ApiId)

	if !isCached {
		api, err = s.db.GetApi(ctx, key.ApiId)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return c.Status(http.StatusNotFound).JSON(VerifyKeyErrorResponse{
					Valid: false,
					ErrorResponse: ErrorResponse{
						Code:  NOT_FOUND,
						Error: "api not found",
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
		s.apiCache.Set(ctx, key.ApiId, api)
	}

	// ---------------------------------------------------------------------------------------------
	// Preflight checks
	// ---------------------------------------------------------------------------------------------

	s.logger.Info("api", zap.Any("api", api))
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

	s.logger.Info("report.key.verifying",
		zap.String("keyId", key.Id),
		zap.String("apiId", key.ApiId),
		zap.String("workspaceId", key.WorkspaceId),
	)

	res := VerifyKeyResponse{
		Valid:   true,
		OwnerId: key.OwnerId,
		Meta:    key.Meta,
	}
	if !key.Expires.IsZero() {
		res.Expires = key.Expires.UnixMilli()
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

	if s.tinybird != nil {
		s.tinybird.PublishKeyVerificationEventChannel() <- tinybird.KeyVerificationEvent{
			WorkspaceId: key.WorkspaceId,
			ApiId:       key.ApiId,
			KeyId:       key.Id,
			Ratelimited: res.Code == RATELIMITED,
			Time:        time.Now().UnixMilli(),
		}
	}

	return c.JSON(res)
}
