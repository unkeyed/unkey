package keys

import (
	"context"
	"fmt"

	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

func (s *keyService) GetKey(ctx context.Context, req *authenticationv1.GetKeyRequest) (*authenticationv1.GetKeyResponse, error) {

	key, found, err := s.db.FindKeyById(ctx, req.KeyId)
	if err != nil {
		return nil, errors.New(errors.ErrInternalServerError, fmt.Errorf("unable to store key: %w", err))
	}
	if !found {
		return nil, errors.New(errors.ErrNotFound, fmt.Errorf("key not found"))
	}
	if key.WorkspaceId != req.AuthorizedWorkspaceId {
		return nil, errors.New(errors.ErrForbidden, fmt.Errorf("workspace not authorized"))
	}

	return &authenticationv1.GetKeyResponse{
		Key: key,
	}, nil
}
