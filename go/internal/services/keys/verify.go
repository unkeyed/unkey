package keys

import (
	"context"
	"database/sql"
	"errors"

	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
)

func (s *service) Verify(ctx context.Context, rawKey string) (VerifyResponse, error) {

	err := assert.NotEmpty(rawKey)
	if err != nil {
		return VerifyResponse{}, fault.Wrap(err, fault.WithDesc("rawKey is empty", ""))
	}
	h := hash.Sha256(rawKey)

	key, err := db.Query.FindKeyByHash(ctx, s.db.RO(), h)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return VerifyResponse{}, fault.Wrap(
				err,
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("key does not exist", "We could not find the requested key."),
			)
		}
		return VerifyResponse{}, fault.Wrap(
			err,
			fault.WithDesc("unable to load key", "We could not load the requested key."),
		)
	}

	// Following are various checks to ensure the validity of the key
	// - Is it enabled?
	// - Is it expired?
	// - Is it ratelimited?

	if key.DeletedAtM.Valid {
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
	if key.ForWorkspaceID.Valid {
		res.AuthorizedWorkspaceID = key.ForWorkspaceID.String
	}

	return res, nil
}
