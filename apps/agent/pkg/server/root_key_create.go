package server

import (
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys/keygen"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type CreateRootKeyRequest struct {
	Name    string `json:"name"`
	Expires int64  `json:"expires"`

	// ForWorkspaceId is used internally when the frontend wants to create a new root key.
	// Therefore we might not want to add this field to our docs.
	ForWorkspaceId string `json:"forWorkspaceId" validate:"required"`

	OwnerId string `json:"ownerId"`
}

type CreateRootKeyResponse struct {
	Key   string `json:"key"`
	KeyId string `json:"keyId"`
}

func (s *Server) createRootKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.createRootKey")
	defer span.End()

	appToken := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(s.unkeyAppAuthToken), []byte(appToken)) == 0 {
		return newHttpError(c, UNAUTHORIZED, "unauthorized")
	}

	req := CreateRootKeyRequest{}
	err := c.BodyParser(&req)
	if err != nil {
		return newHttpError(c, BAD_REQUEST, err.Error())
	}

	err = s.validator.Struct(req)
	if err != nil {
		return newHttpError(c, BAD_REQUEST, err.Error())
	}

	if req.Expires > 0 && req.Expires < time.Now().UnixMilli() {
		return newHttpError(c, BAD_REQUEST, "'expires' must be in the future, did you pass in a timestamp in seconds instead of milliseconds?")
	}

	keyValue, err := keygen.NewV1Key("unkey", 16)
	if err != nil {
		return newHttpError(c, INTERNAL_SERVER_ERROR, err.Error())
	}
	separatorIndex := strings.Index(keyValue, "_")
	keyHash := hash.Sha256(keyValue)

	newKey := &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   s.unkeyKeyAuthId,
		WorkspaceId: s.unkeyWorkspaceId,
		OwnerId:     util.Pointer(req.OwnerId),
		Name:        util.Pointer(req.Name),
		Hash:        keyHash,
		Start:       keyValue[0 : separatorIndex+4],
		CreatedAt:   time.Now().UnixMilli(),
		Ratelimit: &authenticationv1.Ratelimit{
			Type:           authenticationv1.RatelimitType_RATELIMIT_TYPE_FAST,
			Limit:          100,
			RefillRate:     10,
			RefillInterval: 1000,
		},
		ForWorkspaceId: util.Pointer(req.ForWorkspaceId),
	}
	if req.Expires > 0 {
		newKey.Expires = util.Pointer(req.Expires)
	}

	err = s.db.InsertKey(ctx, newKey)
	if err != nil {
		return newHttpError(c, INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to store key: %s", err.Error()))
	}
	if s.events != nil {
		e := events.KeyEvent{}
		e.Type = events.KeyCreated
		e.Key.Id = newKey.KeyId
		e.Key.Hash = newKey.Hash
		s.events.EmitKeyEvent(ctx, e)

	}
	return c.JSON(CreateRootKeyResponse{
		Key:   keyValue,
		KeyId: newKey.KeyId,
	})
}
