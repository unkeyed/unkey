package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/chronark/unkey/apps/api/pkg/database"
	"github.com/gofiber/fiber/v2"
)

type GetApiRequest struct {
	ApiId string `json:"apiId" validate:"required"`
}

type GetApiResponse struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	WorkspaceId string `json:"workspaceId"`
}

func (s *Server) getApi(c *fiber.Ctx) error {
	ctx := c.UserContext()

	req := GetApiRequest{
		ApiId: c.Params("apiId"),
	}

	err := s.validator.Struct(req)
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

	return c.JSON(GetApiResponse{
		Id:          api.Id,
		Name:        api.Name,
		WorkspaceId: api.WorkspaceId,
	})
}
