package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
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

	authorizedWorkspaceId, err := s.authorizeRootKey(ctx, c.Get(authorizationHeader))
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

	keys, err := s.db.ListKeys(ctx, api.KeyAuthId, req.OwnerId, req.Limit, req.Offset)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())

	}

	total, err := s.db.CountKeys(ctx, api.KeyAuthId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())
	}

	res := ListKeysResponse{
		Keys:  make([]keyResponse, len(keys)),
		Total: total,
	}

	for i, k := range keys {
		res.Keys[i] = keyResponse{
			Id:             k.Id,
			ApiId:          api.Id,
			WorkspaceId:    k.WorkspaceId,
			Start:          k.Start,
			OwnerId:        k.OwnerId,
			Meta:           k.Meta,
			CreatedAt:      k.CreatedAt.UnixMilli(),
			ForWorkspaceId: k.ForWorkspaceId,
		}
		if !k.Expires.IsZero() {
			res.Keys[i].Expires = k.Expires.UnixMilli()
		}
		if k.Ratelimit != nil {
			res.Keys[i].Ratelimit = &ratelimitSettng{
				Type:           k.Ratelimit.Type,
				Limit:          k.Ratelimit.Limit,
				RefillRate:     k.Ratelimit.RefillRate,
				RefillInterval: k.Ratelimit.RefillInterval,
			}
		}
		if k.Remaining != nil {
			remaining := *k.Remaining
			res.Keys[i].Remaining = &remaining
		}
	}

	return c.JSON(res)
}
