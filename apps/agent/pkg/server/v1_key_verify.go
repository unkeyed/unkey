package server

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type VerifyKeyRequestV1 struct {
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

type VerifyKeyResponseV1 struct {
	Valid     bool               `json:"valid"`
	OwnerId   string             `json:"ownerId,omitempty"`
	Meta      map[string]any     `json:"meta,omitempty"`
	Expires   int64              `json:"expires,omitempty"`
	Remaining *int32             `json:"remaining,omitempty"`
	Ratelimit *ratelimitResponse `json:"ratelimit,omitempty"`
	Code      string             `json:"code,omitempty"`
	Error     string             `json:"error,omitempty"`
}

func (s *Server) v1VerifyKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.verifyKey")
	defer span.End()

	req := VerifyKeyRequestV1{}
	err := c.BodyParser(&req)
	if err != nil {
		return newHttpError(c, BAD_REQUEST, err.Error())
	}

	err = s.validator.Struct(req)
	if err != nil {
		return newHttpError(c, BAD_REQUEST, err.Error())

	}

	svcRes, err := s.keyService.VerifyKey(ctx, &authenticationv1.VerifyKeyRequest{
		Key:        req.Key,
		SourceIp:   c.Get("Fly-Client-IP"),
		Region:     s.region,
		EdgeRegion: util.Pointer(c.Get("Fly-Region")),
		UserAgent:  util.Pointer(c.Get("User-Agent")),
		Resource:   util.Pointer(c.Path()),
	})
	if err != nil {
		return newHttpError(c, INTERNAL_SERVER_ERROR, err.Error())
	}
	s.logger.Info().Msgf("svcRes: %+v", svcRes)

	res := VerifyKeyResponseV1{
		Valid:     svcRes.Valid,
		OwnerId:   svcRes.GetOwnerId(),
		Expires:   svcRes.GetExpires(),
		Remaining: svcRes.Remaining,
		Code:      svcRes.GetCode(),
		Error:     svcRes.GetError(),
	}
	if svcRes.Meta != nil {
		err = json.Unmarshal([]byte(*svcRes.Meta), &res.Meta)
		if err != nil {
			return newHttpError(c, INTERNAL_SERVER_ERROR, err.Error())
		}
	}
	if svcRes.Ratelimit != nil {
		res.Ratelimit = &ratelimitResponse{
			Limit:     svcRes.Ratelimit.Limit,
			Remaining: svcRes.Ratelimit.Remaining,
			Reset:     svcRes.Ratelimit.ResetAt,
		}
	}

	return c.JSON(res)

}
