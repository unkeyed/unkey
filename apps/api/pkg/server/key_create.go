package server

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chronark/unkey/apps/api/pkg/entities"
	"github.com/chronark/unkey/apps/api/pkg/hash"
	"github.com/chronark/unkey/apps/api/pkg/keys"
	"github.com/chronark/unkey/apps/api/pkg/uid"
	"github.com/gofiber/fiber/v2"
)

type CreateKeyRequest struct {
	ApiId      string         `json:"apiId"`
	Prefix     string         `json:"prefix"`
	ByteLength int            `json:"byteLength"`
	OwnerId    string         `json:"ownerId"`
	Meta       map[string]any `json:"meta"`
	Expires    int64          `json:"expires"`
	Ratelimit  *struct {
		Type           string `json:"type"`
		Limit          int64  `json:"limit"`
		RefillRate     int64  `json:"refillRate"`
		RefillInterval int64  `json:"refillInterval"`
	} `json:"ratelimit"`
	// ForWorkspaceId is used internally when the frontend wants to create a new root key.
	// Therefore we might not want to add this field to our docs.
	ForWorkspaceId string `json:"forWorkspaceId"`
}

type CreateKeyResponse struct {
	Key   string `json:"key"`
	KeyId string `json:"keyId"`
}

func (s *Server) createKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.createKey")
	defer span.End()

	req := CreateKeyRequest{
		// These act as default
		ByteLength: 16,
	}
	err := c.BodyParser(&req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: fmt.Sprintf("unable to parse body: %s", err.Error()),
		})
	}

	err = s.validator.Struct(req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: fmt.Sprintf("unable to validate body: %s", err.Error()),
		})
	}

	if req.Expires > 0 && req.Expires < time.Now().UnixMilli() {
		return c.Status(http.StatusBadRequest).JSON(
			ErrorResponse{
				Code:  BAD_REQUEST,
				Error: "'expires' must be in the future, did you pass in a timestamp in seconds instead of milliseconds?",
			})
	}

	var (
		workspaceId string
		apiId       string
	)
	// If ForWorkspaceId is defined, we need to check for the `UNKEY_APP_AUTH_TOKEN` instead of doing key verification.
	if req.ForWorkspaceId != "" {
		appToken := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(s.unkeyAppAuthToken), []byte(appToken)) == 0 {
			return c.Status(http.StatusUnauthorized).JSON(
				ErrorResponse{
					Code:  UNAUTHORIZED,
					Error: "unauthorized",
				})
		}

		workspaceId = s.unkeyWorkspaceId
		apiId = s.unkeyApiId

	} else {

		if req.ApiId == "" {
			return c.Status(http.StatusBadRequest).JSON(
				ErrorResponse{
					Code:  BAD_REQUEST,
					Error: "'apiId' must be defined",
				})
		}
		apiId = req.ApiId

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

		workspaceId = authKey.WorkspaceId
		apiId = authKey.ApiId

		api, err := s.db.GetApi(ctx, req.ApiId)
		if err != nil {
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
	}

	keyValue, err := keys.NewKey(req.Prefix, req.ByteLength)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: err.Error(),
		})
	}
	split := strings.Split(keyValue, "_")
	keyHash := hash.Sha256(keyValue)

	newKey := entities.Key{
		Id:          uid.Key(),
		ApiId:       apiId,
		WorkspaceId: workspaceId,
		Hash:        keyHash,
		Start:       split[len(split)-1][:4],
		OwnerId:     req.OwnerId,
		Meta:        req.Meta,
		CreatedAt:   time.Now(),
	}
	if req.Expires > 0 {
		newKey.Expires = time.UnixMilli(req.Expires)
	}
	if req.Ratelimit != nil {
		newKey.Ratelimit = &entities.Ratelimit{
			Type:           req.Ratelimit.Type,
			Limit:          req.Ratelimit.Limit,
			RefillRate:     req.Ratelimit.RefillRate,
			RefillInterval: req.Ratelimit.RefillInterval,
		}
	}

	err = s.db.CreateKey(ctx, newKey)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to store key: %s", err.Error()),
		})
	}

	return c.JSON(CreateKeyResponse{
		Key:   keyValue,
		KeyId: newKey.Id,
	})
}
