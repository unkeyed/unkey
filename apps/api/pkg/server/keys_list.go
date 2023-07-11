package server

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"net/http"
)

type ListKeysRequest struct {
	ApiId   string `validate:"required"`
	Limit   int
	Offset  int
	OwnerId string
}

type ratelimitSettng struct {
	Type           string `json:"type"`
	Limit          int64  `json:"limit"`
	RefillRate     int64  `json:"refillRate"`
	RefillInterval int64  `json:"refillInterval"`
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
	Remaining      *int64           `json:"remaining"`
}

type ListKeysResponse struct {
	Keys  []keyResponse `json:"keys"`
	Total int           `json:"total"`
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
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: fmt.Sprintf("unable to validate request: %s", err.Error()),
		})
	}

	authHash, err := getKeyHash(c.Get("Authorization"))
	if err != nil {
		return err
	}

	authKey, err := s.db.GetKeyByHash(ctx, authHash)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to find key: %s", err.Error()),
		})
	}

	if authKey.ForWorkspaceId == "" {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: "wrong key type",
		})
	}

	api, err := s.db.GetApi(ctx, req.ApiId)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(ErrorResponse{
				Code:  NOT_FOUND,
				Error: fmt.Sprintf("unable to find api: %s", req.ApiId),
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to find api: %s", err.Error()),
		})
	}
	if api.WorkspaceId != authKey.ForWorkspaceId {
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:  UNAUTHORIZED,
			Error: "access to workspace denied",
		})
	}

	keys, err := s.db.ListKeysByApiId(ctx, api.Id, req.Limit, req.Offset, req.OwnerId)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: err.Error(),
		})
	}

	total, err := s.db.CountKeys(ctx, api.Id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: err.Error(),
		})
	}

	res := ListKeysResponse{
		Keys:  make([]keyResponse, len(keys)),
		Total: total,
	}

	for i, k := range keys {
		res.Keys[i] = keyResponse{
			Id:             k.Id,
			ApiId:          k.ApiId,
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
		if k.Remaining.Enabled {
			res.Keys[i].Remaining = &k.Remaining.Remaining
		}
	}

	return c.JSON(res)
}
