package keys

// Reason is the internal cause of a verification outcome. Callers only see the
// [KeyStatus]. Reason tells apart causes that share a status, such as the
// different ways a verification ends in NOT_FOUND. It must never be returned
// to callers, because some reasons reveal that a key exists.
//
// The values are the reason label of unkey_key_verifications_total; keep the
// list in internal/services/keys/metrics in sync.
type Reason string

const (
	ReasonNone                    Reason = "none"
	ReasonUnknownHash             Reason = "unknown_hash"
	ReasonMigrationNotFound       Reason = "migration_not_found"
	ReasonKeyDeleted              Reason = "key_deleted"
	ReasonAPIDeleted              Reason = "api_deleted"
	ReasonWorkspaceMismatch       Reason = "workspace_mismatch"
	ReasonKeyspaceMismatch        Reason = "keyspace_mismatch"
	ReasonPermissionDenied        Reason = "permission_denied"
	ReasonDisabled                Reason = "disabled"
	ReasonExpired                 Reason = "expired"
	ReasonIPDenied                Reason = "ip_denied"
	ReasonWorkspaceDisabled       Reason = "workspace_disabled"
	ReasonWorkspaceNotFound       Reason = "workspace_not_found"
	ReasonInsufficientPermissions Reason = "insufficient_permissions"
	ReasonRateLimited             Reason = "rate_limited"
	ReasonUsageExceeded           Reason = "usage_exceeded"
)
