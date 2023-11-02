package keys

import (
	"context"
	"fmt"

	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
)

func (s *keyService) SoftDeleteKey(ctx context.Context, req *authenticationv1.SoftDeleteKeyRequest) (*authenticationv1.SoftDeleteKeyResponse, error) {

	key, found, err := cache.WithCache(s.keyCache, s.db.FindKeyById)(ctx, req.GetKeyId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New(errors.ErrNotFound, fmt.Errorf("key not found"))
	}

	if key.WorkspaceId != req.GetAuthorizedWorkspaceId() {
		return nil, errors.New(errors.ErrUnauthorized, fmt.Errorf("access to workspace denied"))
	}

	err = s.db.SoftDeleteKey(ctx, key.KeyId)
	if err != nil {
		return nil, err
	}

	s.events.EmitKeyEvent(ctx, events.KeyEvent{
		Type: events.KeyDeleted,
		Key: events.Key{
			Id:   key.KeyId,
			Hash: key.Hash,
		},
	})
	return &authenticationv1.SoftDeleteKeyResponse{}, nil
}
