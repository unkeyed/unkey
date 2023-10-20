package keys

import (
	"context"
	"fmt"

	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
)

func (s *keyService) SoftDeleteKey(ctx context.Context, req *keysv1.SoftDeleteKeyRequest) (*keysv1.SoftDeleteKeyResponse, error) {

	key, found, err := cache.WithCache(s.keyCache, s.db.FindKeyById)(ctx, req.GetKeyId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New(errors.ErrNotFound, fmt.Errorf("key not found"))
	}

	err = s.db.SoftDeleteKey(ctx, key.Id)
	if err != nil {
		return nil, err
	}

	s.events.EmitKeyEvent(ctx, events.KeyEvent{
		Type: events.KeyDeleted,
		Key: events.Key{
			Id:   key.Id,
			Hash: key.Hash,
		},
	})
	return &keysv1.SoftDeleteKeyResponse{}, nil
}
