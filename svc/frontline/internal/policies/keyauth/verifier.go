package keyauth

import (
	"context"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
)

// Verifier authenticates a raw credential using the policy's requirements.
// Implementations preserve Frontline faults for failures and may return rate-limit
// results alongside an error when enforcement completed before the failure.
type Verifier interface {
	Verify(ctx context.Context, sess *zen.Session, req VerifyRequest) (VerifyResult, error)
}

// VerifyRequest contains resolved policy options. Empty Keyspaces rejects every
// key; Credits is an explicit cost, including zero. RawKey must not be logged.
type VerifyRequest struct {
	RawKey          string
	AppID           string
	Keyspaces       []string
	Credits         int64
	PermissionQuery string
	Ratelimits      []openapi.KeysVerifyKeyRatelimit
}

// VerifyResult carries enforcement outcomes without exposing a database-backed
// KeyVerifier. Principal is populated only after successful verification.
type VerifyResult struct {
	Status     keys.KeyStatus
	Principal  *principal.Principal
	Ratelimits map[string]keys.RatelimitConfigAndResult
}
