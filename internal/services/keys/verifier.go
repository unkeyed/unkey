package keys

import (
	"context"
	"slices"
	"time"

	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/keys/metrics"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/auth/ipallowlist"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

// RatelimitConfigAndResult holds both the configuration and result for a rate limit
type RatelimitConfigAndResult struct {
	ID         string
	Cost       int64
	Name       string
	Duration   time.Duration
	Limit      int64
	AutoApply  bool
	Identifier string                       // The identifier to use for this rate limit
	Response   *ratelimit.RatelimitResponse // nil until rate limit is checked
}

// KeyVerifier represents a key that has been loaded from the database and is ready for verification.
// It contains all the necessary information and services to perform various validation checks.
type KeyVerifier struct {
	Key                   keysdb.FindKeyForVerificationRow // The key data from the database
	Roles                 []string                         // RBAC roles assigned to this key
	Permissions           []string                         // RBAC permissions assigned to this key
	Status                KeyStatus                        // The current validation status
	AuthorizedWorkspaceID string                           // The workspace ID this key is authorized for
	AllowedIPs            ipallowlist.Allowlist

	ratelimitConfigs map[string]keysdb.KeyFindForVerificationRatelimit // Rate limits configured for this key (name -> config)
	RatelimitResults map[string]RatelimitConfigAndResult               // Combined config and results for rate limits (name -> config+result)

	isRootKey bool // Whether this is a root key (special handling)

	message     string   // Internal message for validation failures
	reason      Reason   // Internal cause of the current Status
	cacheResult string   // How the verification cache answered the lookup; empty when Get did not run
	tags        []string // Tags associated with this verification

	session *zen.Session // The current request session
	region  string       // Geographic region identifier
	source  string       // Where the verification originated (schema.SourceAPI / SourceGateway)

	// Tracking
	startTime    time.Time // Time when verification started
	spentCredits int64     // Credits spent during verification

	// Services
	rateLimiter  ratelimit.Service    // Rate limiting service
	usageLimiter usagelimiter.Service // Usage limiting service
	rBAC         *rbac.RBAC           // Role-based access control service
}

// GetRatelimitConfigs returns the rate limit configurations
func (k *KeyVerifier) GetRatelimitConfigs() map[string]keysdb.KeyFindForVerificationRatelimit {
	return k.ratelimitConfigs
}

func (k *KeyVerifier) VerifyRootKey(ctx context.Context, opts ...VerifyOption) error {
	err := k.Verify(ctx, opts...)
	if err != nil {
		return err
	}

	return k.ToFault()
}

// Verify performs key verification with the given options.
// Authentication failures set Status; configuration and system failures return errors.
// A requested limit without a stored definition or complete inline override returns an error wrapping *RatelimitNotConfiguredError.
func (k *KeyVerifier) Verify(ctx context.Context, opts ...VerifyOption) (err error) {
	before := k.Status
	defer func() {
		if err == nil {
			k.recordVerifyOutcome(before)
		}
	}()

	// nolint:exhaustruct
	config := &verifyConfig{}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return err
		}
	}

	if k.Status == StatusValid && config.tags != nil {
		k.tags = config.tags
	}

	if config.keyspaces != nil && !slices.Contains(config.keyspaces, k.Key.KeyAuthID) {
		k.setInvalid(StatusNotFound, ReasonKeyspaceMismatch, "Key does not belong to an allowed keyspace.")
		return nil
	}

	if k.Status != StatusValid {
		return nil
	}

	if config.permissions != nil {
		err = k.withPermissions(ctx, *config.permissions)
		if err != nil {
			return err
		}
	}

	err = k.withRateLimits(ctx, config.ratelimits)
	if err != nil {
		return err
	}

	if config.credits != nil {
		err = k.withCredits(ctx, *config.credits)
		if err != nil {
			return err
		}
	}

	return nil
}

// recordVerifyOutcome records the terminal status of a verification that was
// still undecided when it entered Verify. Statuses Get already decided, and
// root keys, which never run through Verify, are recorded there instead.
// Verify skips it when it returns an error: the caller gets an error response,
// not a verification outcome, so there is no status to count.
func (k *KeyVerifier) recordVerifyOutcome(before KeyStatus) {
	if before != StatusValid || k.isRootKey {
		return
	}

	k.recordStatus()
}

func (k *KeyVerifier) recordStatus() {
	keyType := "key"
	if k.isRootKey {
		keyType = "root_key"
	}

	metrics.KeyVerificationsTotal.WithLabelValues(keyType, string(k.Status), string(k.reason)).Inc()
}

// Hide reports the key to the caller as NOT_FOUND because the caller may not
// see it, for example because it belongs to another workspace. A key that Get
// left undecided is recorded here, since it never reaches Verify.
func (k *KeyVerifier) Hide(reason Reason) {
	undecided := k.Status == StatusValid && !k.isRootKey
	k.setInvalid(StatusNotFound, reason, "key is hidden from the caller")
	if undecided {
		k.recordStatus()
	}
}

// FailureSnapshot captures a verification that did not end VALID for the
// internal key_verification_failures stream. Outcome is the code the caller
// received. The key ID fields are filled whenever the hash matched a key, even
// one the caller may not see. The key shape fields and SampleWeight are left
// for [FailureRecorder.Record].
func (k *KeyVerifier) FailureSnapshot(p *principal.Principal) schema.KeyVerificationFailure {
	rootKeyID := ""
	if p.Subject.Type == principal.SubjectTypeRootKey {
		rootKeyID = p.Subject.ID
	}

	return schema.KeyVerificationFailure{
		RequestID:      k.session.RequestID(),
		Time:           time.Now().UnixMilli(),
		WorkspaceID:    p.AuthorizedWorkspaceID,
		PrincipalType:  string(p.Type),
		RootKeyID:      rootKeyID,
		Region:         k.region,
		Source:         k.source,
		IPAddress:      k.session.Location(),
		UserAgent:      k.session.UserAgent(),
		Outcome:        string(k.ToOpenAPIStatus()),
		Reason:         string(k.reason),
		CacheResult:    k.cacheResult,
		KeyID:          k.Key.ID,
		KeySpaceID:     k.Key.KeyAuthID,
		KeyWorkspaceID: k.Key.WorkspaceID,
		KeyFingerprint: 0,
		KeyLength:      0,
		KeyPrefix:      "",
		KeyCharset:     "",
		SampleWeight:   0,
	}
}

// TelemetrySnapshot captures the final verification outcome for downstream
// telemetry sinks (currently the key_verifications ClickHouse stream). Callers
// should invoke this after Verify completes so that Verify-stage status
// mutations (FORBIDDEN, INSUFFICIENT_PERMISSIONS, RATE_LIMITED, USAGE_EXCEEDED)
// are reflected in the snapshot.
func (k *KeyVerifier) TelemetrySnapshot() schema.KeyVerification {
	workspaceID := ""
	if !k.isRootKey {
		workspaceID = k.Key.WorkspaceID
	}
	return schema.KeyVerification{
		RequestID:    k.session.RequestID(),
		WorkspaceID:  workspaceID,
		Time:         time.Now().UnixMilli(),
		Outcome:      string(k.Status),
		KeySpaceID:   k.Key.KeyAuthID,
		KeyID:        k.Key.ID,
		IdentityID:   k.Key.IdentityID.String,
		Tags:         k.tags,
		Region:       k.region,
		Source:       k.source,
		AppID:        "",
		ExternalID:   k.Key.ExternalID.String,
		SpentCredits: k.spentCredits,
		Latency:      float64(time.Since(k.startTime).Milliseconds()),
	}
}
