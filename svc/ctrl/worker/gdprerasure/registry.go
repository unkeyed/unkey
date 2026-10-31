package gdprerasure

import (
	"context"
	"database/sql"
	"encoding/json"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// The registries below are the whole erasure policy: per subject, the ordered
// steps that remove or pseudonymize its data. MySQL statements are the sqlc
// queries in svc/ctrl/internal/db/queries/gdpr_erasure_*.sql, and ClickHouse
// statements are the erasure methods in pkg/clickhouse/gdpr_erasure.go.
// coverage_test.go fails when a schema table that can hold workspace data is
// neither erased here nor retained there with a reason.

// workspaceSteps erases a workspace. Compute stops first so no row vanishes
// under a running pod. The Stripe customer goes before workspace_billing, the
// row that names it, and the Depot projects go before projects, the rows that
// name them. Child tables without a workspace_id are found through their
// parents, so they run before them. The workspaces row goes last: until then a
// re-run can still read the WorkOS organization to delete.
var workspaceSteps = concat(
	[]step{
		teardownCompute{},
		deleteStripeCustomer{},
		deleteDepotProjects{},

		// Legacy root keys are root keys from before unkey_root_keys. They live
		// in the internal workspace and point at the workspace they grant
		// access to.
		mysqlDelete{table: "keys_permissions", by: "legacy root key", query: byWorkspace(db.Querier.EraseWorkspaceLegacyRootKeyPermissions)},
		mysqlDelete{table: "keys_roles", by: "legacy root key", query: byWorkspace(db.Querier.EraseWorkspaceLegacyRootKeyRoles)},
		mysqlDelete{table: "encrypted_keys", by: "legacy root key", query: byWorkspace(db.Querier.EraseWorkspaceLegacyRootKeyEncryptions)},
		mysqlDelete{table: "ratelimits", by: "legacy root key", query: byWorkspace(db.Querier.EraseWorkspaceLegacyRootKeyRatelimits)},
		mysqlDelete{table: "keys", by: "legacy root key", query: byWorkspace(db.Querier.EraseWorkspaceLegacyRootKeys)},

		deleteByWorkspace("connection_app_targets", db.Querier.EraseWorkspaceConnectionAppTargets),
		deleteByWorkspace("deployment_connection_app_targets", db.Querier.EraseWorkspaceDeploymentConnectionAppTargets),
		deleteByWorkspace("frontline_routes", db.Querier.EraseWorkspaceFrontlineRoutes),

		deleteByWorkspace("acme_challenges", db.Querier.EraseWorkspaceAcmeChallenges),
		deleteByWorkspace("acme_users", db.Querier.EraseWorkspaceAcmeUsers),
		deleteByWorkspace("apis", db.Querier.EraseWorkspaceApis),
		deleteByWorkspace("app_build_settings", db.Querier.EraseWorkspaceAppBuildSettings),
		deleteByWorkspace("app_connections", db.Querier.EraseWorkspaceAppConnections),
		deleteByWorkspace("app_environment_variables", db.Querier.EraseWorkspaceAppEnvironmentVariables),
		deleteByWorkspace("app_regional_settings", db.Querier.EraseWorkspaceAppRegionalSettings),
		deleteByWorkspace("app_runtime_settings", db.Querier.EraseWorkspaceAppRuntimeSettings),
		deleteByWorkspace("app_source_oci", db.Querier.EraseWorkspaceAppSourceOci),
		deleteByWorkspace("apps", db.Querier.EraseWorkspaceApps),
		deleteByWorkspace("billing_subscriptions", db.Querier.EraseWorkspaceBillingSubscriptions),
		deleteByWorkspace("certificates", db.Querier.EraseWorkspaceCertificates),
		deleteByWorkspace("cilium_network_policies", db.Querier.EraseWorkspaceCiliumNetworkPolicies),
		deleteByWorkspace("cli_sessions", db.Querier.EraseWorkspaceCliSessions),
		deleteByWorkspace("clickhouse_outbox", db.Querier.EraseWorkspaceClickhouseOutbox),
		deleteByWorkspace("clickhouse_workspace_settings", db.Querier.EraseWorkspaceClickhouseWorkspaceSettings),
		deleteByWorkspace("compute_signup_credit_claims", db.Querier.EraseWorkspaceComputeSignupCreditClaims),
		deleteByWorkspace("custom_domains", db.Querier.EraseWorkspaceCustomDomains),
		deleteByWorkspace("deployment_connections", db.Querier.EraseWorkspaceDeploymentConnections),
		deleteByWorkspace("deployment_steps", db.Querier.EraseWorkspaceDeploymentSteps),
		deleteByWorkspace("deployment_topology", db.Querier.EraseWorkspaceDeploymentTopology),
		deleteByWorkspace("deployments", db.Querier.EraseWorkspaceDeployments),
		deleteByWorkspace("device_authorizations", db.Querier.EraseWorkspaceDeviceAuthorizations),
		deleteByWorkspace("encrypted_keys", db.Querier.EraseWorkspaceEncryptedKeys),
		deleteByWorkspace("environments", db.Querier.EraseWorkspaceEnvironments),
		deleteByWorkspace("flag_overrides", db.Querier.EraseWorkspaceFlagOverrides),
		deleteByWorkspace("github_app_installations", db.Querier.EraseWorkspaceGithubAppInstallations),
		deleteByWorkspace("github_repo_connections", db.Querier.EraseWorkspaceGithubRepoConnections),
		deleteByWorkspace("horizontal_autoscaling_policies", db.Querier.EraseWorkspaceHorizontalAutoscalingPolicies),
		deleteByWorkspace("identities", db.Querier.EraseWorkspaceIdentities),
		deleteByWorkspace("instances", db.Querier.EraseWorkspaceInstances),
		deleteByWorkspace("key_auth", db.Querier.EraseWorkspaceKeyAuth),
		deleteByWorkspace("key_migrations", db.Querier.EraseWorkspaceKeyMigrations),
		deleteByWorkspace("keys", db.Querier.EraseWorkspaceKeys),
		deleteByWorkspace("keys_permissions", db.Querier.EraseWorkspaceKeysPermissions),
		deleteByWorkspace("keys_roles", db.Querier.EraseWorkspaceKeysRoles),
		deleteByWorkspace("limits", db.Querier.EraseWorkspaceLimits),
		deleteByWorkspace("logdrains", db.Querier.EraseWorkspaceLogdrains),
		deleteByWorkspace("openapi_specs", db.Querier.EraseWorkspaceOpenapiSpecs),
		deleteByWorkspace("permissions", db.Querier.EraseWorkspacePermissions),
		deleteByWorkspace("portal_domains", db.Querier.EraseWorkspacePortalDomains),
		deleteByWorkspace("portal_sessions", db.Querier.EraseWorkspacePortalSessions),
		deleteByWorkspace("portals", db.Querier.EraseWorkspacePortals),
		deleteByWorkspace("projects", db.Querier.EraseWorkspaceProjects),
		deleteByWorkspace("ratelimit_global_counters", db.Querier.EraseWorkspaceRatelimitGlobalCounters),
		deleteByWorkspace("ratelimit_namespaces", db.Querier.EraseWorkspaceRatelimitNamespaces),
		deleteByWorkspace("ratelimit_overrides", db.Querier.EraseWorkspaceRatelimitOverrides),
		deleteByWorkspace("ratelimits", db.Querier.EraseWorkspaceRatelimits),
		deleteByWorkspace("roles", db.Querier.EraseWorkspaceRoles),
		deleteByWorkspace("roles_permissions", db.Querier.EraseWorkspaceRolesPermissions),
		deleteByWorkspace("shared_secrets", db.Querier.EraseWorkspaceSharedSecrets),
		deleteByWorkspace("unkey_principal_permissions", db.Querier.EraseWorkspaceUnkeyPrincipalPermissions),
		deleteByWorkspace("unkey_root_keys", db.Querier.EraseWorkspaceUnkeyRootKeys),
		deleteByWorkspace("workspace_billing", db.Querier.EraseWorkspaceWorkspaceBilling),
	},
	clickhouseWorkspaceDeletes(),
	[]step{
		deleteWorkOSOrganization{},
		deleteByWorkspace("workspaces", db.Querier.EraseWorkspace),
	},
)

// endUserSteps erases a customer's end user: every identity with the external
// id, including soft-deleted ones, and every key linked to them. Keys go first
// so verifications stop arriving before ClickHouse is cleaned. Identities go
// last because every other step finds its rows through them.
var endUserSteps = concat(
	[]step{
		mysqlDelete{table: "keys_permissions", by: "", query: byEndUser(db.Querier.EraseEndUserKeyPermissions)},
		mysqlDelete{table: "keys_roles", by: "", query: byEndUser(db.Querier.EraseEndUserKeyRoles)},
		mysqlDelete{table: "encrypted_keys", by: "", query: byEndUser(db.Querier.EraseEndUserKeyEncryptions)},
		mysqlDelete{table: "ratelimits", by: "key", query: byEndUser(db.Querier.EraseEndUserKeyRatelimits)},
		mysqlDelete{table: "keys", by: "", query: byEndUser(db.Querier.EraseEndUserKeys)},
	},
	clickhouseEndUserDeletes(),
	[]step{
		mysqlDelete{table: "ratelimits", by: "identity", query: byEndUser(db.Querier.EraseEndUserIdentityRatelimits)},
		mysqlDelete{table: "portal_sessions", by: "", query: byEndUser(db.Querier.EraseEndUserPortalSessions)},
		mysqlDelete{table: "identities", by: "", query: byEndUser(db.Querier.EraseEndUserIdentities)},
	},
)

// userSteps erases a dashboard user. The Resend contact and Plain customer go
// first, so a key that cannot delete them stops the erasure before any row
// changes. Audit log rows stay, because they are the customer's security
// record, but lose the user's name, metadata, address and user agent. The
// outbox is rewritten first so a pending export cannot bring those back into
// ClickHouse.
var userSteps = []step{
	deleteResendContact{},
	deletePlainCustomer{},
	mysqlDelete{table: "cli_sessions", by: "", query: byUser(db.Querier.EraseUserCliSessions)},
	mysqlDelete{table: "device_authorizations", by: "", query: byUser(db.Querier.EraseUserDeviceAuthorizations)},
	mysqlDelete{table: "compute_signup_credit_claims", by: "", query: byUser(db.Querier.EraseUserComputeSignupCreditClaims)},
	mysqlUpdate{table: "deployments", by: "trigger actor", query: byUser(db.Querier.PseudonymizeUserDeploymentTriggers)},
	mysqlUpdate{table: "clickhouse_outbox", by: "audit actor", query: byUserJSON(db.Querier.PseudonymizeUserAuditOutbox)},
	pseudonymizeAuditLogs{},
	deleteWorkOSUser{},
}

func deleteByWorkspace[ID id](table string, query func(db.Querier, context.Context, ID) (int64, error)) step {
	return mysqlDelete{table: table, by: "", query: byWorkspace(query)}
}

func byWorkspace[ID id](query func(db.Querier, context.Context, ID) (int64, error)) mysqlQuery {
	return func(ctx context.Context, q db.Querier, s subject) (int64, error) {
		if s.workspaceID == "" {
			return 0, restate.TerminalErrorf("workspace id is empty for this subject")
		}
		return query(q, ctx, idOf[ID](s.workspaceID))
	}
}

func byUser[ID id](query func(db.Querier, context.Context, ID) (int64, error)) mysqlQuery {
	return func(ctx context.Context, q db.Querier, s subject) (int64, error) {
		if s.userID == "" {
			return 0, restate.TerminalErrorf("user id is empty for this subject")
		}
		return query(q, ctx, idOf[ID](s.userID))
	}
}

// byUserJSON passes the user id as a JSON string, for queries that compare it
// with a value inside a JSON column.
func byUserJSON(query func(db.Querier, context.Context, json.RawMessage) (int64, error)) mysqlQuery {
	return func(ctx context.Context, q db.Querier, s subject) (int64, error) {
		if s.userID == "" {
			return 0, restate.TerminalErrorf("user id is empty for this subject")
		}
		userID, err := json.Marshal(s.userID)
		if err != nil {
			return 0, restate.ToTerminalError(err)
		}
		return query(q, ctx, userID)
	}
}

// id is the type sqlc generates for an id argument: sql.NullString when the
// compared column is nullable, string otherwise.
type id interface {
	string | sql.NullString
}

func idOf[ID id](value string) ID {
	var out ID
	switch target := any(&out).(type) {
	case *string:
		*target = value
	case *sql.NullString:
		*target = sql.NullString{String: value, Valid: true}
	}
	return out
}

// endUserParams is the parameter struct sqlc generates for every end user
// query, so byEndUser accepts each of their distinct types.
type endUserParams interface {
	~struct {
		WorkspaceID string `db:"workspace_id"`
		ExternalID  string `db:"external_id"`
	}
}

func byEndUser[P endUserParams](query func(db.Querier, context.Context, P) (int64, error)) mysqlQuery {
	return func(ctx context.Context, q db.Querier, s subject) (int64, error) {
		if s.workspaceID == "" || s.externalID == "" {
			return 0, restate.TerminalErrorf("workspace id or external id is empty for this subject")
		}
		return query(q, ctx, P{WorkspaceID: s.workspaceID, ExternalID: s.externalID})
	}
}

func clickhouseWorkspaceDeletes() []step {
	var steps []step
	for _, table := range clickhouse.WorkspaceErasureTables() {
		steps = append(steps, clickhouseWorkspaceDelete{table: table})
	}
	return steps
}

func clickhouseEndUserDeletes() []step {
	var steps []step
	for _, table := range clickhouse.EndUserErasureTables() {
		steps = append(steps, clickhouseEndUserDelete{table: table})
	}
	return steps
}

func concat(groups ...[]step) []step {
	var steps []step
	for _, group := range groups {
		steps = append(steps, group...)
	}
	return steps
}
