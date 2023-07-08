package server

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/api/pkg/tinybird"
	"go.uber.org/zap"
	"net/http"
	"time"
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

	hash, err := getKeyHash(req.Key)
	if err != nil {
		return err
	}

	key, isCached := s.cache.Get(ctx, hash)

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
		s.cache.Set(ctx, hash, key)
	}
	if !key.Expires.IsZero() && key.Expires.Before(time.Now()) {
		s.cache.Remove(ctx, hash)
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
