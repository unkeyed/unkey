package keys

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// VerifyRootKey validates a root API key from the Authorization header in a session.
// It extracts the bearer token from the session, verifies it using the Verify method,
// and updates the session with the authorized workspace ID.
//
// Root keys are special API keys that have different authorization rules than regular keys.
// They typically authorize access to a workspace specified in the ForWorkspaceID field.
//
// Parameters:
//   - ctx: Context for the operation, allowing for cancellation and timeout
//   - sess: The session containing authorization information
//
// Returns:
//   - VerifyResponse: Contains the authorized workspace ID and key ID
//   - error: Details about verification failure, if any occurred
//
// The session is modified by setting its WorkspaceID field to the authorized workspace ID
// if verification succeeds.
//
// Common error cases include:
//   - No bearer token in the Authorization header
//   - Invalid root key (not found, deleted, or disabled)
func (s *service) VerifyRootKey(ctx context.Context, sess *zen.Session) (VerifyResponse, error) {

	rootKey, err := zen.Bearer(sess)
	if err != nil {
		return VerifyResponse{}, fault.Wrap(err,
			fault.WithDesc(
				"no bearer",
				"You must provide a valid root key in the Authorization header in the format 'Bearer ROOT_KEY'",
			),
		)
	}

	res, err := s.Verify(ctx, rootKey)
	if err != nil {
		return VerifyResponse{}, fault.Wrap(err,
			fault.WithDesc("invalid root key", "The provided root key is invalid."))
	}
	sess.WorkspaceID = res.AuthorizedWorkspaceID

	return res, nil

}
