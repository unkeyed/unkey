package gdprerasure

// The registries below are the whole erasure policy: per subject, the ordered
// steps that remove or pseudonymize its data. Table names only ever come from
// here. coverage_test.go fails when a schema table that can hold workspace
// data is neither erased here nor retained there with a reason.

const (
	workspaceScope = "workspace_id = :workspace_id"

	endUserIdentities = "SELECT id FROM identities WHERE workspace_id = :workspace_id AND external_id = :external_id"
	endUserKeys       = "SELECT id FROM `keys` WHERE workspace_id = :workspace_id AND identity_id IN (" + endUserIdentities + ")"

	// legacyRootKeys are root keys from before unkey_root_keys. They live in
	// the internal workspace and point at the workspace they grant access to.
	legacyRootKeys = "SELECT id FROM `keys` WHERE for_workspace_id = :workspace_id"

	endUserVerifications = "workspace_id = :workspace_id AND (external_id = :external_id OR has(:identity_ids, identity_id))"

	auditActor = "actor_id = :user_id"
)

// workspaceSteps erases a workspace. Compute stops first so no row vanishes
// under a running pod. Child tables without a workspace_id are found through
// their parents, so they run before them. The workspaces row goes last: until
// then a re-run can still read the WorkOS organization to delete.
var workspaceSteps = concat(
	[]step{
		teardownCompute{},

		mysqlDelete{table: "keys_permissions", by: "legacy root key", where: "key_id IN (" + legacyRootKeys + ")"},
		mysqlDelete{table: "keys_roles", by: "legacy root key", where: "key_id IN (" + legacyRootKeys + ")"},
		mysqlDelete{table: "encrypted_keys", by: "legacy root key", where: "key_id IN (" + legacyRootKeys + ")"},
		mysqlDelete{table: "ratelimits", by: "legacy root key", where: "key_id IN (" + legacyRootKeys + ")"},
		mysqlDelete{table: "keys", by: "legacy root key", where: "for_workspace_id = :workspace_id"},

		mysqlDelete{table: "connection_app_targets", by: "", where: "connection_id IN (SELECT id FROM app_connections WHERE " + workspaceScope + ")"},
		mysqlDelete{table: "deployment_connection_app_targets", by: "", where: "deployment_id IN (SELECT id FROM deployments WHERE " + workspaceScope + ")"},
		mysqlDelete{table: "frontline_routes", by: "", where: "project_id IN (SELECT id FROM projects WHERE " + workspaceScope + ")"},
	},
	mysqlDeleteByWorkspace(
		"acme_challenges",
		"acme_users",
		"apis",
		"app_build_settings",
		"app_connections",
		"app_environment_variables",
		"app_regional_settings",
		"app_runtime_settings",
		"app_source_oci",
		"apps",
		"billing_subscriptions",
		"certificates",
		"cilium_network_policies",
		"cli_sessions",
		"clickhouse_outbox",
		"clickhouse_workspace_settings",
		"compute_signup_credit_claims",
		"custom_domains",
		"deployment_connections",
		"deployment_steps",
		"deployment_topology",
		"deployments",
		"device_authorizations",
		"encrypted_keys",
		"environments",
		"flag_overrides",
		"github_app_installations",
		"github_repo_connections",
		"horizontal_autoscaling_policies",
		"identities",
		"instances",
		"key_auth",
		"key_migrations",
		"keys",
		"keys_permissions",
		"keys_roles",
		"limits",
		"logdrains",
		"openapi_specs",
		"permissions",
		"portal_domains",
		"portal_sessions",
		"portals",
		"projects",
		"ratelimit_global_counters",
		"ratelimit_namespaces",
		"ratelimit_overrides",
		"ratelimits",
		"roles",
		"roles_permissions",
		"shared_secrets",
		"unkey_principal_permissions",
		"unkey_root_keys",
		"workspace_billing",
	),
	clickhouseDeleteByWorkspace(deleteMutation,
		"audit_logs_raw_v1",
		"frontline_requests_raw_v1",
		"key_verifications_raw_v2",
		"ratelimits_raw_v2",
		"runtime_logs_raw_v1",
	),
	clickhouseDeleteByWorkspace(deleteLightweight,
		"active_workspaces_per_month_v2",
		"api_requests_per_day_v2",
		"api_requests_per_hour_v2",
		"api_requests_per_minute_v2",
		"api_requests_per_month_v2",
		"api_requests_raw_v2",
		"billable_ratelimits_per_month_v2",
		"billable_verifications_per_month_v2",
		"build_step_logs_v1",
		"build_steps_v1",
		"frontline_requests_per_15m_v1",
		"frontline_requests_per_5m_v1",
		"frontline_requests_per_day_v1",
		"frontline_requests_per_hour_v1",
		"frontline_requests_per_minute_v1",
		"instance_checkpoints_v1",
		"instance_events_raw_v1",
		"instance_resources_per_15s_v1",
		"instance_resources_per_day_v1",
		"instance_resources_per_hour_v1",
		"instance_resources_per_minute_v1",
		"instance_resources_per_month_v1",
		"instance_usage_per_hour_v1",
		"key_last_used_v1",
		"key_verifications_per_day_v3",
		"key_verifications_per_hour_v3",
		"key_verifications_per_minute_v3",
		"key_verifications_per_month_v3",
		"logdrain_deliveries_raw_v1",
		"ratelimits_last_used_v2",
		"ratelimits_per_day_v2",
		"ratelimits_per_hour_v2",
		"ratelimits_per_minute_v2",
		"ratelimits_per_month_v2",
	),
	[]step{
		deleteWorkOSOrganization{},
		mysqlDelete{table: "workspaces", by: "", where: "id = :workspace_id"},
	},
)

