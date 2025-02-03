package keys

import (
	"context"
)

type KeyService interface {
	Verify(ctx context.Context, hash string) (VerifyResponse, error)
}

type VerifyResponse struct {
	AuthorizedWorkspaceID string
	KeyID                 string
}
