package keyauth

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
)

type internalVerifier struct {
	keyService       keys.KeyService
	keyVerifications *batch.BatchProcessor[schema.KeyVerification]
}

var _ Verifier = (*internalVerifier)(nil)

// NewInternalVerifier uses the internal key service and records one telemetry
// snapshot for each successful lookup, including requests rejected afterward.
func NewInternalVerifier(keyService keys.KeyService, keyVerifications *batch.BatchProcessor[schema.KeyVerification]) (Verifier, error) {
	if err := assert.All(
		assert.NotNil(keyService, "keyService must not be nil"),
		assert.NotNilAndNotZero(keyVerifications, "keyVerifications must not be nil"),
	); err != nil {
		return nil, err
	}
	return &internalVerifier{keyService: keyService, keyVerifications: keyVerifications}, nil
}

func (v *internalVerifier) Verify(ctx context.Context, sess *zen.Session, req VerifyRequest) (VerifyResult, error) {
	var result VerifyResult
	verifier, err := v.keyService.Get(ctx, sess, hash.Sha256(req.RawKey))
	if err != nil {
		return result, fault.Wrap(err,
			fault.Code(codes.Frontline.Auth.InvalidKey.URN()),
			fault.Internal("key lookup failed"),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}
	defer func() {
		verification := verifier.TelemetrySnapshot()
		verification.AppID = req.AppID
		v.keyVerifications.Buffer(verification)
	}()

	if verifier.Status != keys.StatusValid {
		return result, fault.New("invalid API key",
			fault.Code(codes.Frontline.Auth.InvalidKey.URN()),
			fault.Internal("key status: "+string(verifier.Status)),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}

	if req.Credits < 0 {
		return result, fault.New("negative credits cost in keyauth policy",
			fault.Code(codes.Frontline.Internal.InvalidConfiguration.URN()),
			fault.Internal(fmt.Sprintf("negative credits cost: %d", req.Credits)),
			fault.Public("Service configuration error."),
		)
	}
	verifyOpts := []keys.VerifyOption{
		keys.WithKeyspaces(req.Keyspaces...),
		keys.WithCredits(req.Credits),
	}
	if req.PermissionQuery != "" {
		query, err := rbac.ParseQuery(req.PermissionQuery)
		if err != nil {
			return result, fault.Wrap(err,
				fault.Code(codes.Frontline.Internal.InvalidConfiguration.URN()),
				fault.Internal("invalid permission query: "+req.PermissionQuery),
				fault.Public("Service configuration error."),
			)
		}
		verifyOpts = append(verifyOpts, keys.WithPermissions(query))
	}
	if len(req.Ratelimits) > 0 {
		verifyOpts = append(verifyOpts, keys.WithRateLimits(req.Ratelimits))
	}

	if err := verifier.Verify(ctx, verifyOpts...); err != nil {
		return result, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("verification error"),
			fault.Public("An internal error occurred during authentication."),
		)
	}

	result.Status = verifier.Status
	result.Ratelimits = verifier.RatelimitResults
	if verifier.Status != keys.StatusValid {
		return result, nil
	}

	result.Principal, err = principal.KeyPrincipalFromVerifier(verifier)
	if err != nil {
		return result, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to build principal"),
			fault.Public("An internal error occurred during authentication."),
		)
	}
	return result, nil
}
