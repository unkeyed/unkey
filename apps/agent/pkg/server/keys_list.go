package server

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

type ListKeysRequest struct {
	ApiId   string `validate:"required"`
	Limit   int
	Offset  int
	OwnerId string
}

type ratelimitSettng struct {
	Type           string `json:"type"`
	Limit          int32  `json:"limit"`
	RefillRate     int32  `json:"refillRate"`
	RefillInterval int32  `json:"refillInterval"`
}

type keyResponse struct {
	Id             string           `json:"id"`
	ApiId          string           `json:"apiId"`
	WorkspaceId    string           `json:"workspaceId"`
	Name           string           `json:"name,omitempty"`
	Start          string           `json:"start"`
	OwnerId        string           `json:"ownerId,omitempty"`
	Meta           map[string]any   `json:"meta,omitempty"`
	CreatedAt      int64            `json:"createdAt,omitempty"`
	Expires        int64            `json:"expires,omitempty"`
	Ratelimit      *ratelimitSettng `json:"ratelimit,omitempty"`
	ForWorkspaceId string           `json:"forWorkspaceId,omitempty"`
	Remaining      *int32           `json:"remaining,omitempty"`
}

type ListKeysResponse struct {
	Keys  []keyResponse `json:"keys"`
	Total int64         `json:"total"`
}

func (s *Server) listKeys(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.listKeys")
	defer span.End()
	req := ListKeysRequest{
		ApiId: c.Params("apiId"),
	}
	var err error
	req.Limit = c.QueryInt("limit", 100)
	req.Offset = c.QueryInt("offset", 0)
	req.OwnerId = c.Query("ownerId")

	err = s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, fmt.Sprintf("unable to validate request: %s", err.Error()))
	}

	authorizedWorkspaceId, err := s.authorizeRootKey(ctx, c)
	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}
	api, found, err := cache.WithCache(s.apiCache, s.db.FindApi)(ctx, req.ApiId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("unable to find api %s", req.ApiId))

	}
	if api.WorkspaceId != authorizedWorkspaceId {
		return errors.NewHttpError(c, errors.FORBIDDEN, "workspace access denined")
	}

	keys, err := s.db.ListKeys(ctx, api.GetKeyAuthId(), req.OwnerId, req.Limit, req.Offset)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())

	}

	total, err := s.db.CountKeys(ctx, api.GetKeyAuthId())
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())
	}

	res := ListKeysResponse{
		Keys:  make([]keyResponse, len(keys)),
		Total: total,
	}

	for i, k := range keys {
		res.Keys[i] = keyResponse{
			Id:             k.KeyId,
			ApiId:          api.ApiId,
			WorkspaceId:    k.WorkspaceId,
			Name:           k.GetName(),
			Start:          k.Start,
			OwnerId:        k.GetOwnerId(),
			CreatedAt:      k.CreatedAt,
			ForWorkspaceId: k.GetForWorkspaceId(),
		}
		if k.Meta != nil {
			err = json.Unmarshal([]byte(k.GetMeta()), &res.Keys[i].Meta)
			if err != nil {
				return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to unmarshal meta: %s", err.Error()))
			}
		}
		if k.Expires != nil {
			res.Keys[i].Expires = *k.Expires
		}
		if k.Ratelimit != nil {
			res.Keys[i].Ratelimit = &ratelimitSettng{
				Limit:          k.Ratelimit.Limit,
				RefillRate:     k.Ratelimit.RefillRate,
				RefillInterval: k.Ratelimit.RefillInterval,
			}
			switch k.Ratelimit.Type {
			case authenticationv1.RatelimitType_RATELIMIT_TYPE_FAST:
				res.Keys[i].Ratelimit.Type = "fast"
			case authenticationv1.RatelimitType_RATELIMIT_TYPE_CONSISTENT:
				res.Keys[i].Ratelimit.Type = "consistent"
			}
		}
		if k.Remaining != nil {
			remaining := *k.Remaining
			res.Keys[i].Remaining = &remaining
		}
	}

	return c.JSON(res)
}
