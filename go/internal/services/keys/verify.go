package keys

import (
	"context"

	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
)

func (s *service) Verify(ctx context.Context, rawKey string) (VerifyResponse, error) {
	ctx, span := tracing.Start(ctx, "keys.VerifyRootKey")
	defer span.End()

	err := assert.NotEmpty(rawKey)
	if err != nil {
		return VerifyResponse{}, fault.Wrap(err, fault.WithDesc("rawKey is empty", ""))
	}
	h := hash.Sha256(rawKey)

	key, err := s.keyCache.SWR(ctx, h, func(ctx context.Context) (db.Key, error) {
		return db.Query.FindKeyByHash(ctx, s.db.RO(), h)
	}, caches.DefaultFindFirstOp)

	if db.IsNotFound(err) {
		return VerifyResponse{}, fault.Wrap(
			err,
			fault.WithCode(codes.Auth.Authentication.KeyNotFound.URN()),
			fault.WithDesc("key does not exist", "We could not find the requested key."),
		)
	}

	if err != nil {

		return VerifyResponse{}, fault.Wrap(
			err,
			fault.WithDesc("unable to load key", "We could not load the requested key."),
		)
	}

	// Following are various checks to ensure the validity of the key
	// - Is it enabled?
	// - Is it deleted?
	// - Is it expired?
	// - Is it ratelimited?
	// - Is the related workspace deleted?
	// - Is the related workspace disabled?
	// - Is the related forWorkspace deleted?
	// - Is the related forWorkspace disabled?

	if key.DeletedAtM.Valid {
		return VerifyResponse{}, fault.New(
			"key is deleted",
			fault.WithCode(codes.Data.Key.NotFound.URN()),
			fault.WithDesc("deleted_at is non-zero", "The key has been deleted."),
		)
	}
	if !key.Enabled {

		return VerifyResponse{}, fault.New(
			"key is disabled",
			fault.WithCode(codes.Auth.Authorization.KeyDisabled.URN()),
			fault.WithDesc("disabled", "The key is disabled."),
		)
	}

	authorizedWorkspaceID := key.WorkspaceID
	if key.ForWorkspaceID.Valid {
		authorizedWorkspaceID = key.ForWorkspaceID.String
	}

	ws, err := s.workspaceCache.SWR(ctx, authorizedWorkspaceID, func(ctx context.Context) (db.Workspace, error) {
		return db.Query.FindWorkspaceByID(ctx, s.db.RW(), authorizedWorkspaceID)
	}, caches.DefaultFindFirstOp)

	if db.IsNotFound(err) {
		return VerifyResponse{}, fault.New(
			"workspace not found",
			fault.WithCode(codes.Data.Workspace.NotFound.URN()),
			fault.WithDesc("workspace not found", "The requested workspace does not exist."),
		)
	}
	if err != nil {
		s.logger.Error("unable to load workspace",
			"error", err.Error())
		return VerifyResponse{}, fault.Wrap(
			err,
			fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
			fault.WithDesc("unable to load workspace", "We could not load the requested workspace."),
		)
	}

	if !ws.Enabled {
		return VerifyResponse{}, fault.New(
			"workspace is disabled",
			fault.WithCode(codes.Auth.Authorization.WorkspaceDisabled.URN()),
			fault.WithDesc("workspace disabled", "The workspace is disabled."),
		)
	}

	res := VerifyResponse{
		AuthorizedWorkspaceID: authorizedWorkspaceID,
		KeyID:                 key.ID,
	}
	// Root keys store the user's workspace id in `ForWorkspaceID` and we're
	// interested in the user, not our rootkey workspace.
	if key.ForWorkspaceID.Valid {
		res.AuthorizedWorkspaceID = key.ForWorkspaceID.String
	}

	return res, nil
}