// endUserSteps erases a customer's end user: every identity with the external
// id, including soft-deleted ones, and every key linked to them. Keys go first
// so verifications stop arriving before ClickHouse is cleaned. Identities go
// last because every other step finds its rows through them.
var endUserSteps = []step{
	mysqlDelete{table: "keys_permissions", by: "", where: "workspace_id = :workspace_id AND key_id IN (" + endUserKeys + ")"},
	mysqlDelete{table: "keys_roles", by: "", where: "workspace_id = :workspace_id AND key_id IN (" + endUserKeys + ")"},
	mysqlDelete{table: "encrypted_keys", by: "", where: "workspace_id = :workspace_id AND key_id IN (" + endUserKeys + ")"},
	mysqlDelete{table: "ratelimits", by: "key", where: "workspace_id = :workspace_id AND key_id IN (" + endUserKeys + ")"},
	mysqlDelete{table: "keys", by: "", where: "workspace_id = :workspace_id AND identity_id IN (" + endUserIdentities + ")"},

	clickhouseDelete{table: "key_verifications_raw_v2", where: endUserVerifications, mode: deleteMutation},
	clickhouseDelete{table: "key_verifications_per_minute_v3", where: endUserVerifications, mode: deleteLightweight},
	clickhouseDelete{table: "key_verifications_per_hour_v3", where: endUserVerifications, mode: deleteLightweight},
	clickhouseDelete{table: "key_verifications_per_day_v3", where: endUserVerifications, mode: deleteLightweight},
	clickhouseDelete{table: "key_verifications_per_month_v3", where: endUserVerifications, mode: deleteLightweight},
	clickhouseDelete{table: "key_last_used_v1", where: "workspace_id = :workspace_id AND has(:identity_ids, identity_id)", mode: deleteLightweight},

	mysqlDelete{table: "ratelimits", by: "identity", where: "workspace_id = :workspace_id AND identity_id IN (" + endUserIdentities + ")"},
	mysqlDelete{table: "portal_sessions", by: "", where: "workspace_id = :workspace_id AND external_id = :external_id"},
	mysqlDelete{table: "identities", by: "", where: "workspace_id = :workspace_id AND external_id = :external_id"},
}

// userSteps erases a dashboard user. Audit log rows stay, because they are the
// customer's security record, but lose the user's name, metadata, address and
// user agent. The outbox is rewritten first so a pending export cannot bring
// those back into ClickHouse.
var userSteps = []step{
	mysqlDelete{table: "cli_sessions", by: "", where: "user_id = :user_id"},
	mysqlDelete{table: "device_authorizations", by: "", where: "approver_user_id = :user_id"},
	mysqlDelete{table: "compute_signup_credit_claims", by: "", where: "workos_user_id = :user_id"},
	mysqlUpdate{
		table: "clickhouse_outbox",
		by:    "audit actor",
		set:   "payload = JSON_REMOVE(payload, '$.actor.name', '$.actor.meta', '$.remote_ip', '$.user_agent')",
		where: "version = 'audit_log.v1' AND JSON_UNQUOTE(JSON_EXTRACT(payload, '$.actor.id')) = :user_id" +
			" AND JSON_CONTAINS_PATH(payload, 'one', '$.actor.name', '$.actor.meta', '$.remote_ip', '$.user_agent')",
	},
	clickhouseUpdate{
		table: "audit_logs_raw_v1",
		set:   "actor_name = '', actor_meta = '{}', remote_ip = '', user_agent = ''",
		where: auditActor + " AND (actor_name != '' OR remote_ip != '' OR user_agent != '' OR toJSONString(actor_meta) != '{}')",
	},
	deleteWorkOSUser{},
}

func mysqlDeleteByWorkspace(tables ...string) []step {
	steps := make([]step, 0, len(tables))
	for _, table := range tables {
		steps = append(steps, mysqlDelete{table: table, by: "", where: workspaceScope})
	}
	return steps
}

func clickhouseDeleteByWorkspace(mode clickhouseDeleteMode, tables ...string) []step {
	steps := make([]step, 0, len(tables))
	for _, table := range tables {
		steps = append(steps, clickhouseDelete{table: table, where: workspaceScope, mode: mode})
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
