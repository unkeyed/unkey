package keys

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (s *service) Verify(ctx context.Context, hash string) (VerifyResponse, error) {

	err := assert.NotEmpty(hash)
	if err != nil {
		return VerifyResponse{}, fault.Wrap(err, fault.WithDesc("hash is empty", ""))
	}

	key, found := s.cache.SWR(ctx, hash)
	if !found {
		return VerifyResponse{}, fault.New(
			"key does not exist",
			fault.WithTag(fault.NOT_FOUND),
			fault.WithDesc("key does not exist", "We could not find the requested key."),
		)
	}

	// Following are various checks to ensure the validity of the key
	// - Is it enabled?
	// - Is it expired?
	// - Is it ratelimited?

	if key.DeletedAt.IsZero() {
		return VerifyResponse{}, fault.New(
			"key is deleted",
			fault.WithDesc("deleted_at is non-zero", "The key has been deleted."),
		)
	}
	if !key.Enabled {
		return VerifyResponse{}, fault.New(
			"key is disabled",
			fault.WithDesc("", "The key is disabled."),
		)
	}

	res := VerifyResponse{
		AuthorizedWorkspaceID: key.WorkspaceID,
		KeyID:                 key.ID,
	}
	// Root keys store the user's workspace id in `ForWorkspaceID` and we're
	// interested in the user, not our rootkey workspace.
	if key.ForWorkspaceID != "" {
		res.AuthorizedWorkspaceID = key.ForWorkspaceID
	}

	return res, nil
}
