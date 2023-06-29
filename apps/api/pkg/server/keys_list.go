package server

import (
	"errors"
	"fmt"
	"github.com/chronark/unkey/apps/api/pkg/database"
	"github.com/chronark/unkey/apps/api/pkg/entities"
	"github.com/gofiber/fiber/v2"
	"log"
	"net/http"
)

type ListKeysRequest struct {
	ApiId  string `validate:"required"`
	Limit  int
	Offset int
}

type ListKeysResponse struct {
	Keys  []entities.Key `json:"keys"`
	Total int            `json:"total"`
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

	log.Printf("req %s  %+v %+v\n", c.OriginalURL(), req, c.Queries())

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

	keys, err := s.db.ListKeysByApiId(ctx, api.Id, req.Limit, req.Offset)
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
	return c.JSON(ListKeysResponse{
		Keys:  keys,
		Total: total,
	})
}
