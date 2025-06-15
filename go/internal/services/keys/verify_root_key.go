package keys

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func (s *service) VerifyRootKey(ctx context.Context, sess *zen.Session) (VerifyResponse, error) {

	rootKey, err := zen.Bearer(sess)
	if err != nil {
		return VerifyResponse{}, fault.Wrap(err,
			fault.Internal("no bearer"),
			fault.Public("You must provide a valid root key in the Authorization header in the format 'Bearer ROOT_KEY'."),
		)
	}

	res, err := s.Verify(ctx, rootKey)
	if err != nil {
		return VerifyResponse{}, fault.Wrap(err,
			fault.Internal("invalid root key"),
			fault.Public("The provided root key is invalid."))
	}
	sess.WorkspaceID = res.AuthorizedWorkspaceID

	return res, nil

}
