package keys

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/zen"
)

type KeyService interface {
	Verify(ctx context.Context, hash string) (VerifyResponse, error)
	VerifyRootKey(ctx context.Context, sess *zen.Session) (VerifyResponse, error)
}

type VerifyResponse struct {
	AuthorizedWorkspaceID string
	KeyID                 string
}
