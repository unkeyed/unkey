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
			fault.WithTag(fault.UNAUTHORIZED),
			fault.WithDesc(
				"no bearer",
				"You must provide a valid root key in the Authorization header in the format 'Bearer <root_key>'",
			),
		)
	}

	res, err := s.Verify(ctx, rootKey)
	if err != nil {
		return VerifyResponse{}, fault.Wrap(err,
			fault.WithTag(fault.UNAUTHORIZED),
			fault.WithDesc("invalid root key", "The provided root key is invalid"))
	}

	return res, nil

}
