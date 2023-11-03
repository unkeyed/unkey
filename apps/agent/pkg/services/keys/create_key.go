package keys

import (
	"context"
	"fmt"
	"time"

	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys/keygen"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func (s *keyService) CreateKey(ctx context.Context, req *authenticationv1.CreateKeyRequest) (*authenticationv1.CreateKeyResponse, error) {
	if req.Expires != nil && req.GetExpires() < time.Now().UnixMilli() {
		return nil, errors.New(errors.ErrBadRequest, fmt.Errorf("'expires' must be in the future, did you pass in a timestamp in seconds instead of milliseconds?"))
	}

	if req.ByteLength != nil && req.GetByteLength() < 16 {
		return nil, errors.New(errors.ErrBadRequest, fmt.Errorf("'byteLength' must be greater than 16"))
	}

	byteLength := 16
	if req.ByteLength != nil {
		byteLength = int(req.GetByteLength())

	}

	keyValue, err := keygen.NewV1Key(req.GetPrefix(), int(byteLength))
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, fmt.Errorf("unable to create new key: %w", err))

	}
	// how many chars to store, this includes the prefix, delimiter and the first 4 characters of the key
	startLength := 4
	if req.Prefix != nil {
		startLength = len(req.GetPrefix()) + 5 // one for the delimiter and 4 for the key
	}
	keyHash := hash.Sha256(keyValue)

	newKey := &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   req.GetKeyAuthId(),
		WorkspaceId: req.GetWorkspaceId(),
		Name:        req.Name,
		Hash:        keyHash,
		Start:       keyValue[:startLength],
		OwnerId:     req.OwnerId,
		Meta:        req.Meta,
		CreatedAt:   time.Now().UnixMilli(),
		Expires:     req.Expires,
		Remaining:   req.Remaining,
		Ratelimit:   req.Ratelimit,
	}

	err = s.db.InsertKey(ctx, newKey)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, fmt.Errorf("unable to store key: %w", err))
	}
	if s.events != nil {
		e := events.KeyEvent{}
		e.Type = events.KeyCreated
		e.Key.Id = newKey.KeyId
		e.Key.Hash = newKey.Hash
		s.events.EmitKeyEvent(ctx, e)

	}

	return &authenticationv1.CreateKeyResponse{
		Key:   keyValue,
		KeyId: newKey.KeyId,
	}, nil
}
