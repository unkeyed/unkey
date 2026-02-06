---
title: db
description: "provides a database access layer for MySQL with support for"
---

Package db provides a database access layer for MySQL with support for read/write splitting, connection management, and type-safe SQL operations.

The package uses sqlc ([https://sqlc.dev/](https://sqlc.dev/)) to generate type-safe Go code from SQL queries, providing compile-time verification of database operations.

Key features:

\- Primary/replica configuration with automatic routing of reads and writes - Support for transactions - Type-safe query methods generated from SQL - Bulk insert operations for improved performance - BulkQuerier interface for type-safe bulk operations

Basic usage:

	// Initialize the database with primary and optional read replica
	db, err := db.New(db.Config{
	    PrimaryDSN:  "mysql://user:pass@primary:3306/dbname?parseTime=true",
	    ReadOnlyDSN: "mysql://user:pass@replica:3306/dbname?parseTime=true",
	})
	if err != nil {
	    return fmt.Errorf("database initialization failed: %w", err)
	}
	defer db.Close()

	// Execute a read query using the read replica
	workspace, err := db.Query.FindWorkspaceByID(ctx, db.RO(), workspaceID)
	if err != nil {
	    return fmt.Errorf("failed to find workspace: %w", err)
	}

	// Execute a write query using the read/write connection
	err = db.Query.InsertKey(ctx, db.RW(), insertKeyParams)
	if err != nil {
	    return fmt.Errorf("failed to insert key: %w", err)
	}

	// Use bulk operations for efficient batch inserts
	err = db.Query.BulkInsertKey(ctx, db.RW(), []db.InsertKeyParams{
	    insertKeyParams1,
	    insertKeyParams2,
	    insertKeyParams3,
	})
	if err != nil {
	    return fmt.Errorf("failed to bulk insert keys: %w", err)
	}

	// Type-safe bulk operations using the BulkQuerier interface
	var bulkQuerier db.BulkQuerier = db.Query
	err = bulkQuerier.BulkInsertKey(ctx, db.RW(), keyBatch)
	if err != nil {
	    return fmt.Errorf("failed to bulk insert via interface: %w", err)
	}

This package relies on the standard Go database/sql package and the go-sql-driver/mysql driver for the actual database communication.

Package db provides database transaction utilities for the Unkey platform. It offers transaction lifecycle management with automatic rollback on errors and proper error wrapping for consistent fault handling across services.

The package is shared across all Unkey services including the API service for atomic key operations, admin service for workspace management, and audit service for logging operations.

## Constants

MySQL server error codes See: [https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html](https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html)
```go
const (
	// errDeadlock is returned when a deadlock is detected.
	// The entire transaction should be retried.
	errDeadlock = 1213

	// errLockWaitTimeout is returned when a lock wait timeout is exceeded.
	// By default, only the statement is rolled back (not the entire transaction).
	// The statement should be retried.
	errLockWaitTimeout = 1205

	// errTooManyConnections is returned when the server has too many connections.
	// This is a transient error that may resolve when connections are freed.
	errTooManyConnections = 1040
)
```

MySQL client error codes (CR\_\* errors) See: [https://dev.mysql.com/doc/mysql-errors/8.0/en/client-error-reference.html](https://dev.mysql.com/doc/mysql-errors/8.0/en/client-error-reference.html)
```go
const (
	// errServerGone is returned when the MySQL server has gone away.
	// This can happen due to connection timeout or server restart.
	errServerGone = 2006

	// errServerLost is returned when the connection to the server was lost during a query.
	// This can happen due to network issues or server crash.
	errServerLost = 2013
)
```

```go
const (
	// DefaultBackoff is the base duration for exponential backoff in database retries
	DefaultBackoff = 50 * time.Millisecond
	// DefaultAttempts is the maximum number of retry attempts for database operations
	DefaultAttempts = 3
)
```

```go
const (
	statusSuccess = "success"
	statusError   = "error"
)
```

bulkInsertAcmeChallenge is the base query for bulk insert
```go
const bulkInsertAcmeChallenge = `INSERT INTO acme_challenges ( workspace_id, domain_id, token, authorization, status, challenge_type, created_at, updated_at, expires_at ) VALUES %s`
```

bulkInsertAcmeUser is the base query for bulk insert
```go
const bulkInsertAcmeUser = `INSERT INTO acme_users (id, workspace_id, encrypted_key, created_at) VALUES %s`
```

bulkInsertApi is the base query for bulk insert
```go
const bulkInsertApi = `INSERT INTO apis ( id, name, workspace_id, auth_type, ip_whitelist, key_auth_id, created_at_m, deleted_at_m ) VALUES %s`
```

bulkInsertAuditLog is the base query for bulk insert
```go
const bulkInsertAuditLog = `INSERT INTO ` + "`" + `audit_log` + "`" + ` ( id, workspace_id, bucket_id, bucket, event, time, display, remote_ip, user_agent, actor_type, actor_id, actor_name, actor_meta, created_at ) VALUES %s`
```

bulkInsertAuditLogTarget is the base query for bulk insert
```go
const bulkInsertAuditLogTarget = `INSERT INTO ` + "`" + `audit_log_target` + "`" + ` ( workspace_id, bucket_id, bucket, audit_log_id, display_name, type, id, name, meta, created_at ) VALUES %s`
```

bulkInsertCertificate is the base query for bulk insert
```go
const bulkInsertCertificate = `INSERT INTO certificates (id, workspace_id, hostname, certificate, encrypted_private_key, created_at) VALUES %s ON DUPLICATE KEY UPDATE
workspace_id = VALUES(workspace_id),
hostname = VALUES(hostname),
certificate = VALUES(certificate),
encrypted_private_key = VALUES(encrypted_private_key),
updated_at = ?`
```

bulkInsertCiliumNetworkPolicy is the base query for bulk insert
```go
const bulkInsertCiliumNetworkPolicy = `INSERT INTO cilium_network_policies ( id, workspace_id, project_id, environment_id, k8s_name, region, policy, version, created_at ) VALUES %s`
```

bulkInsertClickhouseWorkspaceSettings is the base query for bulk insert
```go
const bulkInsertClickhouseWorkspaceSettings = `INSERT INTO ` + "`" + `clickhouse_workspace_settings` + "`" + ` ( workspace_id, username, password_encrypted, quota_duration_seconds, max_queries_per_window, max_execution_time_per_window, max_query_execution_time, max_query_memory_bytes, max_query_result_rows, created_at, updated_at ) VALUES %s`
```

bulkInsertCustomDomain is the base query for bulk insert
```go
const bulkInsertCustomDomain = `INSERT INTO custom_domains ( id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, target_cname, invocation_id, created_at ) VALUES %s`
```

bulkInsertDeployment is the base query for bulk insert
```go
const bulkInsertDeployment = `INSERT INTO ` + "`" + `deployments` + "`" + ` ( id, k8s_name, workspace_id, project_id, environment_id, git_commit_sha, git_branch, sentinel_config, git_commit_message, git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp, openapi_spec, encrypted_environment_variables, command, status, cpu_millicores, memory_mib, created_at, updated_at ) VALUES %s`
```

bulkInsertDeploymentTopology is the base query for bulk insert
```go
const bulkInsertDeploymentTopology = `INSERT INTO ` + "`" + `deployment_topology` + "`" + ` ( workspace_id, deployment_id, region, desired_replicas, desired_status, version, created_at ) VALUES %s`
```

bulkInsertEnvironment is the base query for bulk insert
```go
const bulkInsertEnvironment = `INSERT INTO environments ( id, workspace_id, project_id, slug, description, created_at, updated_at, sentinel_config ) VALUES %s`
```

bulkInsertFrontlineRoute is the base query for bulk insert
```go
const bulkInsertFrontlineRoute = `INSERT INTO frontline_routes ( id, project_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at, updated_at ) VALUES %s`
```

bulkInsertGithubRepoConnection is the base query for bulk insert
```go
const bulkInsertGithubRepoConnection = `INSERT INTO github_repo_connections ( project_id, installation_id, repository_id, repository_full_name, created_at, updated_at ) VALUES %s`
```

bulkInsertIdentity is the base query for bulk insert
```go
const bulkInsertIdentity = `INSERT INTO ` + "`" + `identities` + "`" + ` ( id, external_id, workspace_id, environment, created_at, meta ) VALUES %s`
```

bulkInsertIdentityRatelimit is the base query for bulk insert
```go
const bulkInsertIdentityRatelimit = `INSERT INTO ` + "`" + `ratelimits` + "`" + ` ( id, workspace_id, identity_id, name, ` + "`" + `limit` + "`" + `, duration, created_at, auto_apply ) VALUES %s ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    ` + "`" + `limit` + "`" + ` = VALUES(` + "`" + `limit` + "`" + `),
    duration = VALUES(duration),
    auto_apply = VALUES(auto_apply),
    updated_at = VALUES(created_at)`
```

bulkInsertKey is the base query for bulk insert
```go
const bulkInsertKey = `INSERT INTO ` + "`" + `keys` + "`" + ` ( id, key_auth_id, hash, start, workspace_id, for_workspace_id, name, owner_id, identity_id, meta, expires, created_at_m, enabled, remaining_requests, refill_day, refill_amount, pending_migration_id ) VALUES %s`
```

bulkInsertKeyAuth is the base query for bulk insert
```go
const bulkInsertKeyAuth = `INSERT INTO key_auth ( id, workspace_id, created_at_m, default_prefix, default_bytes, store_encrypted_keys ) VALUES %s`
```

bulkInsertKeyEncryption is the base query for bulk insert
```go
const bulkInsertKeyEncryption = `INSERT INTO encrypted_keys (workspace_id, key_id, encrypted, encryption_key_id, created_at) VALUES %s`
```

bulkInsertKeyMigration is the base query for bulk insert
```go
const bulkInsertKeyMigration = `INSERT INTO key_migrations ( id, workspace_id, algorithm ) VALUES %s`
```

bulkInsertKeyPermission is the base query for bulk insert
```go
const bulkInsertKeyPermission = `INSERT INTO ` + "`" + `keys_permissions` + "`" + ` ( key_id, permission_id, workspace_id, created_at_m ) VALUES %s ON DUPLICATE KEY UPDATE updated_at_m = ?`
```

bulkInsertKeyRatelimit is the base query for bulk insert
```go
const bulkInsertKeyRatelimit = `INSERT INTO ` + "`" + `ratelimits` + "`" + ` ( id, workspace_id, key_id, name, ` + "`" + `limit` + "`" + `, duration, auto_apply, created_at ) VALUES %s ON DUPLICATE KEY UPDATE
` + "`" + `limit` + "`" + ` = VALUES(` + "`" + `limit` + "`" + `),
duration = VALUES(duration),
auto_apply = VALUES(auto_apply),
updated_at = ?`
```

bulkInsertKeyRole is the base query for bulk insert
```go
const bulkInsertKeyRole = `INSERT INTO keys_roles ( key_id, role_id, workspace_id, created_at_m ) VALUES %s`
```

bulkInsertKeySpace is the base query for bulk insert
```go
const bulkInsertKeySpace = `INSERT INTO ` + "`" + `key_auth` + "`" + ` ( id, workspace_id, created_at_m, store_encrypted_keys, default_prefix, default_bytes, size_approx, size_last_updated_at ) VALUES %s`
```

bulkInsertPermission is the base query for bulk insert
```go
const bulkInsertPermission = `INSERT INTO permissions ( id, workspace_id, name, slug, description, created_at_m ) VALUES %s`
```

bulkInsertProject is the base query for bulk insert
```go
const bulkInsertProject = `INSERT INTO projects ( id, workspace_id, name, slug, git_repository_url, default_branch, delete_protection, created_at, updated_at ) VALUES %s`
```

bulkInsertRatelimitNamespace is the base query for bulk insert
```go
const bulkInsertRatelimitNamespace = `INSERT INTO ` + "`" + `ratelimit_namespaces` + "`" + ` ( id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m ) VALUES %s`
```

bulkInsertRatelimitOverride is the base query for bulk insert
```go
const bulkInsertRatelimitOverride = `INSERT INTO ratelimit_overrides ( id, workspace_id, namespace_id, identifier, ` + "`" + `limit` + "`" + `, duration, async, created_at_m ) VALUES %s ON DUPLICATE KEY UPDATE
    ` + "`" + `limit` + "`" + ` = VALUES(` + "`" + `limit` + "`" + `),
    duration = VALUES(duration),
    async = VALUES(async),
    updated_at_m = ?`
```

bulkInsertRole is the base query for bulk insert
```go
const bulkInsertRole = `INSERT INTO roles ( id, workspace_id, name, description, created_at_m ) VALUES %s`
```

bulkInsertRolePermission is the base query for bulk insert
```go
const bulkInsertRolePermission = `INSERT INTO roles_permissions ( role_id, permission_id, workspace_id, created_at_m ) VALUES %s`
```

bulkInsertSentinel is the base query for bulk insert
```go
const bulkInsertSentinel = `INSERT INTO sentinels ( id, workspace_id, environment_id, project_id, k8s_address, k8s_name, region, image, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at ) VALUES %s`
```

bulkInsertWorkspace is the base query for bulk insert
```go
const bulkInsertWorkspace = `INSERT INTO ` + "`" + `workspaces` + "`" + ` ( id, org_id, name, slug, created_at_m, tier, beta_features, features, enabled, delete_protection ) VALUES %s`
```

bulkUpsertCustomDomain is the base query for bulk insert
```go
const bulkUpsertCustomDomain = `INSERT INTO custom_domains ( id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, target_cname, created_at ) VALUES %s ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    project_id = VALUES(project_id),
    environment_id = VALUES(environment_id),
    challenge_type = VALUES(challenge_type),
    verification_status = VALUES(verification_status),
    target_cname = VALUES(target_cname),
    updated_at = ?`
```

bulkUpsertEnvironment is the base query for bulk insert
```go
const bulkUpsertEnvironment = `INSERT INTO environments ( id, workspace_id, project_id, slug, sentinel_config, created_at ) VALUES %s ON DUPLICATE KEY UPDATE slug = VALUES(slug)`
```

bulkUpsertIdentity is the base query for bulk insert
```go
const bulkUpsertIdentity = `INSERT INTO ` + "`" + `identities` + "`" + ` ( id, external_id, workspace_id, environment, created_at, meta ) VALUES %s ON DUPLICATE KEY UPDATE external_id = external_id`
```

bulkUpsertInstance is the base query for bulk insert
```go
const bulkUpsertInstance = `INSERT INTO instances ( id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status ) VALUES %s ON DUPLICATE KEY UPDATE
	address = ?,
	cpu_millicores = ?,
	memory_mib = ?,
	status = ?`
```

bulkUpsertKeySpace is the base query for bulk insert
```go
const bulkUpsertKeySpace = `INSERT INTO key_auth ( id, workspace_id, created_at_m, default_prefix, default_bytes, store_encrypted_keys ) VALUES %s ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    store_encrypted_keys = VALUES(store_encrypted_keys)`
```

bulkUpsertQuota is the base query for bulk insert
```go
const bulkUpsertQuota = `INSERT INTO quota ( workspace_id, requests_per_month, audit_logs_retention_days, logs_retention_days, team ) VALUES %s ON DUPLICATE KEY UPDATE
    requests_per_month = VALUES(requests_per_month),
    audit_logs_retention_days = VALUES(audit_logs_retention_days),
    logs_retention_days = VALUES(logs_retention_days)`
```

bulkUpsertWorkspace is the base query for bulk insert
```go
const bulkUpsertWorkspace = `INSERT INTO workspaces ( id, org_id, name, slug, created_at_m, tier, beta_features, features, enabled, delete_protection ) VALUES %s ON DUPLICATE KEY UPDATE
    beta_features = VALUES(beta_features),
    name = VALUES(name)`
```

```go
const clearAcmeChallengeTokens = `-- name: ClearAcmeChallengeTokens :exec
UPDATE acme_challenges
SET token = ?, authorization = ?, updated_at = ?
WHERE domain_id = ?
`
```

```go
const deleteAcmeChallengeByDomainID = `-- name: DeleteAcmeChallengeByDomainID :exec
DELETE FROM acme_challenges WHERE domain_id = ?
`
```

```go
const deleteAllKeyPermissionsByKeyID = `-- name: DeleteAllKeyPermissionsByKeyID :exec
DELETE FROM keys_permissions
WHERE key_id = ?
`
```

```go
const deleteAllKeyRolesByKeyID = `-- name: DeleteAllKeyRolesByKeyID :exec
DELETE FROM keys_roles
WHERE key_id = ?
`
```

```go
const deleteCustomDomainByID = `-- name: DeleteCustomDomainByID :exec
DELETE FROM custom_domains WHERE id = ?
`
```

```go
const deleteDeploymentInstances = `-- name: DeleteDeploymentInstances :exec
DELETE FROM instances
WHERE deployment_id = ? AND region = ?
`
```

```go
const deleteFrontlineRouteByFQDN = `-- name: DeleteFrontlineRouteByFQDN :exec
DELETE FROM frontline_routes WHERE fully_qualified_domain_name = ?
`
```

```go
const deleteIdentity = `-- name: DeleteIdentity :exec
DELETE FROM identities
WHERE id = ?
  AND workspace_id = ?
`
```

```go
const deleteInstance = `-- name: DeleteInstance :exec
DELETE FROM instances WHERE k8s_name = ? AND region = ?
`
```

```go
const deleteKeyByID = `-- name: DeleteKeyByID :exec
DELETE k, kp, kr, rl, ek
FROM ` + "`" + `keys` + "`" + ` k
LEFT JOIN keys_permissions kp ON k.id = kp.key_id
LEFT JOIN keys_roles kr ON k.id = kr.key_id
LEFT JOIN ratelimits rl ON k.id = rl.key_id
LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
WHERE k.id = ?
`
```

```go
const deleteKeyPermissionByKeyAndPermissionID = `-- name: DeleteKeyPermissionByKeyAndPermissionID :exec
DELETE FROM keys_permissions
WHERE key_id = ? AND permission_id = ?
`
```

```go
const deleteManyKeyPermissionByKeyAndPermissionIDs = `-- name: DeleteManyKeyPermissionByKeyAndPermissionIDs :exec
DELETE FROM keys_permissions
WHERE key_id = ? AND permission_id IN (/*SLICE:ids*/?)
`
```

```go
const deleteManyKeyPermissionsByPermissionID = `-- name: DeleteManyKeyPermissionsByPermissionID :exec
DELETE FROM keys_permissions
WHERE permission_id = ?
`
```

```go
const deleteManyKeyRolesByKeyAndRoleIDs = `-- name: DeleteManyKeyRolesByKeyAndRoleIDs :exec
DELETE FROM keys_roles
WHERE key_id = ? AND role_id IN(/*SLICE:role_ids*/?)
`
```

```go
const deleteManyKeyRolesByKeyID = `-- name: DeleteManyKeyRolesByKeyID :exec
DELETE FROM keys_roles
WHERE key_id = ? AND role_id = ?
`
```

```go
const deleteManyKeyRolesByRoleID = `-- name: DeleteManyKeyRolesByRoleID :exec
DELETE FROM keys_roles
WHERE role_id = ?
`
```

```go
const deleteManyRatelimitsByIDs = `-- name: DeleteManyRatelimitsByIDs :exec
DELETE FROM ratelimits WHERE id IN (/*SLICE:ids*/?)
`
```

```go
const deleteManyRatelimitsByIdentityID = `-- name: DeleteManyRatelimitsByIdentityID :exec
DELETE FROM ratelimits WHERE identity_id = ?
`
```

```go
const deleteManyRolePermissionsByPermissionID = `-- name: DeleteManyRolePermissionsByPermissionID :exec
DELETE FROM roles_permissions
WHERE permission_id = ?
`
```

```go
const deleteManyRolePermissionsByRoleID = `-- name: DeleteManyRolePermissionsByRoleID :exec
DELETE FROM roles_permissions
WHERE role_id = ?
`
```

```go
const deleteOldIdentityByExternalID = `-- name: DeleteOldIdentityByExternalID :exec
DELETE i, rl
FROM identities i
LEFT JOIN ratelimits rl ON rl.identity_id = i.id
WHERE i.workspace_id = ?
  AND i.external_id = ?
  AND i.id != ?
  AND i.deleted = true
`
```

```go
const deleteOldIdentityWithRatelimits = `-- name: DeleteOldIdentityWithRatelimits :exec
DELETE i, rl
FROM identities i
LEFT JOIN ratelimits rl ON rl.identity_id = i.id
WHERE i.workspace_id = ?
  AND (i.id = ? OR i.external_id = ?)
  AND i.deleted = true
`
```

```go
const deletePermission = `-- name: DeletePermission :exec
DELETE FROM permissions
WHERE id = ?
`
```

```go
const deleteRatelimit = `-- name: DeleteRatelimit :exec
DELETE FROM ` + "`" + `ratelimits` + "`" + ` WHERE id = ?
`
```

```go
const deleteRatelimitNamespace = `-- name: DeleteRatelimitNamespace :execresult
UPDATE ` + "`" + `ratelimit_namespaces` + "`" + `
SET deleted_at_m = ?
WHERE id = ?
`
```

```go
const deleteRoleByID = `-- name: DeleteRoleByID :exec
DELETE FROM roles
WHERE id = ?
`
```

```go
const findAcmeChallengeByToken = `-- name: FindAcmeChallengeByToken :one
SELECT pk, domain_id, workspace_id, token, challenge_type, authorization, status, expires_at, created_at, updated_at FROM acme_challenges WHERE workspace_id = ? AND domain_id = ? AND token = ?
`
```

```go
const findAcmeUserByWorkspaceID = `-- name: FindAcmeUserByWorkspaceID :one
SELECT pk, id, workspace_id, encrypted_key, registration_uri, created_at, updated_at FROM acme_users WHERE workspace_id = ? LIMIT 1
`
```

```go
const findApiByID = `-- name: FindApiByID :one
SELECT pk, id, name, workspace_id, ip_whitelist, auth_type, key_auth_id, created_at_m, updated_at_m, deleted_at_m, delete_protection FROM apis WHERE id = ?
`
```

```go
const findAuditLogTargetByID = `-- name: FindAuditLogTargetByID :many
SELECT audit_log_target.pk, audit_log_target.workspace_id, audit_log_target.bucket_id, audit_log_target.bucket, audit_log_target.audit_log_id, audit_log_target.display_name, audit_log_target.type, audit_log_target.id, audit_log_target.name, audit_log_target.meta, audit_log_target.created_at, audit_log_target.updated_at, audit_log.pk, audit_log.id, audit_log.workspace_id, audit_log.bucket, audit_log.bucket_id, audit_log.event, audit_log.time, audit_log.display, audit_log.remote_ip, audit_log.user_agent, audit_log.actor_type, audit_log.actor_id, audit_log.actor_name, audit_log.actor_meta, audit_log.created_at, audit_log.updated_at
FROM audit_log_target
JOIN audit_log ON audit_log.id = audit_log_target.audit_log_id
WHERE audit_log_target.id = ?
`
```

```go
const findCertificateByHostname = `-- name: FindCertificateByHostname :one
SELECT pk, id, workspace_id, hostname, certificate, encrypted_private_key, created_at, updated_at FROM certificates WHERE hostname = ?
`
```

```go
const findCertificatesByHostnames = `-- name: FindCertificatesByHostnames :many
SELECT pk, id, workspace_id, hostname, certificate, encrypted_private_key, created_at, updated_at FROM certificates WHERE hostname IN (/*SLICE:hostnames*/?)
`
```

```go
const findCiliumNetworkPoliciesByEnvironmentID = `-- name: FindCiliumNetworkPoliciesByEnvironmentID :many
SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, region, policy, version, created_at, updated_at FROM cilium_network_policies WHERE environment_id = ?
`
```

```go
const findCiliumNetworkPolicyByIDAndRegion = `-- name: FindCiliumNetworkPolicyByIDAndRegion :one
SELECT
    n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
    w.k8s_namespace
FROM ` + "`" + `cilium_network_policies` + "`" + ` n
JOIN ` + "`" + `workspaces` + "`" + ` w ON w.id = n.workspace_id
WHERE n.region = ? AND n.id = ?
LIMIT 1
`
```

```go
const findClickhouseWorkspaceSettingsByWorkspaceID = `-- name: FindClickhouseWorkspaceSettingsByWorkspaceID :one
SELECT
    c.pk, c.workspace_id, c.username, c.password_encrypted, c.quota_duration_seconds, c.max_queries_per_window, c.max_execution_time_per_window, c.max_query_execution_time, c.max_query_memory_bytes, c.max_query_result_rows, c.created_at, c.updated_at,
    q.pk, q.workspace_id, q.requests_per_month, q.logs_retention_days, q.audit_logs_retention_days, q.team
FROM ` + "`" + `clickhouse_workspace_settings` + "`" + ` c
JOIN ` + "`" + `quota` + "`" + ` q ON c.workspace_id = q.workspace_id
WHERE c.workspace_id = ?
`
```

```go
const findCustomDomainByDomain = `-- name: FindCustomDomainByDomain :one
SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at
FROM custom_domains
WHERE domain = ?
`
```

```go
const findCustomDomainByDomainOrWildcard = `-- name: FindCustomDomainByDomainOrWildcard :one
SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at FROM custom_domains
WHERE domain IN (?, ?)
ORDER BY
    CASE WHEN domain = ? THEN 0 ELSE 1 END
LIMIT 1
`
```

```go
const findCustomDomainById = `-- name: FindCustomDomainById :one
SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at
FROM custom_domains
WHERE id = ?
`
```

```go
const findCustomDomainWithCertByDomain = `-- name: FindCustomDomainWithCertByDomain :one
SELECT
    cd.pk, cd.id, cd.workspace_id, cd.project_id, cd.environment_id, cd.domain, cd.challenge_type, cd.verification_status, cd.verification_token, cd.ownership_verified, cd.cname_verified, cd.target_cname, cd.last_checked_at, cd.check_attempts, cd.verification_error, cd.invocation_id, cd.created_at, cd.updated_at,
    c.id AS certificate_id
FROM custom_domains cd
LEFT JOIN certificates c ON c.hostname = cd.domain
WHERE cd.domain = ?
`
```

```go
const findDeploymentById = `-- name: FindDeploymentById :one
SELECT pk, id, k8s_name, workspace_id, project_id, environment_id, image, build_id, git_commit_sha, git_branch, git_commit_message, git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp, sentinel_config, openapi_spec, cpu_millicores, memory_mib, desired_state, encrypted_environment_variables, command, status, created_at, updated_at FROM ` + "`" + `deployments` + "`" + ` WHERE id = ?
`
```

```go
const findDeploymentByK8sName = `-- name: FindDeploymentByK8sName :one
SELECT pk, id, k8s_name, workspace_id, project_id, environment_id, image, build_id, git_commit_sha, git_branch, git_commit_message, git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp, sentinel_config, openapi_spec, cpu_millicores, memory_mib, desired_state, encrypted_environment_variables, command, status, created_at, updated_at FROM ` + "`" + `deployments` + "`" + ` WHERE k8s_name = ?
`
```

```go
const findDeploymentRegions = `-- name: FindDeploymentRegions :many
SELECT region
FROM ` + "`" + `deployment_topology` + "`" + `
WHERE deployment_id = ?
`
```

```go
const findDeploymentTopologyByIDAndRegion = `-- name: FindDeploymentTopologyByIDAndRegion :one
SELECT
    d.id,
    d.k8s_name,
    w.k8s_namespace,
    d.workspace_id,
    d.project_id,
    d.environment_id,
    d.build_id,
    d.image,
    dt.region,
    d.cpu_millicores,
    d.memory_mib,
    dt.desired_replicas,
    d.desired_state,
    d.encrypted_environment_variables
FROM ` + "`" + `deployment_topology` + "`" + ` dt
INNER JOIN ` + "`" + `deployments` + "`" + ` d ON dt.deployment_id = d.id
INNER JOIN ` + "`" + `workspaces` + "`" + ` w ON d.workspace_id = w.id
WHERE  dt.region = ?
    AND dt.deployment_id = ?
LIMIT 1
`
```

```go
const findEnvironmentById = `-- name: FindEnvironmentById :one
SELECT id, workspace_id, project_id, slug, description
FROM environments
WHERE id = ?
`
```

```go
const findEnvironmentByProjectIdAndSlug = `-- name: FindEnvironmentByProjectIdAndSlug :one
SELECT pk, id, workspace_id, project_id, slug, description, sentinel_config, delete_protection, created_at, updated_at
FROM environments
WHERE workspace_id = ?
  AND project_id = ?
  AND slug = ?
`
```

```go
const findEnvironmentVariablesByEnvironmentId = `-- name: FindEnvironmentVariablesByEnvironmentId :many
SELECT ` + "`" + `key` + "`" + `, value
FROM environment_variables
WHERE environment_id = ?
`
```

```go
const findFrontlineRouteByFQDN = `-- name: FindFrontlineRouteByFQDN :one
SELECT pk, id, project_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at, updated_at FROM frontline_routes WHERE fully_qualified_domain_name = ?
`
```

```go
const findFrontlineRouteForPromotion = `-- name: FindFrontlineRouteForPromotion :many
SELECT
    id,
    project_id,
    environment_id,
    fully_qualified_domain_name,
    deployment_id,
    sticky,
    created_at,
    updated_at
FROM frontline_routes
WHERE
  environment_id = ?
  AND sticky IN (/*SLICE:sticky*/?)
ORDER BY created_at ASC
`
```

```go
const findFrontlineRoutesByDeploymentID = `-- name: FindFrontlineRoutesByDeploymentID :many
SELECT pk, id, project_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at, updated_at FROM frontline_routes WHERE deployment_id = ?
`
```

```go
const findFrontlineRoutesForRollback = `-- name: FindFrontlineRoutesForRollback :many
SELECT
    id,
    project_id,
    environment_id,
    fully_qualified_domain_name,
    deployment_id,
    sticky,
    created_at,
    updated_at
FROM frontline_routes
WHERE
  environment_id = ?
  AND sticky IN (/*SLICE:sticky*/?)
ORDER BY created_at ASC
`
```

```go
const findGithubRepoConnection = `-- name: FindGithubRepoConnection :one
SELECT
    pk,
    project_id,
    installation_id,
    repository_id,
    repository_full_name,
    created_at,
    updated_at
FROM github_repo_connections
WHERE installation_id = ?
  AND repository_id = ?
`
```

```go
const findIdentities = `-- name: FindIdentities :many
SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
FROM identities
WHERE workspace_id = ?
 AND deleted = ?
 AND (external_id IN(/*SLICE:identities*/?) OR id IN (/*SLICE:identities*/?))
`
```

```go
const findIdentitiesByExternalId = `-- name: FindIdentitiesByExternalId :many
SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
FROM identities
WHERE workspace_id = ? AND external_id IN (/*SLICE:externalIds*/?) AND deleted = ?
`
```

```go
const findIdentity = `-- name: FindIdentity :one
SELECT
    i.pk, i.id, i.external_id, i.workspace_id, i.environment, i.meta, i.deleted, i.created_at, i.updated_at,
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', rl.id,
                'name', rl.name,
                'key_id', rl.key_id,
                'identity_id', rl.identity_id,
                'limit', rl.` + "`" + `limit` + "`" + `,
                'duration', rl.duration,
                'auto_apply', rl.auto_apply = 1
            )
        )
        FROM ratelimits rl WHERE rl.identity_id = i.id),
        JSON_ARRAY()
    ) as ratelimits
FROM identities i
JOIN (
    SELECT id1.id FROM identities id1
    WHERE id1.id = ?
      AND id1.workspace_id = ?
      AND id1.deleted = ?
    UNION ALL
    SELECT id2.id FROM identities id2
    WHERE id2.workspace_id = ?
      AND id2.external_id = ?
      AND id2.deleted = ?
) AS identity_lookup ON i.id = identity_lookup.id
LIMIT 1
`
```

```go
const findIdentityByExternalID = `-- name: FindIdentityByExternalID :one
SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
FROM identities
WHERE workspace_id = ?
  AND external_id = ?
  AND deleted = ?
`
```

```go
const findIdentityByID = `-- name: FindIdentityByID :one
SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
FROM identities
WHERE workspace_id = ?
  AND id = ?
  AND deleted = ?
`
```

```go
const findInstanceByPodName = `-- name: FindInstanceByPodName :one
SELECT
 pk, id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status
FROM instances
  WHERE k8s_name = ? AND region = ?
`
```

```go
const findInstancesByDeploymentId = `-- name: FindInstancesByDeploymentId :many
SELECT
 pk, id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status
FROM instances
WHERE deployment_id = ?
`
```

```go
const findInstancesByDeploymentIdAndRegion = `-- name: FindInstancesByDeploymentIdAndRegion :many
SELECT
 pk, id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status
FROM instances
WHERE deployment_id = ? AND region = ?
`
```

```go
const findKeyAuthsByIds = `-- name: FindKeyAuthsByIds :many
SELECT ka.id as key_auth_id, a.id as api_id
FROM apis a
JOIN key_auth as ka ON ka.id = a.key_auth_id
WHERE a.workspace_id = ?
    AND a.id IN (/*SLICE:api_ids*/?)
    AND ka.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
`
```

```go
const findKeyAuthsByKeyAuthIds = `-- name: FindKeyAuthsByKeyAuthIds :many
SELECT ka.id as key_auth_id, a.id as api_id
FROM key_auth as ka
JOIN apis a ON a.key_auth_id = ka.id
WHERE a.workspace_id = ?
    AND ka.id IN (/*SLICE:key_auth_ids*/?)
    AND ka.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
`
```

```go
const findKeyByID = `-- name: FindKeyByID :one
SELECT pk, id, key_auth_id, hash, start, workspace_id, for_workspace_id, name, owner_id, identity_id, meta, expires, created_at_m, updated_at_m, deleted_at_m, refill_day, refill_amount, last_refill_at, enabled, remaining_requests, ratelimit_async, ratelimit_limit, ratelimit_duration, environment, pending_migration_id FROM ` + "`" + `keys` + "`" + ` k
WHERE k.id = ?
`
```

```go
const findKeyCredits = `-- name: FindKeyCredits :one
SELECT remaining_requests FROM ` + "`" + `keys` + "`" + ` k WHERE k.id = ?
`
```

```go
const findKeyEncryptionByKeyID = `-- name: FindKeyEncryptionByKeyID :one
SELECT pk, workspace_id, key_id, created_at, updated_at, encrypted, encryption_key_id FROM encrypted_keys WHERE key_id = ?
`
```

```go
const findKeyForVerification = `-- name: FindKeyForVerification :one
select k.id,
       k.key_auth_id,
       k.workspace_id,
       k.for_workspace_id,
       k.name,
       k.meta,
       k.expires,
       k.deleted_at_m,
       k.refill_day,
       k.refill_amount,
       k.last_refill_at,
       k.enabled,
       k.remaining_requests,
       k.pending_migration_id,
       a.ip_whitelist,
       a.workspace_id  as api_workspace_id,
       a.id            as api_id,
       a.deleted_at_m  as api_deleted_at_m,

       COALESCE(
               (SELECT JSON_ARRAYAGG(name)
                FROM (SELECT name
                      FROM keys_roles kr
                               JOIN roles r ON r.id = kr.role_id
                      WHERE kr.key_id = k.id) as roles),
               JSON_ARRAY()
       )               as roles,

       COALESCE(
               (SELECT JSON_ARRAYAGG(slug)
                FROM (SELECT slug
                      FROM keys_permissions kp
                               JOIN permissions p ON kp.permission_id = p.id
                      WHERE kp.key_id = k.id

                      UNION ALL

                      SELECT slug
                      FROM keys_roles kr
                               JOIN roles_permissions rp ON kr.role_id = rp.role_id
                               JOIN permissions p ON rp.permission_id = p.id
                      WHERE kr.key_id = k.id) as combined_perms),
               JSON_ARRAY()
       )               as permissions,

       coalesce(
               (select json_arrayagg(
                    json_object(
                       'id', rl.id,
                       'name', rl.name,
                       'key_id', rl.key_id,
                       'identity_id', rl.identity_id,
                       'limit', rl.limit,
                       'duration', rl.duration,
                       'auto_apply', rl.auto_apply
                    )
                )
                from ` + "`" + `ratelimits` + "`" + ` rl
                where rl.key_id = k.id
                   OR rl.identity_id = i.id),
               json_array()
       ) as ratelimits,

       i.id as identity_id,
       i.external_id,
       i.meta          as identity_meta,
       ka.deleted_at_m as key_auth_deleted_at_m,
       ws.enabled      as workspace_enabled,
       fws.enabled     as for_workspace_enabled
from ` + "`" + `keys` + "`" + ` k
         JOIN apis a USING (key_auth_id)
         JOIN key_auth ka ON ka.id = k.key_auth_id
         JOIN workspaces ws ON ws.id = k.workspace_id
         LEFT JOIN workspaces fws ON fws.id = k.for_workspace_id
         LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = 0
where k.hash = ?
  and k.deleted_at_m is null
`
```

```go
const findKeyMigrationByID = `-- name: FindKeyMigrationByID :one
SELECT
    id,
    workspace_id,
    algorithm
FROM key_migrations
WHERE id = ?
and workspace_id = ?
`
```

```go
const findKeyRoleByKeyAndRoleID = `-- name: FindKeyRoleByKeyAndRoleID :many
SELECT pk, key_id, role_id, workspace_id, created_at_m, updated_at_m
FROM keys_roles
WHERE key_id = ?
  AND role_id = ?
`
```

```go
const findKeySpaceByID = `-- name: FindKeySpaceByID :one
SELECT pk, id, workspace_id, created_at_m, updated_at_m, deleted_at_m, store_encrypted_keys, default_prefix, default_bytes, size_approx, size_last_updated_at FROM ` + "`" + `key_auth` + "`" + ` WHERE id = ?
`
```

```go
const findKeysByHash = `-- name: FindKeysByHash :many
SELECT id, hash FROM ` + "`" + `keys` + "`" + ` WHERE hash IN (/*SLICE:hashes*/?)
`
```

```go
const findLiveApiByID = `-- name: FindLiveApiByID :one
SELECT apis.pk, apis.id, apis.name, apis.workspace_id, apis.ip_whitelist, apis.auth_type, apis.key_auth_id, apis.created_at_m, apis.updated_at_m, apis.deleted_at_m, apis.delete_protection, ka.pk, ka.id, ka.workspace_id, ka.created_at_m, ka.updated_at_m, ka.deleted_at_m, ka.store_encrypted_keys, ka.default_prefix, ka.default_bytes, ka.size_approx, ka.size_last_updated_at
FROM apis
JOIN key_auth as ka ON ka.id = apis.key_auth_id
WHERE apis.id = ?
    AND ka.deleted_at_m IS NULL
    AND apis.deleted_at_m IS NULL
LIMIT 1
`
```

```go
const findLiveKeyByHash = `-- name: FindLiveKeyByHash :one
SELECT
    k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
    a.pk, a.id, a.name, a.workspace_id, a.ip_whitelist, a.auth_type, a.key_auth_id, a.created_at_m, a.updated_at_m, a.deleted_at_m, a.delete_protection,
    ka.pk, ka.id, ka.workspace_id, ka.created_at_m, ka.updated_at_m, ka.deleted_at_m, ka.store_encrypted_keys, ka.default_prefix, ka.default_bytes, ka.size_approx, ka.size_last_updated_at,
    ws.pk, ws.id, ws.org_id, ws.name, ws.slug, ws.k8s_namespace, ws.partition_id, ws.plan, ws.tier, ws.stripe_customer_id, ws.stripe_subscription_id, ws.beta_features, ws.features, ws.subscriptions, ws.enabled, ws.delete_protection, ws.created_at_m, ws.updated_at_m, ws.deleted_at_m,
    i.id as identity_table_id,
    i.external_id as identity_external_id,
    i.meta as identity_meta,
    ek.encrypted as encrypted_key,
    ek.encryption_key_id as encryption_key_id,

    -- Roles with both IDs and names
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', r.id,
                'name', r.name,
                'description', r.description
            )
        )
        FROM keys_roles kr
        JOIN roles r ON r.id = kr.role_id
        WHERE kr.key_id = k.id),
        JSON_ARRAY()
    ) as roles,

    -- Direct permissions attached to the key
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', p.id,
                'name', p.name,
                'slug', p.slug,
                'description', p.description
            )
        )
        FROM keys_permissions kp
        JOIN permissions p ON kp.permission_id = p.id
        WHERE kp.key_id = k.id),
        JSON_ARRAY()
    ) as permissions,

    -- Permissions from roles
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', p.id,
                'name', p.name,
                'slug', p.slug,
                'description', p.description
            )
        )
        FROM keys_roles kr
        JOIN roles_permissions rp ON kr.role_id = rp.role_id
        JOIN permissions p ON rp.permission_id = p.id
        WHERE kr.key_id = k.id),
        JSON_ARRAY()
    ) as role_permissions,

    -- Rate limits
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', rl.id,
                'name', rl.name,
                'key_id', rl.key_id,
                'identity_id', rl.identity_id,
                'limit', rl.` + "`" + `limit` + "`" + `,
                'duration', rl.duration,
                'auto_apply', rl.auto_apply = 1
            )
        )
        FROM ratelimits rl
        WHERE rl.key_id = k.id OR rl.identity_id = i.id),
        JSON_ARRAY()
    ) as ratelimits

FROM ` + "`" + `keys` + "`" + ` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON ws.id = k.workspace_id
LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
WHERE k.hash = ?
    AND k.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
    AND ka.deleted_at_m IS NULL
    AND ws.deleted_at_m IS NULL
`
```

```go
const findLiveKeyByID = `-- name: FindLiveKeyByID :one
SELECT
    k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
    a.pk, a.id, a.name, a.workspace_id, a.ip_whitelist, a.auth_type, a.key_auth_id, a.created_at_m, a.updated_at_m, a.deleted_at_m, a.delete_protection,
    ka.pk, ka.id, ka.workspace_id, ka.created_at_m, ka.updated_at_m, ka.deleted_at_m, ka.store_encrypted_keys, ka.default_prefix, ka.default_bytes, ka.size_approx, ka.size_last_updated_at,
    ws.pk, ws.id, ws.org_id, ws.name, ws.slug, ws.k8s_namespace, ws.partition_id, ws.plan, ws.tier, ws.stripe_customer_id, ws.stripe_subscription_id, ws.beta_features, ws.features, ws.subscriptions, ws.enabled, ws.delete_protection, ws.created_at_m, ws.updated_at_m, ws.deleted_at_m,
    i.id as identity_table_id,
    i.external_id as identity_external_id,
    i.meta as identity_meta,
    ek.encrypted as encrypted_key,
    ek.encryption_key_id as encryption_key_id,

    -- Roles with both IDs and names
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', r.id,
                'name', r.name,
                'description', r.description
            )
        )
        FROM keys_roles kr
        JOIN roles r ON r.id = kr.role_id
        WHERE kr.key_id = k.id),
        JSON_ARRAY()
    ) as roles,

    -- Direct permissions attached to the key
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', p.id,
                'name', p.name,
                'slug', p.slug,
                'description', p.description
            )
        )
        FROM keys_permissions kp
        JOIN permissions p ON kp.permission_id = p.id
        WHERE kp.key_id = k.id),
        JSON_ARRAY()
    ) as permissions,

    -- Permissions from roles
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', p.id,
                'name', p.name,
                'slug', p.slug,
                'description', p.description
            )
        )
        FROM keys_roles kr
        JOIN roles_permissions rp ON kr.role_id = rp.role_id
        JOIN permissions p ON rp.permission_id = p.id
        WHERE kr.key_id = k.id),
        JSON_ARRAY()
    ) as role_permissions,

    -- Rate limits
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', rl.id,
                'name', rl.name,
                'key_id', rl.key_id,
                'identity_id', rl.identity_id,
                'limit', rl.` + "`" + `limit` + "`" + `,
                'duration', rl.duration,
                'auto_apply', rl.auto_apply = 1
            )
        )
        FROM ratelimits rl
        WHERE rl.key_id = k.id
            OR rl.identity_id = i.id),
        JSON_ARRAY()
    ) as ratelimits

FROM ` + "`" + `keys` + "`" + ` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON ws.id = k.workspace_id
LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
WHERE k.id = ?
    AND k.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
    AND ka.deleted_at_m IS NULL
    AND ws.deleted_at_m IS NULL
`
```

```go
const findManyRatelimitNamespaces = `-- name: FindManyRatelimitNamespaces :many
SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m,
       coalesce(
               (select json_arrayagg(
                               json_object(
                                       'id', ro.id,
                                       'identifier', ro.identifier,
                                       'limit', ro.limit,
                                       'duration', ro.duration
                               )
                       )
                from ratelimit_overrides ro where ro.namespace_id = ns.id AND ro.deleted_at_m IS NULL),
               json_array()
       ) as overrides
FROM ` + "`" + `ratelimit_namespaces` + "`" + ` ns
WHERE ns.workspace_id = ?
  AND (ns.id IN (/*SLICE:namespaces*/?) OR ns.name IN (/*SLICE:namespaces*/?))
`
```

```go
const findManyRolesByIdOrNameWithPerms = `-- name: FindManyRolesByIdOrNameWithPerms :many
SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
        (SELECT JSON_ARRAYAGG(
            json_object(
                'id', permission.id,
                'name', permission.name,
                'slug', permission.slug,
                'description', permission.description
           )
        )
         FROM (SELECT name, id, slug, description
               FROM roles_permissions rp
                        JOIN permissions p ON p.id = rp.permission_id
               WHERE rp.role_id = r.id) as permission),
        JSON_ARRAY()
) as permissions
FROM roles r
WHERE r.workspace_id = ? AND (
    r.id IN (/*SLICE:search*/?)
    OR r.name IN (/*SLICE:search*/?)
)
`
```

```go
const findManyRolesByNamesWithPerms = `-- name: FindManyRolesByNamesWithPerms :many
SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
        (SELECT JSON_ARRAYAGG(
            json_object(
                'id', permission.id,
                'name', permission.name,
                'slug', permission.slug,
                'description', permission.description
           )
        )
         FROM (SELECT name, id, slug, description
               FROM roles_permissions rp
                        JOIN permissions p ON p.id = rp.permission_id
               WHERE rp.role_id = r.id) as permission),
        JSON_ARRAY()
) as permissions
FROM roles r
WHERE r.workspace_id = ? AND r.name IN (/*SLICE:names*/?)
`
```

```go
const findPermissionByID = `-- name: FindPermissionByID :one
SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
FROM permissions
WHERE id = ?
LIMIT 1
`
```

```go
const findPermissionByIdOrSlug = `-- name: FindPermissionByIdOrSlug :one
SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
FROM permissions
WHERE workspace_id = ? AND (id = ? OR slug = ?)
`
```

```go
const findPermissionByNameAndWorkspaceID = `-- name: FindPermissionByNameAndWorkspaceID :one
SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
FROM permissions
WHERE name = ?
AND workspace_id = ?
LIMIT 1
`
```

```go
const findPermissionBySlugAndWorkspaceID = `-- name: FindPermissionBySlugAndWorkspaceID :one
SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
FROM permissions
WHERE slug = ?
AND workspace_id = ?
LIMIT 1
`
```

```go
const findPermissionsBySlugs = `-- name: FindPermissionsBySlugs :many
SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m FROM permissions WHERE workspace_id = ? AND slug IN (/*SLICE:slugs*/?)
`
```

```go
const findProjectById = `-- name: FindProjectById :one
SELECT
    id,
    workspace_id,
    name,
    slug,
    git_repository_url,
    default_branch,
    delete_protection,
    live_deployment_id,
    is_rolled_back,
    created_at,
    updated_at,
    depot_project_id,
    command
FROM projects
WHERE id = ?
`
```

```go
const findProjectByWorkspaceSlug = `-- name: FindProjectByWorkspaceSlug :one
SELECT
    id,
    workspace_id,
    name,
    slug,
    git_repository_url,
    default_branch,
    delete_protection,
    created_at,
    updated_at
FROM projects
WHERE workspace_id = ? AND slug = ?
LIMIT 1
`
```

```go
const findQuotaByWorkspaceID = `-- name: FindQuotaByWorkspaceID :one
SELECT pk, workspace_id, requests_per_month, logs_retention_days, audit_logs_retention_days, team
FROM ` + "`" + `quota` + "`" + `
WHERE workspace_id = ?
`
```

```go
const findRatelimitNamespace = `-- name: FindRatelimitNamespace :one
SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m,
       coalesce(
               (select json_arrayagg(
                               json_object(
                                       'id', ro.id,
                                       'identifier', ro.identifier,
                                       'limit', ro.limit,
                                       'duration', ro.duration
                               )
                       )
                from ratelimit_overrides ro where ro.namespace_id = ns.id AND ro.deleted_at_m IS NULL),
               json_array()
       ) as overrides
FROM ` + "`" + `ratelimit_namespaces` + "`" + ` ns
WHERE ns.workspace_id = ?
AND (ns.id = ? OR ns.name = ?)
`
```

```go
const findRatelimitNamespaceByID = `-- name: FindRatelimitNamespaceByID :one
SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m FROM ` + "`" + `ratelimit_namespaces` + "`" + `
WHERE id = ?
`
```

```go
const findRatelimitNamespaceByName = `-- name: FindRatelimitNamespaceByName :one
SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m FROM ` + "`" + `ratelimit_namespaces` + "`" + `
WHERE name = ?
AND workspace_id = ?
`
```

```go
const findRatelimitOverrideByID = `-- name: FindRatelimitOverrideByID :one
SELECT pk, id, workspace_id, namespace_id, identifier, ` + "`" + `limit` + "`" + `, duration, async, sharding, created_at_m, updated_at_m, deleted_at_m FROM ratelimit_overrides
WHERE
    workspace_id = ?
    AND id = ?
`
```

```go
const findRatelimitOverrideByIdentifier = `-- name: FindRatelimitOverrideByIdentifier :one
SELECT pk, id, workspace_id, namespace_id, identifier, ` + "`" + `limit` + "`" + `, duration, async, sharding, created_at_m, updated_at_m, deleted_at_m FROM ratelimit_overrides
WHERE
    workspace_id = ?
    AND namespace_id = ?
    AND identifier = ?
`
```

```go
const findRoleByID = `-- name: FindRoleByID :one
SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m
FROM roles
WHERE id = ?
LIMIT 1
`
```

```go
const findRoleByIdOrNameWithPerms = `-- name: FindRoleByIdOrNameWithPerms :one
SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
        (SELECT JSON_ARRAYAGG(
            json_object(
                'id', permission.id,
                'name', permission.name,
                'slug', permission.slug,
                'description', permission.description
           )
        )
         FROM (SELECT name, id, slug, description
               FROM roles_permissions rp
                        JOIN permissions p ON p.id = rp.permission_id
               WHERE rp.role_id = r.id) as permission),
        JSON_ARRAY()
) as permissions
FROM roles r
WHERE r.workspace_id = ? AND (
    r.id = ?
    OR r.name = ?
)
`
```

```go
const findRoleByNameAndWorkspaceID = `-- name: FindRoleByNameAndWorkspaceID :one
SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m
FROM roles
WHERE name = ?
AND workspace_id = ?
LIMIT 1
`
```

```go
const findRolePermissionByRoleAndPermissionID = `-- name: FindRolePermissionByRoleAndPermissionID :many
SELECT pk, role_id, permission_id, workspace_id, created_at_m, updated_at_m
FROM roles_permissions
WHERE role_id = ?
  AND permission_id = ?
`
```

```go
const findRolesByNames = `-- name: FindRolesByNames :many
SELECT id, name FROM roles WHERE workspace_id = ? AND name IN (/*SLICE:names*/?)
`
```

```go
const findSentinelByID = `-- name: FindSentinelByID :one
SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at FROM sentinels s
WHERE id = ? LIMIT 1
`
```

```go
const findSentinelsByEnvironmentID = `-- name: FindSentinelsByEnvironmentID :many
SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at FROM sentinels WHERE environment_id = ?
`
```

```go
const findWorkspaceByID = `-- name: FindWorkspaceByID :one
SELECT pk, id, org_id, name, slug, k8s_namespace, partition_id, plan, tier, stripe_customer_id, stripe_subscription_id, beta_features, features, subscriptions, enabled, delete_protection, created_at_m, updated_at_m, deleted_at_m FROM ` + "`" + `workspaces` + "`" + `
WHERE id = ?
`
```

```go
const getKeyAuthByID = `-- name: GetKeyAuthByID :one
SELECT
    id,
    workspace_id,
    created_at_m,
    default_prefix,
    default_bytes,
    store_encrypted_keys
FROM key_auth
WHERE id = ?
  AND deleted_at_m IS NULL
`
```

```go
const getWorkspacesForQuotaCheckByIDs = `-- name: GetWorkspacesForQuotaCheckByIDs :many
SELECT
   w.id,
   w.org_id,
   w.name,
   w.stripe_customer_id,
   w.tier,
   w.enabled,
   q.requests_per_month
FROM ` + "`" + `workspaces` + "`" + ` w
LEFT JOIN quota q ON w.id = q.workspace_id
WHERE w.id IN (/*SLICE:workspace_ids*/?)
`
```

```go
const hardDeleteWorkspace = `-- name: HardDeleteWorkspace :execresult
DELETE FROM ` + "`" + `workspaces` + "`" + `
WHERE id = ?
AND delete_protection = false
`
```

```go
const insertAcmeChallenge = `-- name: InsertAcmeChallenge :exec
INSERT INTO acme_challenges (
    workspace_id,
    domain_id,
    token,
    authorization,
    status,
    challenge_type,
    created_at,
    updated_at,
    expires_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`
```

```go
const insertAcmeUser = `-- name: InsertAcmeUser :exec

INSERT INTO acme_users (id, workspace_id, encrypted_key, created_at)
VALUES (?,?,?,?)
`
```

```go
const insertApi = `-- name: InsertApi :exec
INSERT INTO apis (
    id,
    name,
    workspace_id,
    auth_type,
    ip_whitelist,
    key_auth_id,
    created_at_m,
    deleted_at_m
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    NULL
)
`
```

```go
const insertAuditLog = `-- name: InsertAuditLog :exec
INSERT INTO ` + "`" + `audit_log` + "`" + ` (
    id,
    workspace_id,
    bucket_id,
    bucket,
    event,
    time,
    display,
    remote_ip,
    user_agent,
    actor_type,
    actor_id,
    actor_name,
    actor_meta,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    CAST(? AS JSON),
    ?
)
`
```

```go
const insertAuditLogTarget = `-- name: InsertAuditLogTarget :exec
INSERT INTO ` + "`" + `audit_log_target` + "`" + ` (
    workspace_id,
    bucket_id,
    bucket,
    audit_log_id,
    display_name,
    type,
    id,
    name,
    meta,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    CAST(? AS JSON),
    ?
)
`
```

```go
const insertCertificate = `-- name: InsertCertificate :exec
INSERT INTO certificates (id, workspace_id, hostname, certificate, encrypted_private_key, created_at)
VALUES (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE
workspace_id = VALUES(workspace_id),
hostname = VALUES(hostname),
certificate = VALUES(certificate),
encrypted_private_key = VALUES(encrypted_private_key),
updated_at = ?
`
```

```go
const insertCiliumNetworkPolicy = `-- name: InsertCiliumNetworkPolicy :exec
INSERT INTO cilium_network_policies (
    id,
    workspace_id,
    project_id,
    environment_id,
    k8s_name,
    region,
    policy,
    version,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`
```

```go
const insertClickhouseWorkspaceSettings = `-- name: InsertClickhouseWorkspaceSettings :exec
INSERT INTO ` + "`" + `clickhouse_workspace_settings` + "`" + ` (
    workspace_id,
    username,
    password_encrypted,
    quota_duration_seconds,
    max_queries_per_window,
    max_execution_time_per_window,
    max_query_execution_time,
    max_query_memory_bytes,
    max_query_result_rows,
    created_at,
    updated_at
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`
```

```go
const insertCustomDomain = `-- name: InsertCustomDomain :exec
INSERT INTO custom_domains (
    id, workspace_id, project_id, environment_id, domain,
    challenge_type, verification_status, verification_token, target_cname, invocation_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
```

```go
const insertDeployment = `-- name: InsertDeployment :exec
INSERT INTO ` + "`" + `deployments` + "`" + ` (
    id,
    k8s_name,
    workspace_id,
    project_id,
    environment_id,
    git_commit_sha,
    git_branch,
    sentinel_config,
    git_commit_message,
    git_commit_author_handle,
    git_commit_author_avatar_url,
    git_commit_timestamp,
    openapi_spec,
    encrypted_environment_variables,
    command,
    status,
    cpu_millicores,
    memory_mib,
    created_at,
    updated_at
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`
```

```go
const insertDeploymentTopology = `-- name: InsertDeploymentTopology :exec
INSERT INTO ` + "`" + `deployment_topology` + "`" + ` (
    workspace_id,
    deployment_id,
    region,
    desired_replicas,
    desired_status,
    version,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`
```

```go
const insertEnvironment = `-- name: InsertEnvironment :exec
INSERT INTO environments (
    id,
    workspace_id,
    project_id,
    slug,
    description,
    created_at,
    updated_at,
    sentinel_config
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
)
`
```

```go
const insertFrontlineRoute = `-- name: InsertFrontlineRoute :exec
INSERT INTO frontline_routes (
    id,
    project_id,
    deployment_id,
    environment_id,
    fully_qualified_domain_name,
    sticky,
    created_at,
    updated_at
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`
```

```go
const insertGithubRepoConnection = `-- name: InsertGithubRepoConnection :exec
INSERT INTO github_repo_connections (
    project_id,
    installation_id,
    repository_id,
    repository_full_name,
    created_at,
    updated_at
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`
```

```go
const insertIdentity = `-- name: InsertIdentity :exec
INSERT INTO ` + "`" + `identities` + "`" + ` (
    id,
    external_id,
    workspace_id,
    environment,
    created_at,
    meta
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    CAST(? AS JSON)
)
`
```

```go
const insertIdentityRatelimit = `-- name: InsertIdentityRatelimit :exec
INSERT INTO ` + "`" + `ratelimits` + "`" + ` (
    id,
    workspace_id,
    identity_id,
    name,
    ` + "`" + `limit` + "`" + `,
    duration,
    created_at,
    auto_apply
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
) ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    ` + "`" + `limit` + "`" + ` = VALUES(` + "`" + `limit` + "`" + `),
    duration = VALUES(duration),
    auto_apply = VALUES(auto_apply),
    updated_at = VALUES(created_at)
`
```

```go
const insertKey = `-- name: InsertKey :exec
INSERT INTO ` + "`" + `keys` + "`" + ` (
    id,
    key_auth_id,
    hash,
    start,
    workspace_id,
    for_workspace_id,
    name,
    owner_id,
    identity_id,
    meta,
    expires,
    created_at_m,
    enabled,
    remaining_requests,
    refill_day,
    refill_amount,
    pending_migration_id
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    null,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`
```

```go
const insertKeyAuth = `-- name: InsertKeyAuth :exec
INSERT INTO key_auth (
    id,
    workspace_id,
    created_at_m,
    default_prefix,
    default_bytes,
    store_encrypted_keys
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    false
)
`
```

```go
const insertKeyEncryption = `-- name: InsertKeyEncryption :exec
INSERT INTO encrypted_keys
(workspace_id, key_id, encrypted, encryption_key_id, created_at)
VALUES (?, ?, ?, ?, ?)
`
```

```go
const insertKeyMigration = `-- name: InsertKeyMigration :exec
INSERT INTO key_migrations (
    id,
    workspace_id,
    algorithm
) VALUES (
    ?,
    ?,
    ?
)
`
```

```go
const insertKeyPermission = `-- name: InsertKeyPermission :exec
INSERT INTO ` + "`" + `keys_permissions` + "`" + ` (
    key_id,
    permission_id,
    workspace_id,
    created_at_m
) VALUES (
    ?,
    ?,
    ?,
    ?
) ON DUPLICATE KEY UPDATE updated_at_m = ?
`
```

```go
const insertKeyRatelimit = `-- name: InsertKeyRatelimit :exec
INSERT INTO ` + "`" + `ratelimits` + "`" + ` (
    id,
    workspace_id,
    key_id,
    name,
    ` + "`" + `limit` + "`" + `,
    duration,
    auto_apply,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
) ON DUPLICATE KEY UPDATE
` + "`" + `limit` + "`" + ` = VALUES(` + "`" + `limit` + "`" + `),
duration = VALUES(duration),
auto_apply = VALUES(auto_apply),
updated_at = ?
`
```

```go
const insertKeyRole = `-- name: InsertKeyRole :exec
INSERT INTO keys_roles (
  key_id,
  role_id,
  workspace_id,
  created_at_m
)
VALUES (
  ?,
  ?,
  ?,
  ?
)
`
```

```go
const insertKeySpace = `-- name: InsertKeySpace :exec
INSERT INTO ` + "`" + `key_auth` + "`" + ` (
    id,
    workspace_id,
    created_at_m,
    store_encrypted_keys,
    default_prefix,
    default_bytes,
    size_approx,
    size_last_updated_at
) VALUES (
    ?,
    ?,
      ?,
    ?,
    ?,
    ?,
    0,
    0
)
`
```

```go
const insertPermission = `-- name: InsertPermission :exec
INSERT INTO permissions (
  id,
  workspace_id,
  name,
  slug,
  description,
  created_at_m
)
VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
```

```go
const insertProject = `-- name: InsertProject :exec
INSERT INTO projects (
    id,
    workspace_id,
    name,
    slug,
    git_repository_url,
    default_branch,
    delete_protection,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
`
```

```go
const insertRatelimitNamespace = `-- name: InsertRatelimitNamespace :exec
INSERT INTO
    ` + "`" + `ratelimit_namespaces` + "`" + ` (
        id,
        workspace_id,
        name,
        created_at_m,
        updated_at_m,
        deleted_at_m
        )
VALUES
    (
        ?,
        ?,
        ?,
         ?,
        NULL,
        NULL
    )
`
```

```go
const insertRatelimitOverride = `-- name: InsertRatelimitOverride :exec
INSERT INTO ratelimit_overrides (
    id,
    workspace_id,
    namespace_id,
    identifier,
    ` + "`" + `limit` + "`" + `,
    duration,
    async,
    created_at_m
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    false,
    ?
)
ON DUPLICATE KEY UPDATE
    ` + "`" + `limit` + "`" + ` = VALUES(` + "`" + `limit` + "`" + `),
    duration = VALUES(duration),
    async = VALUES(async),
    updated_at_m = ?
`
```

```go
const insertRole = `-- name: InsertRole :exec
INSERT INTO roles (
  id,
  workspace_id,
  name,
  description,
  created_at_m
)
VALUES (
  ?,
  ?,
  ?,
  ?,
  ?
)
`
```

```go
const insertRolePermission = `-- name: InsertRolePermission :exec
INSERT INTO roles_permissions (
  role_id,
  permission_id,
  workspace_id,
  created_at_m
)
VALUES (
  ?,
  ?,
  ?,
  ?
)
`
```

```go
const insertSentinel = `-- name: InsertSentinel :exec
INSERT INTO sentinels (
    id,
    workspace_id,
    environment_id,
    project_id,
    k8s_address,
    k8s_name,
    region,
    image,
    health,
    desired_replicas,
    available_replicas,
    cpu_millicores,
    memory_mib,
    version,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
`
```

```go
const insertWorkspace = `-- name: InsertWorkspace :exec
INSERT INTO ` + "`" + `workspaces` + "`" + ` (
    id,
    org_id,
    name,
    slug,
    created_at_m,
    tier,
    beta_features,
    features,
    enabled,
    delete_protection
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    'Free',
    '{}',
    '{}',
    true,
    true
)
`
```

```go
const listCiliumNetworkPoliciesByRegion = `-- name: ListCiliumNetworkPoliciesByRegion :many
SELECT
    n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
    w.k8s_namespace
FROM ` + "`" + `cilium_network_policies` + "`" + ` n
JOIN ` + "`" + `workspaces` + "`" + ` w ON w.id = n.workspace_id
WHERE n.region = ? AND n.version > ?
ORDER BY n.version ASC
LIMIT ?
`
```

```go
const listCustomDomainsByProjectID = `-- name: ListCustomDomainsByProjectID :many
SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at
FROM custom_domains
WHERE project_id = ?
ORDER BY created_at DESC
`
```

```go
const listDeploymentTopologyByRegion = `-- name: ListDeploymentTopologyByRegion :many
SELECT
    dt.pk, dt.workspace_id, dt.deployment_id, dt.region, dt.desired_replicas, dt.version, dt.desired_status, dt.created_at, dt.updated_at,
    d.pk, d.id, d.k8s_name, d.workspace_id, d.project_id, d.environment_id, d.image, d.build_id, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.sentinel_config, d.openapi_spec, d.cpu_millicores, d.memory_mib, d.desired_state, d.encrypted_environment_variables, d.command, d.status, d.created_at, d.updated_at,
    w.k8s_namespace
FROM ` + "`" + `deployment_topology` + "`" + ` dt
INNER JOIN ` + "`" + `deployments` + "`" + ` d ON dt.deployment_id = d.id
INNER JOIN ` + "`" + `workspaces` + "`" + ` w ON d.workspace_id = w.id
WHERE dt.region = ? AND dt.version > ?
ORDER BY dt.version ASC
LIMIT ?
`
```

```go
const listDesiredDeploymentTopology = `-- name: ListDesiredDeploymentTopology :many
SELECT
    dt.pk, dt.workspace_id, dt.deployment_id, dt.region, dt.desired_replicas, dt.version, dt.desired_status, dt.created_at, dt.updated_at,
    d.pk, d.id, d.k8s_name, d.workspace_id, d.project_id, d.environment_id, d.image, d.build_id, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.sentinel_config, d.openapi_spec, d.cpu_millicores, d.memory_mib, d.desired_state, d.encrypted_environment_variables, d.command, d.status, d.created_at, d.updated_at,
    w.k8s_namespace
FROM ` + "`" + `deployment_topology` + "`" + ` dt
INNER JOIN ` + "`" + `deployments` + "`" + ` d ON dt.deployment_id = d.id
INNER JOIN ` + "`" + `workspaces` + "`" + ` w ON d.workspace_id = w.id
WHERE (? = '' OR dt.region = ?)
    AND d.desired_state = ?
    AND dt.deployment_id > ?
ORDER BY dt.deployment_id ASC
LIMIT ?
`
```

```go
const listDesiredNetworkPolicies = `-- name: ListDesiredNetworkPolicies :many
SELECT
    n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
    w.k8s_namespace
FROM ` + "`" + `cilium_network_policies` + "`" + ` n
INNER JOIN ` + "`" + `workspaces` + "`" + ` w ON n.workspace_id = w.id
WHERE (? = '' OR n.region = ?) AND n.id > ?
ORDER BY n.id ASC
LIMIT ?
`
```

```go
const listDesiredSentinels = `-- name: ListDesiredSentinels :many
SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at
FROM ` + "`" + `sentinels` + "`" + `
WHERE (? = '' OR region = ?)
    AND desired_state = ?
    AND id > ?
ORDER BY id ASC
LIMIT ?
`
```

```go
const listDirectPermissionsByKeyID = `-- name: ListDirectPermissionsByKeyID :many
SELECT p.pk, p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
FROM keys_permissions kp
JOIN permissions p ON kp.permission_id = p.id
WHERE kp.key_id = ?
ORDER BY p.slug
`
```

```go
const listExecutableChallenges = `-- name: ListExecutableChallenges :many
SELECT dc.workspace_id, dc.challenge_type, d.domain FROM acme_challenges dc
JOIN custom_domains d ON dc.domain_id = d.id
WHERE (dc.status = 'waiting' OR (dc.status = 'verified' AND dc.expires_at <= UNIX_TIMESTAMP(DATE_ADD(NOW(), INTERVAL 30 DAY)) * 1000))
AND dc.challenge_type IN (/*SLICE:verification_types*/?)
ORDER BY d.created_at ASC
`
```

```go
const listIdentities = `-- name: ListIdentities :many
SELECT
    i.id,
    i.external_id,
    i.workspace_id,
    i.environment,
    i.meta,
    i.deleted,
    i.created_at,
    i.updated_at,
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', r.id,
                'name', r.name,
                'limit', r.` + "`" + `limit` + "`" + `,
                'duration', r.duration,
                'auto_apply', r.auto_apply = 1
            )
        )
        FROM ratelimits r
        WHERE r.identity_id = i.id),
        JSON_ARRAY()
    ) as ratelimits
FROM identities i
WHERE i.workspace_id = ?
AND i.deleted = ?
AND i.id >= ?
ORDER BY i.id ASC
LIMIT ?
`
```

```go
const listIdentityRatelimits = `-- name: ListIdentityRatelimits :many
SELECT pk, id, name, workspace_id, created_at, updated_at, key_id, identity_id, ` + "`" + `limit` + "`" + `, duration, auto_apply
FROM ratelimits
WHERE identity_id = ?
ORDER BY id ASC
`
```

```go
const listIdentityRatelimitsByID = `-- name: ListIdentityRatelimitsByID :many
SELECT pk, id, name, workspace_id, created_at, updated_at, key_id, identity_id, ` + "`" + `limit` + "`" + `, duration, auto_apply FROM ratelimits WHERE identity_id = ?
`
```

```go
const listIdentityRatelimitsByIDs = `-- name: ListIdentityRatelimitsByIDs :many
SELECT pk, id, name, workspace_id, created_at, updated_at, key_id, identity_id, ` + "`" + `limit` + "`" + `, duration, auto_apply FROM ratelimits WHERE identity_id IN (/*SLICE:ids*/?)
`
```

```go
const listKeysByKeySpaceID = `-- name: ListKeysByKeySpaceID :many
SELECT
  k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
  i.id as identity_id,
  i.external_id as external_id,
  i.meta as identity_meta,
  ek.encrypted as encrypted_key,
  ek.encryption_key_id as encryption_key_id

FROM ` + "`" + `keys` + "`" + ` k
LEFT JOIN ` + "`" + `identities` + "`" + ` i ON k.identity_id = i.id
LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
WHERE k.key_auth_id = ?
AND k.id >= ?
AND (? IS NULL OR k.identity_id = ?)
AND k.deleted_at_m IS NULL
ORDER BY k.id ASC
LIMIT ?
`
```

```go
const listLiveKeysByKeySpaceID = `-- name: ListLiveKeysByKeySpaceID :many
SELECT k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
       i.id                 as identity_table_id,
       i.external_id        as identity_external_id,
       i.meta               as identity_meta,
       ek.encrypted         as encrypted_key,
       ek.encryption_key_id as encryption_key_id,
       -- Roles with both IDs and names (sorted by name)
       COALESCE(
               (SELECT JSON_ARRAYAGG(
                               JSON_OBJECT(
                                       'id', r.id,
                                       'name', r.name,
                                       'description', r.description
                               )
                       )
                FROM keys_roles kr
                         JOIN roles r ON r.id = kr.role_id
                WHERE kr.key_id = k.id
                ORDER BY r.name),
               JSON_ARRAY()
       )                    as roles,
       -- Direct permissions attached to the key (sorted by slug)
       COALESCE(
               (SELECT JSON_ARRAYAGG(
                               JSON_OBJECT(
                                       'id', p.id,
                                       'name', p.name,
                                       'slug', p.slug,
                                       'description', p.description
                               )
                       )
                FROM keys_permissions kp
                         JOIN permissions p ON kp.permission_id = p.id
                WHERE kp.key_id = k.id
                ORDER BY p.slug),
               JSON_ARRAY()
       )                    as permissions,
       -- Permissions from roles (sorted by slug)
       COALESCE(
               (SELECT JSON_ARRAYAGG(
                               JSON_OBJECT(
                                       'id', p.id,
                                       'name', p.name,
                                       'slug', p.slug,
                                       'description', p.description
                               )
                       )
                FROM keys_roles kr
                         JOIN roles_permissions rp ON kr.role_id = rp.role_id
                         JOIN permissions p ON rp.permission_id = p.id
                WHERE kr.key_id = k.id
                ORDER BY p.slug),
               JSON_ARRAY()
       )                    as role_permissions,
       -- Rate limits
       COALESCE(
               (SELECT JSON_ARRAYAGG(
                               JSON_OBJECT(
                                       'id', id,
                                       'name', name,
                                       'key_id', key_id,
                                       'identity_id', identity_id,
                                       'limit', ` + "`" + `limit` + "`" + `,
                                       'duration', duration,
                                       'auto_apply', auto_apply = 1
                               )
                       )
                FROM (
                    SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.` + "`" + `limit` + "`" + `, rl.duration, rl.auto_apply
                    FROM ratelimits rl
                    WHERE rl.key_id = k.id
                    UNION ALL
                    SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.` + "`" + `limit` + "`" + `, rl.duration, rl.auto_apply
                    FROM ratelimits rl
                    WHERE rl.identity_id = i.id
                ) AS combined_rl),
               JSON_ARRAY()
       )                    AS ratelimits
FROM ` + "`" + `keys` + "`" + ` k
         STRAIGHT_JOIN key_auth ka ON ka.id = k.key_auth_id
         LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
         LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
WHERE k.key_auth_id = ?
  AND k.id >= ?
  AND (? IS NULL OR k.identity_id = ?)
  AND k.deleted_at_m IS NULL
  AND ka.deleted_at_m IS NULL
ORDER BY k.id ASC
LIMIT ?
`
```

```go
const listNetworkPolicyByRegion = `-- name: ListNetworkPolicyByRegion :many
SELECT
    n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
    w.k8s_namespace
FROM ` + "`" + `cilium_network_policies` + "`" + ` n
INNER JOIN ` + "`" + `workspaces` + "`" + ` w ON n.workspace_id = w.id
WHERE n.region = ? AND n.version > ?
ORDER BY n.version ASC
LIMIT ?
`
```

```go
const listPermissions = `-- name: ListPermissions :many
SELECT p.pk, p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
FROM permissions p
WHERE p.workspace_id = ?
  AND p.id >= ?
ORDER BY p.id
LIMIT ?
`
```

```go
const listPermissionsByKeyID = `-- name: ListPermissionsByKeyID :many
WITH direct_permissions AS (
    SELECT p.slug as permission_slug
    FROM keys_permissions kp
    JOIN permissions p ON kp.permission_id = p.id
    WHERE kp.key_id = ?
),
role_permissions AS (
    SELECT p.slug as permission_slug
    FROM keys_roles kr
    JOIN roles_permissions rp ON kr.role_id = rp.role_id
    JOIN permissions p ON rp.permission_id = p.id
    WHERE kr.key_id = ?
)
SELECT DISTINCT permission_slug
FROM (
    SELECT permission_slug FROM direct_permissions
    UNION ALL
    SELECT permission_slug FROM role_permissions
) all_permissions
`
```

```go
const listPermissionsByRoleID = `-- name: ListPermissionsByRoleID :many
SELECT p.pk, p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
FROM permissions p
JOIN roles_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = ?
ORDER BY p.slug
`
```

```go
const listRatelimitOverridesByNamespaceID = `-- name: ListRatelimitOverridesByNamespaceID :many
SELECT pk, id, workspace_id, namespace_id, identifier, ` + "`" + `limit` + "`" + `, duration, async, sharding, created_at_m, updated_at_m, deleted_at_m FROM ratelimit_overrides
WHERE
workspace_id = ?
AND namespace_id = ?
AND deleted_at_m IS NULL
AND id >= ?
ORDER BY id ASC
LIMIT ?
`
```

```go
const listRatelimitsByKeyID = `-- name: ListRatelimitsByKeyID :many
SELECT
  id,
  name,
  ` + "`" + `limit` + "`" + `,
  duration,
  auto_apply
FROM ratelimits
WHERE key_id = ?
`
```

```go
const listRatelimitsByKeyIDs = `-- name: ListRatelimitsByKeyIDs :many
SELECT
  id,
  key_id,
  name,
  ` + "`" + `limit` + "`" + `,
  duration,
  auto_apply
FROM ratelimits
WHERE key_id IN (/*SLICE:key_ids*/?)
ORDER BY key_id, id
`
```

```go
const listRoles = `-- name: ListRoles :many
SELECT r.pk, r.id, r.workspace_id, r.name, r.description, r.created_at_m, r.updated_at_m, COALESCE(
        (SELECT JSON_ARRAYAGG(
            json_object(
                'id', permission.id,
                'name', permission.name,
                'slug', permission.slug,
                'description', permission.description
           )
        )
         FROM (SELECT name, id, slug, description
               FROM roles_permissions rp
                        JOIN permissions p ON p.id = rp.permission_id
               WHERE rp.role_id = r.id) as permission),
        JSON_ARRAY()
) as permissions
FROM roles r
WHERE r.workspace_id = ?
AND r.id >= ?
ORDER BY r.id
LIMIT ?
`
```

```go
const listRolesByKeyID = `-- name: ListRolesByKeyID :many
SELECT r.pk, r.id, r.workspace_id, r.name, r.description, r.created_at_m, r.updated_at_m, COALESCE(
        (SELECT JSON_ARRAYAGG(
            json_object(
                'id', permission.id,
                'name', permission.name,
                'slug', permission.slug,
                'description', permission.description
           )
        )
         FROM (SELECT name, id, slug, description
               FROM roles_permissions rp
                        JOIN permissions p ON p.id = rp.permission_id
               WHERE rp.role_id = r.id) as permission),
        JSON_ARRAY()
) as permissions
FROM keys_roles kr
JOIN roles r ON kr.role_id = r.id
WHERE kr.key_id = ?
ORDER BY r.name
`
```

```go
const listSentinelsByRegion = `-- name: ListSentinelsByRegion :many
SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at FROM ` + "`" + `sentinels` + "`" + `
WHERE region = ? AND version > ?
ORDER BY version ASC
LIMIT ?
`
```

```go
const listWorkspaces = `-- name: ListWorkspaces :many
SELECT
   w.pk, w.id, w.org_id, w.name, w.slug, w.k8s_namespace, w.partition_id, w.plan, w.tier, w.stripe_customer_id, w.stripe_subscription_id, w.beta_features, w.features, w.subscriptions, w.enabled, w.delete_protection, w.created_at_m, w.updated_at_m, w.deleted_at_m,
   q.pk, q.workspace_id, q.requests_per_month, q.logs_retention_days, q.audit_logs_retention_days, q.team
FROM ` + "`" + `workspaces` + "`" + ` w
LEFT JOIN quota q ON w.id = q.workspace_id
WHERE w.id > ?
ORDER BY w.id ASC
LIMIT 100
`
```

```go
const listWorkspacesForQuotaCheck = `-- name: ListWorkspacesForQuotaCheck :many
SELECT
   w.id,
   w.org_id,
   w.name,
   w.stripe_customer_id,
   w.tier,
   w.enabled,
   q.requests_per_month
FROM ` + "`" + `workspaces` + "`" + ` w
LEFT JOIN quota q ON w.id = q.workspace_id
WHERE w.id > ?
ORDER BY w.id ASC
LIMIT 100
`
```

```go
const lockIdentityForUpdate = `-- name: LockIdentityForUpdate :one
SELECT id FROM identities
WHERE id = ?
FOR UPDATE
`
```

```go
const lockKeyForUpdate = `-- name: LockKeyForUpdate :one
SELECT id FROM ` + "`" + `keys` + "`" + `
WHERE id = ?
FOR UPDATE
`
```

```go
const reassignFrontlineRoute = `-- name: ReassignFrontlineRoute :exec
UPDATE frontline_routes
SET
  deployment_id = ?,
  updated_at = ?
WHERE id = ?
`
```

```go
const resetCustomDomainVerification = `-- name: ResetCustomDomainVerification :exec
UPDATE custom_domains
SET verification_status = ?,
    check_attempts = ?,
    verification_error = NULL,
    last_checked_at = NULL,
    invocation_id = ?,
    updated_at = ?
WHERE domain = ?
`
```

```go
const setWorkspaceK8sNamespace = `-- name: SetWorkspaceK8sNamespace :exec
UPDATE ` + "`" + `workspaces` + "`" + `
SET k8s_namespace = ?
WHERE id = ? AND k8s_namespace IS NULL
`
```

```go
const softDeleteApi = `-- name: SoftDeleteApi :exec
UPDATE apis
SET deleted_at_m = ?
WHERE id = ?
`
```

```go
const softDeleteIdentity = `-- name: SoftDeleteIdentity :exec
UPDATE identities
SET deleted = 1
WHERE id = ?
  AND workspace_id = ?
`
```

```go
const softDeleteKeyByID = `-- name: SoftDeleteKeyByID :exec
UPDATE ` + "`" + `keys` + "`" + ` SET deleted_at_m = ? WHERE id = ?
`
```

```go
const softDeleteManyKeysByKeySpaceID = `-- name: SoftDeleteManyKeysByKeySpaceID :exec
UPDATE ` + "`" + `keys` + "`" + `
SET deleted_at_m = ?
WHERE key_auth_id = ?
AND deleted_at_m IS NULL
`
```

```go
const softDeleteRatelimitNamespace = `-- name: SoftDeleteRatelimitNamespace :exec
UPDATE ` + "`" + `ratelimit_namespaces` + "`" + `
SET
    deleted_at_m =  ?
WHERE id = ?
`
```

```go
const softDeleteRatelimitOverride = `-- name: SoftDeleteRatelimitOverride :exec
UPDATE ` + "`" + `ratelimit_overrides` + "`" + `
SET
    deleted_at_m =  ?
WHERE id = ?
`
```

```go
const softDeleteWorkspace = `-- name: SoftDeleteWorkspace :execresult
UPDATE ` + "`" + `workspaces` + "`" + `
SET deleted_at_m = ?
WHERE id = ?
AND delete_protection = false
`
```

```go
const updateAcmeChallengePending = `-- name: UpdateAcmeChallengePending :exec
UPDATE acme_challenges
SET status = ?, token = ?, authorization = ?, updated_at = ?
WHERE domain_id = ?
`
```

```go
const updateAcmeChallengeStatus = `-- name: UpdateAcmeChallengeStatus :exec
UPDATE acme_challenges
SET status = ?, updated_at = ?
WHERE domain_id = ?
`
```

```go
const updateAcmeChallengeTryClaiming = `-- name: UpdateAcmeChallengeTryClaiming :exec
UPDATE acme_challenges
SET status = ?, updated_at = ?
WHERE domain_id = ? AND status = 'waiting'
`
```

```go
const updateAcmeChallengeVerifiedWithExpiry = `-- name: UpdateAcmeChallengeVerifiedWithExpiry :exec
UPDATE acme_challenges
SET status = ?, expires_at = ?, updated_at = ?
WHERE domain_id = ?
`
```

```go
const updateAcmeUserRegistrationURI = `-- name: UpdateAcmeUserRegistrationURI :exec
UPDATE acme_users SET registration_uri = ? WHERE id = ?
`
```

```go
const updateApiDeleteProtection = `-- name: UpdateApiDeleteProtection :exec
UPDATE apis
SET delete_protection = ?
WHERE id = ?
`
```

```go
const updateClickhouseWorkspaceSettingsLimits = `-- name: UpdateClickhouseWorkspaceSettingsLimits :exec
UPDATE ` + "`" + `clickhouse_workspace_settings` + "`" + `
SET
    quota_duration_seconds = ?,
    max_queries_per_window = ?,
    max_execution_time_per_window = ?,
    max_query_execution_time = ?,
    max_query_memory_bytes = ?,
    max_query_result_rows = ?,
    updated_at = ?
WHERE workspace_id = ?
`
```

```go
const updateCustomDomainCheckAttempt = `-- name: UpdateCustomDomainCheckAttempt :exec
UPDATE custom_domains
SET check_attempts = ?,
    last_checked_at = ?,
    updated_at = ?
WHERE id = ?
`
```

```go
const updateCustomDomainFailed = `-- name: UpdateCustomDomainFailed :exec
UPDATE custom_domains
SET verification_status = ?,
    verification_error = ?,
    updated_at = ?
WHERE id = ?
`
```

```go
const updateCustomDomainInvocationID = `-- name: UpdateCustomDomainInvocationID :exec
UPDATE custom_domains
SET invocation_id = ?,
    updated_at = ?
WHERE id = ?
`
```

```go
const updateCustomDomainOwnership = `-- name: UpdateCustomDomainOwnership :exec
UPDATE custom_domains
SET ownership_verified = ?, cname_verified = ?, updated_at = ?
WHERE id = ?
`
```

```go
const updateCustomDomainVerificationStatus = `-- name: UpdateCustomDomainVerificationStatus :exec
UPDATE custom_domains
SET verification_status = ?,
    updated_at = ?
WHERE id = ?
`
```

```go
const updateDeploymentBuildID = `-- name: UpdateDeploymentBuildID :exec
UPDATE deployments
SET build_id = ?, updated_at = ?
WHERE id = ?
`
```

```go
const updateDeploymentImage = `-- name: UpdateDeploymentImage :exec
UPDATE deployments
SET image = ?, updated_at = ?
WHERE id = ?
`
```

```go
const updateDeploymentOpenapiSpec = `-- name: UpdateDeploymentOpenapiSpec :exec
UPDATE deployments
SET openapi_spec = ?, updated_at = ?
WHERE id = ?
`
```

```go
const updateDeploymentStatus = `-- name: UpdateDeploymentStatus :exec
UPDATE deployments
SET status = ?, updated_at = ?
WHERE id = ?
`
```

```go
const updateFrontlineRouteDeploymentId = `-- name: UpdateFrontlineRouteDeploymentId :exec
UPDATE frontline_routes
SET deployment_id = ?
WHERE id = ?
`
```

```go
const updateIdentity = `-- name: UpdateIdentity :exec
UPDATE ` + "`" + `identities` + "`" + `
SET
    meta = CAST(? AS JSON),
    updated_at = NOW()
WHERE
    id = ?
`
```

```go
const updateKey = `-- name: UpdateKey :exec
UPDATE ` + "`" + `keys` + "`" + ` k SET
    name = CASE 
        WHEN CAST(? AS UNSIGNED) = 1 THEN ? 
        ELSE k.name 
    END,
    identity_id = CASE 
        WHEN CAST(? AS UNSIGNED) = 1 THEN ? 
        ELSE k.identity_id 
    END,
    enabled = CASE 
        WHEN CAST(? AS UNSIGNED) = 1 THEN ? 
        ELSE k.enabled 
    END,
    meta = CASE 
        WHEN CAST(? AS UNSIGNED) = 1 THEN ? 
        ELSE k.meta 
    END,
    expires = CASE 
        WHEN CAST(? AS UNSIGNED) = 1 THEN ? 
        ELSE k.expires 
    END,
    remaining_requests = CASE 
        WHEN CAST(? AS UNSIGNED) = 1 THEN ? 
        ELSE k.remaining_requests 
    END,
    refill_amount = CASE 
        WHEN CAST(? AS UNSIGNED) = 1 THEN ? 
        ELSE k.refill_amount 
    END,
    refill_day = CASE 
        WHEN CAST(? AS UNSIGNED) = 1 THEN ? 
        ELSE k.refill_day 
    END,
    updated_at_m = ?
WHERE id = ?
`
```

```go
const updateKeyCreditsDecrement = `-- name: UpdateKeyCreditsDecrement :exec
UPDATE ` + "`" + `keys` + "`" + `
SET remaining_requests = CASE
    WHEN remaining_requests >= ? THEN remaining_requests - ?
    ELSE 0
END
WHERE id = ?
`
```

```go
const updateKeyCreditsIncrement = `-- name: UpdateKeyCreditsIncrement :exec
UPDATE ` + "`" + `keys` + "`" + `
SET remaining_requests = remaining_requests + ?
WHERE id = ?
`
```

```go
const updateKeyCreditsRefill = `-- name: UpdateKeyCreditsRefill :exec
UPDATE ` + "`" + `keys` + "`" + ` SET refill_amount = ?, refill_day = ? WHERE id = ?
`
```

```go
const updateKeyCreditsSet = `-- name: UpdateKeyCreditsSet :exec
UPDATE ` + "`" + `keys` + "`" + `
SET remaining_requests = ?
WHERE id = ?
`
```

```go
const updateKeyHashAndMigration = `-- name: UpdateKeyHashAndMigration :exec
UPDATE ` + "`" + `keys` + "`" + `
SET
    hash = ?,
    pending_migration_id = ?,
    start = ?,
    updated_at_m = ?
WHERE id = ?
`
```

```go
const updateKeySpaceKeyEncryption = `-- name: UpdateKeySpaceKeyEncryption :exec
UPDATE ` + "`" + `key_auth` + "`" + ` SET store_encrypted_keys = ? WHERE id = ?
`
```

```go
const updateProjectDeployments = `-- name: UpdateProjectDeployments :exec
UPDATE projects
SET
  live_deployment_id = ?,
  is_rolled_back = ?,
  updated_at = ?
WHERE id = ?
`
```

```go
const updateProjectDepotID = `-- name: UpdateProjectDepotID :exec
UPDATE projects
SET
    depot_project_id = ?,
    updated_at = ?
WHERE id = ?
`
```

```go
const updateRatelimit = `-- name: UpdateRatelimit :exec
UPDATE ` + "`" + `ratelimits` + "`" + `
SET
    name = ?,
    ` + "`" + `limit` + "`" + ` = ?,
    duration = ?,
    auto_apply = ?,
    updated_at = NOW()
WHERE
    id = ?
`
```

```go
const updateRatelimitOverride = `-- name: UpdateRatelimitOverride :execresult
UPDATE ` + "`" + `ratelimit_overrides` + "`" + `
SET
    ` + "`" + `limit` + "`" + ` = ?,
    duration = ?,
    async = ?,
    updated_at_m= ?
WHERE id = ?
`
```

```go
const updateSentinelAvailableReplicasAndHealth = `-- name: UpdateSentinelAvailableReplicasAndHealth :exec
UPDATE sentinels SET
available_replicas = ?,
health = ?,
updated_at = ?
WHERE k8s_name = ?
`
```

```go
const updateWorkspaceEnabled = `-- name: UpdateWorkspaceEnabled :execresult
UPDATE ` + "`" + `workspaces` + "`" + `
SET enabled = ?
WHERE id = ?
`
```

```go
const updateWorkspacePlan = `-- name: UpdateWorkspacePlan :exec
UPDATE ` + "`" + `workspaces` + "`" + `
SET plan = ?
WHERE id = ?
`
```

```go
const upsertCustomDomain = `-- name: UpsertCustomDomain :exec
INSERT INTO custom_domains (
    id, workspace_id, project_id, environment_id, domain,
    challenge_type, verification_status, verification_token, target_cname, created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    project_id = VALUES(project_id),
    environment_id = VALUES(environment_id),
    challenge_type = VALUES(challenge_type),
    verification_status = VALUES(verification_status),
    target_cname = VALUES(target_cname),
    updated_at = ?
`
```

```go
const upsertEnvironment = `-- name: UpsertEnvironment :exec
INSERT INTO environments (
    id,
    workspace_id,
    project_id,
    slug,
    sentinel_config,
    created_at
) VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE slug = VALUES(slug)
`
```

```go
const upsertIdentity = `-- name: UpsertIdentity :exec
INSERT INTO ` + "`" + `identities` + "`" + ` (
    id,
    external_id,
    workspace_id,
    environment,
    created_at,
    meta
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    CAST(? AS JSON)
)
ON DUPLICATE KEY UPDATE external_id = external_id
`
```

```go
const upsertInstance = `-- name: UpsertInstance :exec
INSERT INTO instances (
	id,
	deployment_id,
	workspace_id,
	project_id,
	region,
	k8s_name,
	address,
	cpu_millicores,
	memory_mib,
	status
)
VALUES (
	?,
	?,
	?,
	?,
	?,
	?,
	?,
	?,
	?,
	?
)
ON DUPLICATE KEY UPDATE
	address = ?,
	cpu_millicores = ?,
	memory_mib = ?,
	status = ?
`
```

```go
const upsertKeySpace = `-- name: UpsertKeySpace :exec
INSERT INTO key_auth (
    id,
    workspace_id,
    created_at_m,
    default_prefix,
    default_bytes,
    store_encrypted_keys
) VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    store_encrypted_keys = VALUES(store_encrypted_keys)
`
```

```go
const upsertQuota = `-- name: UpsertQuota :exec
INSERT INTO quota (
    workspace_id,
    requests_per_month,
    audit_logs_retention_days,
    logs_retention_days,
    team
) VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    requests_per_month = VALUES(requests_per_month),
    audit_logs_retention_days = VALUES(audit_logs_retention_days),
    logs_retention_days = VALUES(logs_retention_days)
`
```

```go
const upsertWorkspace = `-- name: UpsertWorkspace :exec
INSERT INTO workspaces (
    id,
    org_id,
    name,
    slug,
    created_at_m,
    tier,
    beta_features,
    features,
    enabled,
    delete_protection
) VALUES (?, ?, ?, ?, ?, ?, ?, '{}', true, false)
ON DUPLICATE KEY UPDATE
    beta_features = VALUES(beta_features),
    name = VALUES(name)
`
```


## Functions

### func IsConnectionError

```go
func IsConnectionError(err error) bool
```

IsConnectionError returns true if the error indicates a connection problem. This includes server gone (2006), server lost (2013), and network errors. These are transient errors that may resolve on retry with a new connection.

### func IsDeadlockError

```go
func IsDeadlockError(err error) bool
```

IsDeadlockError returns true if the error is a MySQL deadlock error (1213). Deadlocks cause InnoDB to roll back the entire transaction. The entire transaction should be retried.

### func IsDuplicateKeyError

```go
func IsDuplicateKeyError(err error) bool
```

### func IsLockWaitTimeoutError

```go
func IsLockWaitTimeoutError(err error) bool
```

IsLockWaitTimeoutError returns true if the error is a MySQL lock wait timeout error (1205). By default, only the current statement is rolled back (not the entire transaction). See: [https://dev.mysql.com/doc/refman/8.0/en/innodb-error-handling.html](https://dev.mysql.com/doc/refman/8.0/en/innodb-error-handling.html)

### func IsNotFound

```go
func IsNotFound(err error) bool
```

IsNotFound returns true if the error is sql.ErrNoRows. Use this for consistent not-found handling across the codebase.

### func IsTooManyConnectionsError

```go
func IsTooManyConnectionsError(err error) bool
```

IsTooManyConnectionsError returns true if the error is a MySQL too many connections error (1040). This is a transient error that may resolve when other connections are closed.

### func IsTransientError

```go
func IsTransientError(err error) bool
```

IsTransientError returns true if the error is a transient MySQL error that should be retried. This includes deadlocks, lock wait timeouts, connection errors, and too many connections.

### func Tx

```go
func Tx(ctx context.Context, db *Replica, fn func(context.Context, DBTX) error) error
```

Tx executes fn within a database transaction without returning a result. It is a convenience wrapper around \[TxWithResult] for operations that only need error handling.

Tx begins a transaction on db, executes fn with the transaction context, and commits on success or rolls back on failure. All database errors are wrapped with ServiceUnavailable fault codes.

The ctx parameter provides cancellation and timeout control. The db parameter must be a valid \[Replica] instance. The fn parameter receives the transaction context and a \[DBTX] interface for database operations.

Tx returns nil on successful commit, or an error if any step fails. Error handling follows the same patterns as \[TxWithResult].

Use Tx for operations that don't need to return values, such as:

  - Deleting records with audit logging
  - Updating configuration settings
  - Batch cleanup operations
  - State changes that only need success/failure indication

Example batch deletion with audit:

	err := db.Tx(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		// Delete expired keys
		deletedCount, err := db.Query.DeleteExpiredKeys(ctx, tx, time.Now())
		if err != nil {
			return fmt.Errorf("failed to delete expired keys: %w", err)
		}

		// Log the cleanup operation
		err = db.Query.InsertAuditLog(ctx, tx, db.InsertAuditLogParams{
			Action:  "cleanup_expired_keys",
			Details: fmt.Sprintf("deleted %d expired keys", deletedCount),
		})
		if err != nil {
			return fmt.Errorf("failed to log cleanup: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("cleanup transaction failed: %w", err)
	}

See \[TxWithResult] for detailed transaction behavior and \[DBTX] for available database operations.

### func TxRetry

```go
func TxRetry(ctx context.Context, db *Replica, fn func(context.Context, DBTX) error) error
```

TxRetry executes a transaction with automatic retry on transient errors like deadlocks. It is a convenience wrapper around TxWithResultRetry for operations that don't return a value.

Usage:

	err := db.TxRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		// Perform transactional operations
		return nil
	})

### func TxWithResult

```go
func TxWithResult[T any](ctx context.Context, db *Replica, fn func(context.Context, DBTX) (T, error)) (T, error)
```

TxWithResult executes fn within a database transaction and returns the result. It begins a transaction on db, executes fn with the transaction context, and commits on success or rolls back on failure.

The function automatically handles the complete transaction lifecycle: begin, execute, and commit/rollback. All database errors are wrapped with ServiceUnavailable fault codes for consistent error handling.

TxWithResult is generic and preserves type safety for return values. The ctx parameter provides cancellation and timeout control for the entire transaction. The db parameter must be a valid \[Replica] instance, typically from \[Database.RW] for write operations.

The fn parameter receives the transaction context and a \[DBTX] interface for database operations. It should perform all required operations and return the result with any error.

TxWithResult returns the function result on successful commit, or an error if any step fails. Transaction begin errors return ServiceUnavailable. Rollback errors during error handling also return ServiceUnavailable, except for sql.ErrTxDone which indicates the transaction was already completed. Commit errors return ServiceUnavailable.

Context cancellation triggers automatic rollback. The function is safe for concurrent use but callers must avoid operations that could deadlock with other concurrent transactions.

Common usage scenarios include:

  - Creating API keys with associated permissions atomically
  - Updating workspace settings with audit trail creation
  - Batch operations that must succeed or fail as a unit
  - Complex queries requiring consistency guarantees

Edge cases and limitations:

  - If fn returns an error, rollback is attempted even if the transaction is already in a failed state, which may produce additional errors
  - Database connection issues during commit may leave the transaction in an undefined state on the server side
  - Context cancellation after fn execution causes rollback instead of commit

Anti-patterns to avoid:

  - Long-running operations within fn that could timeout
  - Nesting calls to TxWithResult (creates nested transactions)
  - Ignoring the returned error from fn
  - Accessing the DBTX parameter outside of the fn callback

Use context.WithTimeout to prevent indefinite blocking. For operations that may conflict, implement retry logic with exponential backoff at the caller level.

Example atomic key creation with permissions:

	result, err := db.TxWithResult(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) (*models.Key, error) {
		key, err := db.Query.InsertKey(ctx, tx, db.InsertKeyParams{
			ID:        keyID,
			KeyAuthID: keyAuthID,
			Hash:      hashedKey,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to insert key: %w", err)
		}

		for _, permissionID := range permissionIDs {
			err = db.Query.InsertKeyPermission(ctx, tx, db.InsertKeyPermissionParams{
				KeyID:        key.ID,
				PermissionID: permissionID,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add permission: %w", err)
			}
		}

		return &key, nil
	})
	if err != nil {
		return fmt.Errorf("key creation transaction failed: %w", err)
	}

Example with timeout handling:

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := db.TxWithResult(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) (*SomeResult, error) {
		// Context cancellation is automatically handled
		// Perform database operations...
		return &SomeResult{}, nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

See \[Replica.Begin] for transaction initiation and \[DBTX] for available operations within transactions. For read-only operations that don't require transactions, use query methods directly on \[Database.RO].

### func TxWithResultRetry

```go
func TxWithResultRetry[T any](ctx context.Context, db *Replica, fn func(context.Context, DBTX) (T, error)) (T, error)
```

TxWithResultRetry executes a transaction with automatic retry on transient errors. It wraps TxWithResult with retry logic, retrying the entire transaction (begin -> fn -> commit) on retryable errors.

This is useful for transactions that may encounter transient errors due to concurrent access patterns or temporary resource constraints. When such errors occur, MySQL rolls back the transaction, so we retry from the beginning.

Configuration:

  - 3 attempts maximum
  - Exponential backoff: 50ms, 100ms, 200ms

Retries on transient errors:

  - Deadlocks (MySQL error 1213)
  - Lock wait timeouts (MySQL error 1205)
  - Connection errors (MySQL errors 2006, 2013, network errors)
  - Too many connections (MySQL error 1040)

Does NOT retry on permanent errors:

  - Not found errors
  - Duplicate key errors (MySQL error 1062)

Usage:

	result, err := db.TxWithResultRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) (*Result, error) {
		// Perform transactional operations
		return &Result{}, nil
	})

### func UnmarshalNullableJSONTo

```go
func UnmarshalNullableJSONTo[T any](data any) (T, error)
```

UnmarshalNullableJSONTo unmarshals JSON data from database columns into Go types. It handles the common pattern where database queries return JSON as \[]byte that needs to be deserialized into structs, slices, or maps.

The function accepts 'any' type because database drivers return interface{} for JSON columns, even though the underlying value is typically \[]byte.

Returns:

  - (T, nil) on successful unmarshal
  - (zero, nil) if data is nil or empty \[]byte (these are valid null/empty states)
  - (zero, error) if type assertion fails or JSON unmarshal fails

Example usage:

	roles, err := UnmarshalNullableJSONTo[[]RoleInfo](row.Roles)
	if err != nil {
	    logger.Error("failed to unmarshal roles", "error", err)
	    return err
	}

### func WithRetryContext

```go
func WithRetryContext[T any](ctx context.Context, fn func() (T, error)) (T, error)
```

WithRetryContext executes a database operation with optimized retry configuration while respecting context cancellation and deadlines. It retries transient errors with exponential backoff but skips non-retryable errors like "not found" or "duplicate key" to avoid unnecessary delays.

Context behavior:

  - Returns immediately if context is already cancelled or deadline exceeded
  - Detects context cancellation during backoff sleep without waiting for full duration
  - Returns context.Canceled or context.DeadlineExceeded on context errors

Configuration:

  - 3 attempts maximum
  - Exponential backoff: 50ms, 100ms, 200ms
  - Skips retries for "not found" and "duplicate key" errors

Usage:

	result, err := db.WithRetryContext(ctx, func() (SomeType, error) {
		return db.Query.SomeOperation(ctx, db.RO(), params)
	})


## Types

### type AcmeChallenge

```go
type AcmeChallenge struct {
	Pk            uint64                      `db:"pk"`
	DomainID      string                      `db:"domain_id"`
	WorkspaceID   string                      `db:"workspace_id"`
	Token         string                      `db:"token"`
	ChallengeType AcmeChallengesChallengeType `db:"challenge_type"`
	Authorization string                      `db:"authorization"`
	Status        AcmeChallengesStatus        `db:"status"`
	ExpiresAt     int64                       `db:"expires_at"`
	CreatedAt     int64                       `db:"created_at"`
	UpdatedAt     sql.NullInt64               `db:"updated_at"`
}
```

### type AcmeChallengesChallengeType

```go
type AcmeChallengesChallengeType string
```

```go
const (
	AcmeChallengesChallengeTypeHTTP01 AcmeChallengesChallengeType = "HTTP-01"
	AcmeChallengesChallengeTypeDNS01  AcmeChallengesChallengeType = "DNS-01"
)
```

#### func (AcmeChallengesChallengeType) Scan

```go
func (e *AcmeChallengesChallengeType) Scan(src interface{}) error
```

### type AcmeChallengesStatus

```go
type AcmeChallengesStatus string
```

```go
const (
	AcmeChallengesStatusWaiting  AcmeChallengesStatus = "waiting"
	AcmeChallengesStatusPending  AcmeChallengesStatus = "pending"
	AcmeChallengesStatusVerified AcmeChallengesStatus = "verified"
	AcmeChallengesStatusFailed   AcmeChallengesStatus = "failed"
)
```

#### func (AcmeChallengesStatus) Scan

```go
func (e *AcmeChallengesStatus) Scan(src interface{}) error
```

### type AcmeUser

```go
type AcmeUser struct {
	Pk              uint64         `db:"pk"`
	ID              string         `db:"id"`
	WorkspaceID     string         `db:"workspace_id"`
	EncryptedKey    string         `db:"encrypted_key"`
	RegistrationUri sql.NullString `db:"registration_uri"`
	CreatedAt       int64          `db:"created_at"`
	UpdatedAt       sql.NullInt64  `db:"updated_at"`
}
```

### type Api

```go
type Api struct {
	Pk               uint64           `db:"pk"`
	ID               string           `db:"id"`
	Name             string           `db:"name"`
	WorkspaceID      string           `db:"workspace_id"`
	IpWhitelist      sql.NullString   `db:"ip_whitelist"`
	AuthType         NullApisAuthType `db:"auth_type"`
	KeyAuthID        sql.NullString   `db:"key_auth_id"`
	CreatedAtM       int64            `db:"created_at_m"`
	UpdatedAtM       sql.NullInt64    `db:"updated_at_m"`
	DeletedAtM       sql.NullInt64    `db:"deleted_at_m"`
	DeleteProtection sql.NullBool     `db:"delete_protection"`
}
```

### type ApisAuthType

```go
type ApisAuthType string
```

```go
const (
	ApisAuthTypeKey ApisAuthType = "key"
	ApisAuthTypeJwt ApisAuthType = "jwt"
)
```

#### func (ApisAuthType) Scan

```go
func (e *ApisAuthType) Scan(src interface{}) error
```

### type AuditLog

```go
type AuditLog struct {
	Pk          uint64         `db:"pk"`
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	Bucket      string         `db:"bucket"`
	BucketID    string         `db:"bucket_id"`
	Event       string         `db:"event"`
	Time        int64          `db:"time"`
	Display     string         `db:"display"`
	RemoteIp    sql.NullString `db:"remote_ip"`
	UserAgent   sql.NullString `db:"user_agent"`
	ActorType   string         `db:"actor_type"`
	ActorID     string         `db:"actor_id"`
	ActorName   sql.NullString `db:"actor_name"`
	ActorMeta   []byte         `db:"actor_meta"`
	CreatedAt   int64          `db:"created_at"`
	UpdatedAt   sql.NullInt64  `db:"updated_at"`
}
```

### type AuditLogBucket

```go
type AuditLogBucket struct {
	Pk               uint64        `db:"pk"`
	ID               string        `db:"id"`
	WorkspaceID      string        `db:"workspace_id"`
	Name             string        `db:"name"`
	RetentionDays    sql.NullInt32 `db:"retention_days"`
	CreatedAt        int64         `db:"created_at"`
	UpdatedAt        sql.NullInt64 `db:"updated_at"`
	DeleteProtection sql.NullBool  `db:"delete_protection"`
}
```

### type AuditLogTarget

```go
type AuditLogTarget struct {
	Pk          uint64         `db:"pk"`
	WorkspaceID string         `db:"workspace_id"`
	BucketID    string         `db:"bucket_id"`
	Bucket      string         `db:"bucket"`
	AuditLogID  string         `db:"audit_log_id"`
	DisplayName string         `db:"display_name"`
	Type        string         `db:"type"`
	ID          string         `db:"id"`
	Name        sql.NullString `db:"name"`
	Meta        []byte         `db:"meta"`
	CreatedAt   int64          `db:"created_at"`
	UpdatedAt   sql.NullInt64  `db:"updated_at"`
}
```

### type BulkQuerier

```go
type BulkQuerier interface {
	InsertAcmeChallenges(ctx context.Context, db DBTX, args []InsertAcmeChallengeParams) error
	InsertAcmeUsers(ctx context.Context, db DBTX, args []InsertAcmeUserParams) error
	InsertApis(ctx context.Context, db DBTX, args []InsertApiParams) error
	InsertAuditLogs(ctx context.Context, db DBTX, args []InsertAuditLogParams) error
	InsertAuditLogTargets(ctx context.Context, db DBTX, args []InsertAuditLogTargetParams) error
	InsertCertificates(ctx context.Context, db DBTX, args []InsertCertificateParams) error
	InsertCiliumNetworkPolicies(ctx context.Context, db DBTX, args []InsertCiliumNetworkPolicyParams) error
	InsertClickhouseWorkspaceSettingses(ctx context.Context, db DBTX, args []InsertClickhouseWorkspaceSettingsParams) error
	InsertCustomDomains(ctx context.Context, db DBTX, args []InsertCustomDomainParams) error
	UpsertCustomDomain(ctx context.Context, db DBTX, args []UpsertCustomDomainParams) error
	InsertDeployments(ctx context.Context, db DBTX, args []InsertDeploymentParams) error
	InsertDeploymentTopologies(ctx context.Context, db DBTX, args []InsertDeploymentTopologyParams) error
	InsertEnvironments(ctx context.Context, db DBTX, args []InsertEnvironmentParams) error
	UpsertEnvironment(ctx context.Context, db DBTX, args []UpsertEnvironmentParams) error
	InsertGithubRepoConnections(ctx context.Context, db DBTX, args []InsertGithubRepoConnectionParams) error
	InsertIdentities(ctx context.Context, db DBTX, args []InsertIdentityParams) error
	InsertIdentityRatelimits(ctx context.Context, db DBTX, args []InsertIdentityRatelimitParams) error
	UpsertIdentity(ctx context.Context, db DBTX, args []UpsertIdentityParams) error
	InsertFrontlineRoutes(ctx context.Context, db DBTX, args []InsertFrontlineRouteParams) error
	UpsertInstance(ctx context.Context, db DBTX, args []UpsertInstanceParams) error
	InsertKeyAuths(ctx context.Context, db DBTX, args []InsertKeyAuthParams) error
	InsertKeyEncryptions(ctx context.Context, db DBTX, args []InsertKeyEncryptionParams) error
	InsertKeys(ctx context.Context, db DBTX, args []InsertKeyParams) error
	InsertKeyRatelimits(ctx context.Context, db DBTX, args []InsertKeyRatelimitParams) error
	InsertKeyMigrations(ctx context.Context, db DBTX, args []InsertKeyMigrationParams) error
	InsertKeyPermissions(ctx context.Context, db DBTX, args []InsertKeyPermissionParams) error
	InsertKeyRoles(ctx context.Context, db DBTX, args []InsertKeyRoleParams) error
	InsertKeySpaces(ctx context.Context, db DBTX, args []InsertKeySpaceParams) error
	UpsertKeySpace(ctx context.Context, db DBTX, args []UpsertKeySpaceParams) error
	InsertPermissions(ctx context.Context, db DBTX, args []InsertPermissionParams) error
	InsertProjects(ctx context.Context, db DBTX, args []InsertProjectParams) error
	UpsertQuota(ctx context.Context, db DBTX, args []UpsertQuotaParams) error
	InsertRatelimitNamespaces(ctx context.Context, db DBTX, args []InsertRatelimitNamespaceParams) error
	InsertRatelimitOverrides(ctx context.Context, db DBTX, args []InsertRatelimitOverrideParams) error
	InsertRoles(ctx context.Context, db DBTX, args []InsertRoleParams) error
	InsertRolePermissions(ctx context.Context, db DBTX, args []InsertRolePermissionParams) error
	InsertSentinels(ctx context.Context, db DBTX, args []InsertSentinelParams) error
	InsertWorkspaces(ctx context.Context, db DBTX, args []InsertWorkspaceParams) error
	UpsertWorkspace(ctx context.Context, db DBTX, args []UpsertWorkspaceParams) error
}
```

BulkQuerier contains bulk insert methods.

### type BulkQueries

```go
type BulkQueries struct{}
```

#### func (BulkQueries) InsertAcmeChallenges

```go
func (q *BulkQueries) InsertAcmeChallenges(ctx context.Context, db DBTX, args []InsertAcmeChallengeParams) error
```

InsertAcmeChallenges performs bulk insert in a single query

#### func (BulkQueries) InsertAcmeUsers

```go
func (q *BulkQueries) InsertAcmeUsers(ctx context.Context, db DBTX, args []InsertAcmeUserParams) error
```

InsertAcmeUsers performs bulk insert in a single query

#### func (BulkQueries) InsertApis

```go
func (q *BulkQueries) InsertApis(ctx context.Context, db DBTX, args []InsertApiParams) error
```

InsertApis performs bulk insert in a single query

#### func (BulkQueries) InsertAuditLogTargets

```go
func (q *BulkQueries) InsertAuditLogTargets(ctx context.Context, db DBTX, args []InsertAuditLogTargetParams) error
```

InsertAuditLogTargets performs bulk insert in a single query

#### func (BulkQueries) InsertAuditLogs

```go
func (q *BulkQueries) InsertAuditLogs(ctx context.Context, db DBTX, args []InsertAuditLogParams) error
```

InsertAuditLogs performs bulk insert in a single query

#### func (BulkQueries) InsertCertificates

```go
func (q *BulkQueries) InsertCertificates(ctx context.Context, db DBTX, args []InsertCertificateParams) error
```

InsertCertificates performs bulk insert in a single query

#### func (BulkQueries) InsertCiliumNetworkPolicies

```go
func (q *BulkQueries) InsertCiliumNetworkPolicies(ctx context.Context, db DBTX, args []InsertCiliumNetworkPolicyParams) error
```

InsertCiliumNetworkPolicies performs bulk insert in a single query

#### func (BulkQueries) InsertClickhouseWorkspaceSettingses

```go
func (q *BulkQueries) InsertClickhouseWorkspaceSettingses(ctx context.Context, db DBTX, args []InsertClickhouseWorkspaceSettingsParams) error
```

InsertClickhouseWorkspaceSettingses performs bulk insert in a single query

#### func (BulkQueries) InsertCustomDomains

```go
func (q *BulkQueries) InsertCustomDomains(ctx context.Context, db DBTX, args []InsertCustomDomainParams) error
```

InsertCustomDomains performs bulk insert in a single query

#### func (BulkQueries) InsertDeploymentTopologies

```go
func (q *BulkQueries) InsertDeploymentTopologies(ctx context.Context, db DBTX, args []InsertDeploymentTopologyParams) error
```

InsertDeploymentTopologies performs bulk insert in a single query

#### func (BulkQueries) InsertDeployments

```go
func (q *BulkQueries) InsertDeployments(ctx context.Context, db DBTX, args []InsertDeploymentParams) error
```

InsertDeployments performs bulk insert in a single query

#### func (BulkQueries) InsertEnvironments

```go
func (q *BulkQueries) InsertEnvironments(ctx context.Context, db DBTX, args []InsertEnvironmentParams) error
```

InsertEnvironments performs bulk insert in a single query

#### func (BulkQueries) InsertFrontlineRoutes

```go
func (q *BulkQueries) InsertFrontlineRoutes(ctx context.Context, db DBTX, args []InsertFrontlineRouteParams) error
```

InsertFrontlineRoutes performs bulk insert in a single query

#### func (BulkQueries) InsertGithubRepoConnections

```go
func (q *BulkQueries) InsertGithubRepoConnections(ctx context.Context, db DBTX, args []InsertGithubRepoConnectionParams) error
```

InsertGithubRepoConnections performs bulk insert in a single query

#### func (BulkQueries) InsertIdentities

```go
func (q *BulkQueries) InsertIdentities(ctx context.Context, db DBTX, args []InsertIdentityParams) error
```

InsertIdentities performs bulk insert in a single query

#### func (BulkQueries) InsertIdentityRatelimits

```go
func (q *BulkQueries) InsertIdentityRatelimits(ctx context.Context, db DBTX, args []InsertIdentityRatelimitParams) error
```

InsertIdentityRatelimits performs bulk insert in a single query

#### func (BulkQueries) InsertKeyAuths

```go
func (q *BulkQueries) InsertKeyAuths(ctx context.Context, db DBTX, args []InsertKeyAuthParams) error
```

InsertKeyAuths performs bulk insert in a single query

#### func (BulkQueries) InsertKeyEncryptions

```go
func (q *BulkQueries) InsertKeyEncryptions(ctx context.Context, db DBTX, args []InsertKeyEncryptionParams) error
```

InsertKeyEncryptions performs bulk insert in a single query

#### func (BulkQueries) InsertKeyMigrations

```go
func (q *BulkQueries) InsertKeyMigrations(ctx context.Context, db DBTX, args []InsertKeyMigrationParams) error
```

InsertKeyMigrations performs bulk insert in a single query

#### func (BulkQueries) InsertKeyPermissions

```go
func (q *BulkQueries) InsertKeyPermissions(ctx context.Context, db DBTX, args []InsertKeyPermissionParams) error
```

InsertKeyPermissions performs bulk insert in a single query

#### func (BulkQueries) InsertKeyRatelimits

```go
func (q *BulkQueries) InsertKeyRatelimits(ctx context.Context, db DBTX, args []InsertKeyRatelimitParams) error
```

InsertKeyRatelimits performs bulk insert in a single query

#### func (BulkQueries) InsertKeyRoles

```go
func (q *BulkQueries) InsertKeyRoles(ctx context.Context, db DBTX, args []InsertKeyRoleParams) error
```

InsertKeyRoles performs bulk insert in a single query

#### func (BulkQueries) InsertKeySpaces

```go
func (q *BulkQueries) InsertKeySpaces(ctx context.Context, db DBTX, args []InsertKeySpaceParams) error
```

InsertKeySpaces performs bulk insert in a single query

#### func (BulkQueries) InsertKeys

```go
func (q *BulkQueries) InsertKeys(ctx context.Context, db DBTX, args []InsertKeyParams) error
```

InsertKeys performs bulk insert in a single query

#### func (BulkQueries) InsertPermissions

```go
func (q *BulkQueries) InsertPermissions(ctx context.Context, db DBTX, args []InsertPermissionParams) error
```

InsertPermissions performs bulk insert in a single query

#### func (BulkQueries) InsertProjects

```go
func (q *BulkQueries) InsertProjects(ctx context.Context, db DBTX, args []InsertProjectParams) error
```

InsertProjects performs bulk insert in a single query

#### func (BulkQueries) InsertRatelimitNamespaces

```go
func (q *BulkQueries) InsertRatelimitNamespaces(ctx context.Context, db DBTX, args []InsertRatelimitNamespaceParams) error
```

InsertRatelimitNamespaces performs bulk insert in a single query

#### func (BulkQueries) InsertRatelimitOverrides

```go
func (q *BulkQueries) InsertRatelimitOverrides(ctx context.Context, db DBTX, args []InsertRatelimitOverrideParams) error
```

InsertRatelimitOverrides performs bulk insert in a single query

#### func (BulkQueries) InsertRolePermissions

```go
func (q *BulkQueries) InsertRolePermissions(ctx context.Context, db DBTX, args []InsertRolePermissionParams) error
```

InsertRolePermissions performs bulk insert in a single query

#### func (BulkQueries) InsertRoles

```go
func (q *BulkQueries) InsertRoles(ctx context.Context, db DBTX, args []InsertRoleParams) error
```

InsertRoles performs bulk insert in a single query

#### func (BulkQueries) InsertSentinels

```go
func (q *BulkQueries) InsertSentinels(ctx context.Context, db DBTX, args []InsertSentinelParams) error
```

InsertSentinels performs bulk insert in a single query

#### func (BulkQueries) InsertWorkspaces

```go
func (q *BulkQueries) InsertWorkspaces(ctx context.Context, db DBTX, args []InsertWorkspaceParams) error
```

InsertWorkspaces performs bulk insert in a single query

#### func (BulkQueries) UpsertCustomDomain

```go
func (q *BulkQueries) UpsertCustomDomain(ctx context.Context, db DBTX, args []UpsertCustomDomainParams) error
```

UpsertCustomDomain performs bulk insert in a single query

#### func (BulkQueries) UpsertEnvironment

```go
func (q *BulkQueries) UpsertEnvironment(ctx context.Context, db DBTX, args []UpsertEnvironmentParams) error
```

UpsertEnvironment performs bulk insert in a single query

#### func (BulkQueries) UpsertIdentity

```go
func (q *BulkQueries) UpsertIdentity(ctx context.Context, db DBTX, args []UpsertIdentityParams) error
```

UpsertIdentity performs bulk insert in a single query

#### func (BulkQueries) UpsertInstance

```go
func (q *BulkQueries) UpsertInstance(ctx context.Context, db DBTX, args []UpsertInstanceParams) error
```

UpsertInstance performs bulk insert in a single query

#### func (BulkQueries) UpsertKeySpace

```go
func (q *BulkQueries) UpsertKeySpace(ctx context.Context, db DBTX, args []UpsertKeySpaceParams) error
```

UpsertKeySpace performs bulk insert in a single query

#### func (BulkQueries) UpsertQuota

```go
func (q *BulkQueries) UpsertQuota(ctx context.Context, db DBTX, args []UpsertQuotaParams) error
```

UpsertQuota performs bulk insert in a single query

#### func (BulkQueries) UpsertWorkspace

```go
func (q *BulkQueries) UpsertWorkspace(ctx context.Context, db DBTX, args []UpsertWorkspaceParams) error
```

UpsertWorkspace performs bulk insert in a single query

### type CachedKeyData

```go
type CachedKeyData struct {
	FindKeyForVerificationRow
	ParsedIPWhitelist map[string]struct{} // Pre-parsed IP addresses for O(1) lookup
}
```

CachedKeyData embeds FindKeyForVerificationRow and adds pre-processed data for caching. This struct is stored in the cache to avoid redundant parsing operations.

### type Certificate

```go
type Certificate struct {
	Pk                  uint64        `db:"pk"`
	ID                  string        `db:"id"`
	WorkspaceID         string        `db:"workspace_id"`
	Hostname            string        `db:"hostname"`
	Certificate         string        `db:"certificate"`
	EncryptedPrivateKey string        `db:"encrypted_private_key"`
	CreatedAt           int64         `db:"created_at"`
	UpdatedAt           sql.NullInt64 `db:"updated_at"`
}
```

### type CiliumNetworkPolicy

```go
type CiliumNetworkPolicy struct {
	Pk            uint64          `db:"pk"`
	ID            string          `db:"id"`
	WorkspaceID   string          `db:"workspace_id"`
	ProjectID     string          `db:"project_id"`
	EnvironmentID string          `db:"environment_id"`
	K8sName       string          `db:"k8s_name"`
	Region        string          `db:"region"`
	Policy        json.RawMessage `db:"policy"`
	Version       uint64          `db:"version"`
	CreatedAt     int64           `db:"created_at"`
	UpdatedAt     sql.NullInt64   `db:"updated_at"`
}
```

### type ClearAcmeChallengeTokensParams

```go
type ClearAcmeChallengeTokensParams struct {
	Token         string        `db:"token"`
	Authorization string        `db:"authorization"`
	UpdatedAt     sql.NullInt64 `db:"updated_at"`
	DomainID      string        `db:"domain_id"`
}
```

### type ClickhouseWorkspaceSetting

```go
type ClickhouseWorkspaceSetting struct {
	Pk                        uint64        `db:"pk"`
	WorkspaceID               string        `db:"workspace_id"`
	Username                  string        `db:"username"`
	PasswordEncrypted         string        `db:"password_encrypted"`
	QuotaDurationSeconds      int32         `db:"quota_duration_seconds"`
	MaxQueriesPerWindow       int32         `db:"max_queries_per_window"`
	MaxExecutionTimePerWindow int32         `db:"max_execution_time_per_window"`
	MaxQueryExecutionTime     int32         `db:"max_query_execution_time"`
	MaxQueryMemoryBytes       int64         `db:"max_query_memory_bytes"`
	MaxQueryResultRows        int32         `db:"max_query_result_rows"`
	CreatedAt                 int64         `db:"created_at"`
	UpdatedAt                 sql.NullInt64 `db:"updated_at"`
}
```

### type Config

```go
type Config struct {
	// The primary DSN for your database. This must support both reads and writes.
	PrimaryDSN string

	// The readonly replica will be used for most read queries.
	// If omitted, the primary is used.
	ReadOnlyDSN string
}
```

Config defines the parameters needed to establish database connections. It supports separate connections for read and write operations to allow for primary/replica setups.

### type CustomDomain

```go
type CustomDomain struct {
	Pk                 uint64                          `db:"pk"`
	ID                 string                          `db:"id"`
	WorkspaceID        string                          `db:"workspace_id"`
	ProjectID          string                          `db:"project_id"`
	EnvironmentID      string                          `db:"environment_id"`
	Domain             string                          `db:"domain"`
	ChallengeType      CustomDomainsChallengeType      `db:"challenge_type"`
	VerificationStatus CustomDomainsVerificationStatus `db:"verification_status"`
	VerificationToken  string                          `db:"verification_token"`
	OwnershipVerified  bool                            `db:"ownership_verified"`
	CnameVerified      bool                            `db:"cname_verified"`
	TargetCname        string                          `db:"target_cname"`
	LastCheckedAt      sql.NullInt64                   `db:"last_checked_at"`
	CheckAttempts      int32                           `db:"check_attempts"`
	VerificationError  sql.NullString                  `db:"verification_error"`
	InvocationID       sql.NullString                  `db:"invocation_id"`
	CreatedAt          int64                           `db:"created_at"`
	UpdatedAt          sql.NullInt64                   `db:"updated_at"`
}
```

### type CustomDomainsChallengeType

```go
type CustomDomainsChallengeType string
```

```go
const (
	CustomDomainsChallengeTypeHTTP01 CustomDomainsChallengeType = "HTTP-01"
	CustomDomainsChallengeTypeDNS01  CustomDomainsChallengeType = "DNS-01"
)
```

#### func (CustomDomainsChallengeType) Scan

```go
func (e *CustomDomainsChallengeType) Scan(src interface{}) error
```

### type CustomDomainsVerificationStatus

```go
type CustomDomainsVerificationStatus string
```

```go
const (
	CustomDomainsVerificationStatusPending   CustomDomainsVerificationStatus = "pending"
	CustomDomainsVerificationStatusVerifying CustomDomainsVerificationStatus = "verifying"
	CustomDomainsVerificationStatusVerified  CustomDomainsVerificationStatus = "verified"
	CustomDomainsVerificationStatusFailed    CustomDomainsVerificationStatus = "failed"
)
```

#### func (CustomDomainsVerificationStatus) Scan

```go
func (e *CustomDomainsVerificationStatus) Scan(src interface{}) error
```

### type DBTX

```go
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
```

DBTX is an interface that abstracts database operations for both direct connections and transactions. It allows query methods to work with either a database or transaction, making transaction handling more flexible.

This interface is implemented by both sql.DB and sql.Tx, as well as the custom Replica type in this package.

### type DBTx

```go
type DBTx interface {
	DBTX
	Commit() error
	Rollback() error
}
```

DBTx represents a database transaction with commit and rollback capabilities. It extends DBTX with transaction-specific methods.

#### func WrapTxWithContext

```go
func WrapTxWithContext(tx *sql.Tx, mode string, ctx context.Context) DBTx
```

WrapTxWithContext wraps a standard sql.Tx with our DBTx interface for tracing, using the provided context

### type Database

```go
type Database interface {
	// RW returns the write (primary) replica for write operations
	RW() *Replica

	// RO returns the read replica for read operations
	// If no read replica is configured, it returns the write replica
	RO() *Replica

	// Close properly terminates all database connections
	Close() error
}
```

Database defines the interface for database operations, providing access to read and write replicas and the ability to close connections.

### type DeleteDeploymentInstancesParams

```go
type DeleteDeploymentInstancesParams struct {
	DeploymentID string `db:"deployment_id"`
	Region       string `db:"region"`
}
```

### type DeleteIdentityParams

```go
type DeleteIdentityParams struct {
	IdentityID  string `db:"identity_id"`
	WorkspaceID string `db:"workspace_id"`
}
```

### type DeleteInstanceParams

```go
type DeleteInstanceParams struct {
	K8sName string `db:"k8s_name"`
	Region  string `db:"region"`
}
```

### type DeleteKeyPermissionByKeyAndPermissionIDParams

```go
type DeleteKeyPermissionByKeyAndPermissionIDParams struct {
	KeyID        string `db:"key_id"`
	PermissionID string `db:"permission_id"`
}
```

### type DeleteManyKeyPermissionByKeyAndPermissionIDsParams

```go
type DeleteManyKeyPermissionByKeyAndPermissionIDsParams struct {
	KeyID string   `db:"key_id"`
	Ids   []string `db:"ids"`
}
```

### type DeleteManyKeyRolesByKeyAndRoleIDsParams

```go
type DeleteManyKeyRolesByKeyAndRoleIDsParams struct {
	KeyID   string   `db:"key_id"`
	RoleIds []string `db:"role_ids"`
}
```

### type DeleteManyKeyRolesByKeyIDParams

```go
type DeleteManyKeyRolesByKeyIDParams struct {
	KeyID  string `db:"key_id"`
	RoleID string `db:"role_id"`
}
```

### type DeleteOldIdentityByExternalIDParams

```go
type DeleteOldIdentityByExternalIDParams struct {
	WorkspaceID       string `db:"workspace_id"`
	ExternalID        string `db:"external_id"`
	CurrentIdentityID string `db:"current_identity_id"`
}
```

### type DeleteOldIdentityWithRatelimitsParams

```go
type DeleteOldIdentityWithRatelimitsParams struct {
	WorkspaceID string `db:"workspace_id"`
	Identity    string `db:"identity"`
}
```

### type DeleteRatelimitNamespaceParams

```go
type DeleteRatelimitNamespaceParams struct {
	Now sql.NullInt64 `db:"now"`
	ID  string        `db:"id"`
}
```

### type Deployment

```go
type Deployment struct {
	Pk                            uint64                  `db:"pk"`
	ID                            string                  `db:"id"`
	K8sName                       string                  `db:"k8s_name"`
	WorkspaceID                   string                  `db:"workspace_id"`
	ProjectID                     string                  `db:"project_id"`
	EnvironmentID                 string                  `db:"environment_id"`
	Image                         sql.NullString          `db:"image"`
	BuildID                       sql.NullString          `db:"build_id"`
	GitCommitSha                  sql.NullString          `db:"git_commit_sha"`
	GitBranch                     sql.NullString          `db:"git_branch"`
	GitCommitMessage              sql.NullString          `db:"git_commit_message"`
	GitCommitAuthorHandle         sql.NullString          `db:"git_commit_author_handle"`
	GitCommitAuthorAvatarUrl      sql.NullString          `db:"git_commit_author_avatar_url"`
	GitCommitTimestamp            sql.NullInt64           `db:"git_commit_timestamp"`
	SentinelConfig                []byte                  `db:"sentinel_config"`
	OpenapiSpec                   sql.NullString          `db:"openapi_spec"`
	CpuMillicores                 int32                   `db:"cpu_millicores"`
	MemoryMib                     int32                   `db:"memory_mib"`
	DesiredState                  DeploymentsDesiredState `db:"desired_state"`
	EncryptedEnvironmentVariables []byte                  `db:"encrypted_environment_variables"`
	Command                       dbtype.StringSlice      `db:"command"`
	Status                        DeploymentsStatus       `db:"status"`
	CreatedAt                     int64                   `db:"created_at"`
	UpdatedAt                     sql.NullInt64           `db:"updated_at"`
}
```

### type DeploymentTopology

```go
type DeploymentTopology struct {
	Pk              uint64                          `db:"pk"`
	WorkspaceID     string                          `db:"workspace_id"`
	DeploymentID    string                          `db:"deployment_id"`
	Region          string                          `db:"region"`
	DesiredReplicas int32                           `db:"desired_replicas"`
	Version         uint64                          `db:"version"`
	DesiredStatus   DeploymentTopologyDesiredStatus `db:"desired_status"`
	CreatedAt       int64                           `db:"created_at"`
	UpdatedAt       sql.NullInt64                   `db:"updated_at"`
}
```

### type DeploymentTopologyDesiredStatus

```go
type DeploymentTopologyDesiredStatus string
```

```go
const (
	DeploymentTopologyDesiredStatusStarting DeploymentTopologyDesiredStatus = "starting"
	DeploymentTopologyDesiredStatusStarted  DeploymentTopologyDesiredStatus = "started"
	DeploymentTopologyDesiredStatusStopping DeploymentTopologyDesiredStatus = "stopping"
	DeploymentTopologyDesiredStatusStopped  DeploymentTopologyDesiredStatus = "stopped"
)
```

#### func (DeploymentTopologyDesiredStatus) Scan

```go
func (e *DeploymentTopologyDesiredStatus) Scan(src interface{}) error
```

### type DeploymentsDesiredState

```go
type DeploymentsDesiredState string
```

```go
const (
	DeploymentsDesiredStateRunning  DeploymentsDesiredState = "running"
	DeploymentsDesiredStateStandby  DeploymentsDesiredState = "standby"
	DeploymentsDesiredStateArchived DeploymentsDesiredState = "archived"
)
```

#### func (DeploymentsDesiredState) Scan

```go
func (e *DeploymentsDesiredState) Scan(src interface{}) error
```

### type DeploymentsStatus

```go
type DeploymentsStatus string
```

```go
const (
	DeploymentsStatusPending   DeploymentsStatus = "pending"
	DeploymentsStatusBuilding  DeploymentsStatus = "building"
	DeploymentsStatusDeploying DeploymentsStatus = "deploying"
	DeploymentsStatusNetwork   DeploymentsStatus = "network"
	DeploymentsStatusReady     DeploymentsStatus = "ready"
	DeploymentsStatusFailed    DeploymentsStatus = "failed"
)
```

#### func (DeploymentsStatus) Scan

```go
func (e *DeploymentsStatus) Scan(src interface{}) error
```

### type EncryptedKey

```go
type EncryptedKey struct {
	Pk              uint64        `db:"pk"`
	WorkspaceID     string        `db:"workspace_id"`
	KeyID           string        `db:"key_id"`
	CreatedAt       int64         `db:"created_at"`
	UpdatedAt       sql.NullInt64 `db:"updated_at"`
	Encrypted       string        `db:"encrypted"`
	EncryptionKeyID string        `db:"encryption_key_id"`
}
```

### type Environment

```go
type Environment struct {
	Pk               uint64        `db:"pk"`
	ID               string        `db:"id"`
	WorkspaceID      string        `db:"workspace_id"`
	ProjectID        string        `db:"project_id"`
	Slug             string        `db:"slug"`
	Description      string        `db:"description"`
	SentinelConfig   []byte        `db:"sentinel_config"`
	DeleteProtection sql.NullBool  `db:"delete_protection"`
	CreatedAt        int64         `db:"created_at"`
	UpdatedAt        sql.NullInt64 `db:"updated_at"`
}
```

### type EnvironmentVariable

```go
type EnvironmentVariable struct {
	Pk               uint64                   `db:"pk"`
	ID               string                   `db:"id"`
	WorkspaceID      string                   `db:"workspace_id"`
	EnvironmentID    string                   `db:"environment_id"`
	Key              string                   `db:"key"`
	Value            string                   `db:"value"`
	Type             EnvironmentVariablesType `db:"type"`
	Description      sql.NullString           `db:"description"`
	DeleteProtection sql.NullBool             `db:"delete_protection"`
	CreatedAt        int64                    `db:"created_at"`
	UpdatedAt        sql.NullInt64            `db:"updated_at"`
}
```

### type EnvironmentVariablesType

```go
type EnvironmentVariablesType string
```

```go
const (
	EnvironmentVariablesTypeRecoverable EnvironmentVariablesType = "recoverable"
	EnvironmentVariablesTypeWriteonly   EnvironmentVariablesType = "writeonly"
)
```

#### func (EnvironmentVariablesType) Scan

```go
func (e *EnvironmentVariablesType) Scan(src interface{}) error
```

### type FindAcmeChallengeByTokenParams

```go
type FindAcmeChallengeByTokenParams struct {
	WorkspaceID string `db:"workspace_id"`
	DomainID    string `db:"domain_id"`
	Token       string `db:"token"`
}
```

### type FindAuditLogTargetByIDRow

```go
type FindAuditLogTargetByIDRow struct {
	AuditLogTarget AuditLogTarget `db:"audit_log_target"`
	AuditLog       AuditLog       `db:"audit_log"`
}
```

### type FindCiliumNetworkPolicyByIDAndRegionParams

```go
type FindCiliumNetworkPolicyByIDAndRegionParams struct {
	Region                string `db:"region"`
	CiliumNetworkPolicyID string `db:"cilium_network_policy_id"`
}
```

### type FindCiliumNetworkPolicyByIDAndRegionRow

```go
type FindCiliumNetworkPolicyByIDAndRegionRow struct {
	CiliumNetworkPolicy CiliumNetworkPolicy `db:"cilium_network_policy"`
	K8sNamespace        sql.NullString      `db:"k8s_namespace"`
}
```

### type FindClickhouseWorkspaceSettingsByWorkspaceIDRow

```go
type FindClickhouseWorkspaceSettingsByWorkspaceIDRow struct {
	ClickhouseWorkspaceSetting ClickhouseWorkspaceSetting `db:"clickhouse_workspace_setting"`
	Quotas                     Quotum                     `db:"quotum"`
}
```

### type FindCustomDomainByDomainOrWildcardParams

```go
type FindCustomDomainByDomainOrWildcardParams struct {
	Domain   string `db:"domain"`
	Domain_2 string `db:"domain_2"`
	Domain_3 string `db:"domain_3"`
}
```

### type FindCustomDomainWithCertByDomainRow

```go
type FindCustomDomainWithCertByDomainRow struct {
	Pk                 uint64                          `db:"pk"`
	ID                 string                          `db:"id"`
	WorkspaceID        string                          `db:"workspace_id"`
	ProjectID          string                          `db:"project_id"`
	EnvironmentID      string                          `db:"environment_id"`
	Domain             string                          `db:"domain"`
	ChallengeType      CustomDomainsChallengeType      `db:"challenge_type"`
	VerificationStatus CustomDomainsVerificationStatus `db:"verification_status"`
	VerificationToken  string                          `db:"verification_token"`
	OwnershipVerified  bool                            `db:"ownership_verified"`
	CnameVerified      bool                            `db:"cname_verified"`
	TargetCname        string                          `db:"target_cname"`
	LastCheckedAt      sql.NullInt64                   `db:"last_checked_at"`
	CheckAttempts      int32                           `db:"check_attempts"`
	VerificationError  sql.NullString                  `db:"verification_error"`
	InvocationID       sql.NullString                  `db:"invocation_id"`
	CreatedAt          int64                           `db:"created_at"`
	UpdatedAt          sql.NullInt64                   `db:"updated_at"`
	CertificateID      sql.NullString                  `db:"certificate_id"`
}
```

### type FindDeploymentTopologyByIDAndRegionParams

```go
type FindDeploymentTopologyByIDAndRegionParams struct {
	Region       string `db:"region"`
	DeploymentID string `db:"deployment_id"`
}
```

### type FindDeploymentTopologyByIDAndRegionRow

```go
type FindDeploymentTopologyByIDAndRegionRow struct {
	ID                            string                  `db:"id"`
	K8sName                       string                  `db:"k8s_name"`
	K8sNamespace                  sql.NullString          `db:"k8s_namespace"`
	WorkspaceID                   string                  `db:"workspace_id"`
	ProjectID                     string                  `db:"project_id"`
	EnvironmentID                 string                  `db:"environment_id"`
	BuildID                       sql.NullString          `db:"build_id"`
	Image                         sql.NullString          `db:"image"`
	Region                        string                  `db:"region"`
	CpuMillicores                 int32                   `db:"cpu_millicores"`
	MemoryMib                     int32                   `db:"memory_mib"`
	DesiredReplicas               int32                   `db:"desired_replicas"`
	DesiredState                  DeploymentsDesiredState `db:"desired_state"`
	EncryptedEnvironmentVariables []byte                  `db:"encrypted_environment_variables"`
}
```

### type FindEnvironmentByIdRow

```go
type FindEnvironmentByIdRow struct {
	ID          string `db:"id"`
	WorkspaceID string `db:"workspace_id"`
	ProjectID   string `db:"project_id"`
	Slug        string `db:"slug"`
	Description string `db:"description"`
}
```

### type FindEnvironmentByProjectIdAndSlugParams

```go
type FindEnvironmentByProjectIdAndSlugParams struct {
	WorkspaceID string `db:"workspace_id"`
	ProjectID   string `db:"project_id"`
	Slug        string `db:"slug"`
}
```

### type FindEnvironmentVariablesByEnvironmentIdRow

```go
type FindEnvironmentVariablesByEnvironmentIdRow struct {
	Key   string `db:"key"`
	Value string `db:"value"`
}
```

### type FindFrontlineRouteForPromotionParams

```go
type FindFrontlineRouteForPromotionParams struct {
	EnvironmentID string                  `db:"environment_id"`
	Sticky        []FrontlineRoutesSticky `db:"sticky"`
}
```

### type FindFrontlineRouteForPromotionRow

```go
type FindFrontlineRouteForPromotionRow struct {
	ID                       string                `db:"id"`
	ProjectID                string                `db:"project_id"`
	EnvironmentID            string                `db:"environment_id"`
	FullyQualifiedDomainName string                `db:"fully_qualified_domain_name"`
	DeploymentID             string                `db:"deployment_id"`
	Sticky                   FrontlineRoutesSticky `db:"sticky"`
	CreatedAt                int64                 `db:"created_at"`
	UpdatedAt                sql.NullInt64         `db:"updated_at"`
}
```

### type FindFrontlineRoutesForRollbackParams

```go
type FindFrontlineRoutesForRollbackParams struct {
	EnvironmentID string                  `db:"environment_id"`
	Sticky        []FrontlineRoutesSticky `db:"sticky"`
}
```

### type FindFrontlineRoutesForRollbackRow

```go
type FindFrontlineRoutesForRollbackRow struct {
	ID                       string                `db:"id"`
	ProjectID                string                `db:"project_id"`
	EnvironmentID            string                `db:"environment_id"`
	FullyQualifiedDomainName string                `db:"fully_qualified_domain_name"`
	DeploymentID             string                `db:"deployment_id"`
	Sticky                   FrontlineRoutesSticky `db:"sticky"`
	CreatedAt                int64                 `db:"created_at"`
	UpdatedAt                sql.NullInt64         `db:"updated_at"`
}
```

### type FindGithubRepoConnectionParams

```go
type FindGithubRepoConnectionParams struct {
	InstallationID int64 `db:"installation_id"`
	RepositoryID   int64 `db:"repository_id"`
}
```

### type FindIdentitiesByExternalIdParams

```go
type FindIdentitiesByExternalIdParams struct {
	WorkspaceID string   `db:"workspace_id"`
	ExternalIds []string `db:"externalIds"`
	Deleted     bool     `db:"deleted"`
}
```

### type FindIdentitiesParams

```go
type FindIdentitiesParams struct {
	WorkspaceID string   `db:"workspace_id"`
	Deleted     bool     `db:"deleted"`
	Identities  []string `db:"identities"`
}
```

### type FindIdentityByExternalIDParams

```go
type FindIdentityByExternalIDParams struct {
	WorkspaceID string `db:"workspace_id"`
	ExternalID  string `db:"external_id"`
	Deleted     bool   `db:"deleted"`
}
```

### type FindIdentityByIDParams

```go
type FindIdentityByIDParams struct {
	WorkspaceID string `db:"workspace_id"`
	IdentityID  string `db:"identity_id"`
	Deleted     bool   `db:"deleted"`
}
```

### type FindIdentityParams

```go
type FindIdentityParams struct {
	Identity    string `db:"identity"`
	WorkspaceID string `db:"workspace_id"`
	Deleted     bool   `db:"deleted"`
}
```

### type FindIdentityRow

```go
type FindIdentityRow struct {
	Pk          uint64        `db:"pk"`
	ID          string        `db:"id"`
	ExternalID  string        `db:"external_id"`
	WorkspaceID string        `db:"workspace_id"`
	Environment string        `db:"environment"`
	Meta        []byte        `db:"meta"`
	Deleted     bool          `db:"deleted"`
	CreatedAt   int64         `db:"created_at"`
	UpdatedAt   sql.NullInt64 `db:"updated_at"`
	Ratelimits  interface{}   `db:"ratelimits"`
}
```

### type FindInstanceByPodNameParams

```go
type FindInstanceByPodNameParams struct {
	K8sName string `db:"k8s_name"`
	Region  string `db:"region"`
}
```

### type FindInstancesByDeploymentIdAndRegionParams

```go
type FindInstancesByDeploymentIdAndRegionParams struct {
	Deploymentid string `db:"deploymentid"`
	Region       string `db:"region"`
}
```

### type FindKeyAuthsByIdsParams

```go
type FindKeyAuthsByIdsParams struct {
	WorkspaceID string   `db:"workspace_id"`
	ApiIds      []string `db:"api_ids"`
}
```

### type FindKeyAuthsByIdsRow

```go
type FindKeyAuthsByIdsRow struct {
	KeyAuthID string `db:"key_auth_id"`
	ApiID     string `db:"api_id"`
}
```

### type FindKeyAuthsByKeyAuthIdsParams

```go
type FindKeyAuthsByKeyAuthIdsParams struct {
	WorkspaceID string   `db:"workspace_id"`
	KeyAuthIds  []string `db:"key_auth_ids"`
}
```

### type FindKeyAuthsByKeyAuthIdsRow

```go
type FindKeyAuthsByKeyAuthIdsRow struct {
	KeyAuthID string `db:"key_auth_id"`
	ApiID     string `db:"api_id"`
}
```

### type FindKeyForVerificationRow

```go
type FindKeyForVerificationRow struct {
	ID                  string         `db:"id"`
	KeyAuthID           string         `db:"key_auth_id"`
	WorkspaceID         string         `db:"workspace_id"`
	ForWorkspaceID      sql.NullString `db:"for_workspace_id"`
	Name                sql.NullString `db:"name"`
	Meta                sql.NullString `db:"meta"`
	Expires             sql.NullTime   `db:"expires"`
	DeletedAtM          sql.NullInt64  `db:"deleted_at_m"`
	RefillDay           sql.NullInt16  `db:"refill_day"`
	RefillAmount        sql.NullInt32  `db:"refill_amount"`
	LastRefillAt        sql.NullTime   `db:"last_refill_at"`
	Enabled             bool           `db:"enabled"`
	RemainingRequests   sql.NullInt32  `db:"remaining_requests"`
	PendingMigrationID  sql.NullString `db:"pending_migration_id"`
	IpWhitelist         sql.NullString `db:"ip_whitelist"`
	ApiWorkspaceID      string         `db:"api_workspace_id"`
	ApiID               string         `db:"api_id"`
	ApiDeletedAtM       sql.NullInt64  `db:"api_deleted_at_m"`
	Roles               interface{}    `db:"roles"`
	Permissions         interface{}    `db:"permissions"`
	Ratelimits          interface{}    `db:"ratelimits"`
	IdentityID          sql.NullString `db:"identity_id"`
	ExternalID          sql.NullString `db:"external_id"`
	IdentityMeta        []byte         `db:"identity_meta"`
	KeyAuthDeletedAtM   sql.NullInt64  `db:"key_auth_deleted_at_m"`
	WorkspaceEnabled    bool           `db:"workspace_enabled"`
	ForWorkspaceEnabled sql.NullBool   `db:"for_workspace_enabled"`
}
```

### type FindKeyMigrationByIDParams

```go
type FindKeyMigrationByIDParams struct {
	ID          string `db:"id"`
	WorkspaceID string `db:"workspace_id"`
}
```

### type FindKeyMigrationByIDRow

```go
type FindKeyMigrationByIDRow struct {
	ID          string                 `db:"id"`
	WorkspaceID string                 `db:"workspace_id"`
	Algorithm   KeyMigrationsAlgorithm `db:"algorithm"`
}
```

### type FindKeyRoleByKeyAndRoleIDParams

```go
type FindKeyRoleByKeyAndRoleIDParams struct {
	KeyID  string `db:"key_id"`
	RoleID string `db:"role_id"`
}
```

### type FindKeysByHashRow

```go
type FindKeysByHashRow struct {
	ID   string `db:"id"`
	Hash string `db:"hash"`
}
```

### type FindLiveApiByIDRow

```go
type FindLiveApiByIDRow struct {
	Pk               uint64           `db:"pk"`
	ID               string           `db:"id"`
	Name             string           `db:"name"`
	WorkspaceID      string           `db:"workspace_id"`
	IpWhitelist      sql.NullString   `db:"ip_whitelist"`
	AuthType         NullApisAuthType `db:"auth_type"`
	KeyAuthID        sql.NullString   `db:"key_auth_id"`
	CreatedAtM       int64            `db:"created_at_m"`
	UpdatedAtM       sql.NullInt64    `db:"updated_at_m"`
	DeletedAtM       sql.NullInt64    `db:"deleted_at_m"`
	DeleteProtection sql.NullBool     `db:"delete_protection"`
	KeyAuth          KeyAuth          `db:"key_auth"`
}
```

### type FindLiveKeyByHashRow

```go
type FindLiveKeyByHashRow struct {
	Pk                 uint64         `db:"pk"`
	ID                 string         `db:"id"`
	KeyAuthID          string         `db:"key_auth_id"`
	Hash               string         `db:"hash"`
	Start              string         `db:"start"`
	WorkspaceID        string         `db:"workspace_id"`
	ForWorkspaceID     sql.NullString `db:"for_workspace_id"`
	Name               sql.NullString `db:"name"`
	OwnerID            sql.NullString `db:"owner_id"`
	IdentityID         sql.NullString `db:"identity_id"`
	Meta               sql.NullString `db:"meta"`
	Expires            sql.NullTime   `db:"expires"`
	CreatedAtM         int64          `db:"created_at_m"`
	UpdatedAtM         sql.NullInt64  `db:"updated_at_m"`
	DeletedAtM         sql.NullInt64  `db:"deleted_at_m"`
	RefillDay          sql.NullInt16  `db:"refill_day"`
	RefillAmount       sql.NullInt32  `db:"refill_amount"`
	LastRefillAt       sql.NullTime   `db:"last_refill_at"`
	Enabled            bool           `db:"enabled"`
	RemainingRequests  sql.NullInt32  `db:"remaining_requests"`
	RatelimitAsync     sql.NullBool   `db:"ratelimit_async"`
	RatelimitLimit     sql.NullInt32  `db:"ratelimit_limit"`
	RatelimitDuration  sql.NullInt64  `db:"ratelimit_duration"`
	Environment        sql.NullString `db:"environment"`
	PendingMigrationID sql.NullString `db:"pending_migration_id"`
	Api                Api            `db:"api"`
	KeyAuth            KeyAuth        `db:"key_auth"`
	Workspace          Workspace      `db:"workspace"`
	IdentityTableID    sql.NullString `db:"identity_table_id"`
	IdentityExternalID sql.NullString `db:"identity_external_id"`
	IdentityMeta       []byte         `db:"identity_meta"`
	EncryptedKey       sql.NullString `db:"encrypted_key"`
	EncryptionKeyID    sql.NullString `db:"encryption_key_id"`
	Roles              interface{}    `db:"roles"`
	Permissions        interface{}    `db:"permissions"`
	RolePermissions    interface{}    `db:"role_permissions"`
	Ratelimits         interface{}    `db:"ratelimits"`
}
```

### type FindLiveKeyByIDRow

```go
type FindLiveKeyByIDRow struct {
	Pk                 uint64         `db:"pk"`
	ID                 string         `db:"id"`
	KeyAuthID          string         `db:"key_auth_id"`
	Hash               string         `db:"hash"`
	Start              string         `db:"start"`
	WorkspaceID        string         `db:"workspace_id"`
	ForWorkspaceID     sql.NullString `db:"for_workspace_id"`
	Name               sql.NullString `db:"name"`
	OwnerID            sql.NullString `db:"owner_id"`
	IdentityID         sql.NullString `db:"identity_id"`
	Meta               sql.NullString `db:"meta"`
	Expires            sql.NullTime   `db:"expires"`
	CreatedAtM         int64          `db:"created_at_m"`
	UpdatedAtM         sql.NullInt64  `db:"updated_at_m"`
	DeletedAtM         sql.NullInt64  `db:"deleted_at_m"`
	RefillDay          sql.NullInt16  `db:"refill_day"`
	RefillAmount       sql.NullInt32  `db:"refill_amount"`
	LastRefillAt       sql.NullTime   `db:"last_refill_at"`
	Enabled            bool           `db:"enabled"`
	RemainingRequests  sql.NullInt32  `db:"remaining_requests"`
	RatelimitAsync     sql.NullBool   `db:"ratelimit_async"`
	RatelimitLimit     sql.NullInt32  `db:"ratelimit_limit"`
	RatelimitDuration  sql.NullInt64  `db:"ratelimit_duration"`
	Environment        sql.NullString `db:"environment"`
	PendingMigrationID sql.NullString `db:"pending_migration_id"`
	Api                Api            `db:"api"`
	KeyAuth            KeyAuth        `db:"key_auth"`
	Workspace          Workspace      `db:"workspace"`
	IdentityTableID    sql.NullString `db:"identity_table_id"`
	IdentityExternalID sql.NullString `db:"identity_external_id"`
	IdentityMeta       []byte         `db:"identity_meta"`
	EncryptedKey       sql.NullString `db:"encrypted_key"`
	EncryptionKeyID    sql.NullString `db:"encryption_key_id"`
	Roles              interface{}    `db:"roles"`
	Permissions        interface{}    `db:"permissions"`
	RolePermissions    interface{}    `db:"role_permissions"`
	Ratelimits         interface{}    `db:"ratelimits"`
}
```

### type FindManyRatelimitNamespacesParams

```go
type FindManyRatelimitNamespacesParams struct {
	WorkspaceID string   `db:"workspace_id"`
	Namespaces  []string `db:"namespaces"`
}
```

### type FindManyRatelimitNamespacesRow

```go
type FindManyRatelimitNamespacesRow struct {
	Pk          uint64        `db:"pk"`
	ID          string        `db:"id"`
	WorkspaceID string        `db:"workspace_id"`
	Name        string        `db:"name"`
	CreatedAtM  int64         `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64 `db:"updated_at_m"`
	DeletedAtM  sql.NullInt64 `db:"deleted_at_m"`
	Overrides   interface{}   `db:"overrides"`
}
```

### type FindManyRolesByIdOrNameWithPermsParams

```go
type FindManyRolesByIdOrNameWithPermsParams struct {
	WorkspaceID string   `db:"workspace_id"`
	Search      []string `db:"search"`
}
```

### type FindManyRolesByIdOrNameWithPermsRow

```go
type FindManyRolesByIdOrNameWithPermsRow struct {
	Pk          uint64         `db:"pk"`
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAtM  int64          `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64  `db:"updated_at_m"`
	Permissions interface{}    `db:"permissions"`
}
```

### type FindManyRolesByNamesWithPermsParams

```go
type FindManyRolesByNamesWithPermsParams struct {
	WorkspaceID string   `db:"workspace_id"`
	Names       []string `db:"names"`
}
```

### type FindManyRolesByNamesWithPermsRow

```go
type FindManyRolesByNamesWithPermsRow struct {
	Pk          uint64         `db:"pk"`
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAtM  int64          `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64  `db:"updated_at_m"`
	Permissions interface{}    `db:"permissions"`
}
```

### type FindPermissionByIdOrSlugParams

```go
type FindPermissionByIdOrSlugParams struct {
	WorkspaceID string `db:"workspace_id"`
	Search      string `db:"search"`
}
```

### type FindPermissionByNameAndWorkspaceIDParams

```go
type FindPermissionByNameAndWorkspaceIDParams struct {
	Name        string `db:"name"`
	WorkspaceID string `db:"workspace_id"`
}
```

### type FindPermissionBySlugAndWorkspaceIDParams

```go
type FindPermissionBySlugAndWorkspaceIDParams struct {
	Slug        string `db:"slug"`
	WorkspaceID string `db:"workspace_id"`
}
```

### type FindPermissionsBySlugsParams

```go
type FindPermissionsBySlugsParams struct {
	WorkspaceID string   `db:"workspace_id"`
	Slugs       []string `db:"slugs"`
}
```

### type FindProjectByIdRow

```go
type FindProjectByIdRow struct {
	ID               string             `db:"id"`
	WorkspaceID      string             `db:"workspace_id"`
	Name             string             `db:"name"`
	Slug             string             `db:"slug"`
	GitRepositoryUrl sql.NullString     `db:"git_repository_url"`
	DefaultBranch    sql.NullString     `db:"default_branch"`
	DeleteProtection sql.NullBool       `db:"delete_protection"`
	LiveDeploymentID sql.NullString     `db:"live_deployment_id"`
	IsRolledBack     bool               `db:"is_rolled_back"`
	CreatedAt        int64              `db:"created_at"`
	UpdatedAt        sql.NullInt64      `db:"updated_at"`
	DepotProjectID   sql.NullString     `db:"depot_project_id"`
	Command          dbtype.StringSlice `db:"command"`
}
```

### type FindProjectByWorkspaceSlugParams

```go
type FindProjectByWorkspaceSlugParams struct {
	WorkspaceID string `db:"workspace_id"`
	Slug        string `db:"slug"`
}
```

### type FindProjectByWorkspaceSlugRow

```go
type FindProjectByWorkspaceSlugRow struct {
	ID               string         `db:"id"`
	WorkspaceID      string         `db:"workspace_id"`
	Name             string         `db:"name"`
	Slug             string         `db:"slug"`
	GitRepositoryUrl sql.NullString `db:"git_repository_url"`
	DefaultBranch    sql.NullString `db:"default_branch"`
	DeleteProtection sql.NullBool   `db:"delete_protection"`
	CreatedAt        int64          `db:"created_at"`
	UpdatedAt        sql.NullInt64  `db:"updated_at"`
}
```

### type FindRatelimitNamespace

```go
type FindRatelimitNamespace struct {
	ID                string                                         `db:"id"`
	WorkspaceID       string                                         `db:"workspace_id"`
	Name              string                                         `db:"name"`
	CreatedAtM        int64                                          `db:"created_at_m"`
	UpdatedAtM        sql.NullInt64                                  `db:"updated_at_m"`
	DeletedAtM        sql.NullInt64                                  `db:"deleted_at_m"`
	DirectOverrides   map[string]FindRatelimitNamespaceLimitOverride `db:"direct_overrides"`
	WildcardOverrides []FindRatelimitNamespaceLimitOverride          `db:"wildcard_overrides"`
}
```

### type FindRatelimitNamespaceByNameParams

```go
type FindRatelimitNamespaceByNameParams struct {
	Name        string `db:"name"`
	WorkspaceID string `db:"workspace_id"`
}
```

### type FindRatelimitNamespaceLimitOverride

```go
type FindRatelimitNamespaceLimitOverride struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Limit      int64  `json:"limit"`
	Duration   int64  `json:"duration"`
}
```

### type FindRatelimitNamespaceParams

```go
type FindRatelimitNamespaceParams struct {
	WorkspaceID string `db:"workspace_id"`
	Namespace   string `db:"namespace"`
}
```

### type FindRatelimitNamespaceRow

```go
type FindRatelimitNamespaceRow struct {
	Pk          uint64        `db:"pk"`
	ID          string        `db:"id"`
	WorkspaceID string        `db:"workspace_id"`
	Name        string        `db:"name"`
	CreatedAtM  int64         `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64 `db:"updated_at_m"`
	DeletedAtM  sql.NullInt64 `db:"deleted_at_m"`
	Overrides   interface{}   `db:"overrides"`
}
```

### type FindRatelimitOverrideByIDParams

```go
type FindRatelimitOverrideByIDParams struct {
	WorkspaceID string `db:"workspace_id"`
	OverrideID  string `db:"override_id"`
}
```

### type FindRatelimitOverrideByIdentifierParams

```go
type FindRatelimitOverrideByIdentifierParams struct {
	WorkspaceID string `db:"workspace_id"`
	NamespaceID string `db:"namespace_id"`
	Identifier  string `db:"identifier"`
}
```

### type FindRoleByIdOrNameWithPermsParams

```go
type FindRoleByIdOrNameWithPermsParams struct {
	WorkspaceID string `db:"workspace_id"`
	Search      string `db:"search"`
}
```

### type FindRoleByIdOrNameWithPermsRow

```go
type FindRoleByIdOrNameWithPermsRow struct {
	Pk          uint64         `db:"pk"`
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAtM  int64          `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64  `db:"updated_at_m"`
	Permissions interface{}    `db:"permissions"`
}
```

### type FindRoleByNameAndWorkspaceIDParams

```go
type FindRoleByNameAndWorkspaceIDParams struct {
	Name        string `db:"name"`
	WorkspaceID string `db:"workspace_id"`
}
```

### type FindRolePermissionByRoleAndPermissionIDParams

```go
type FindRolePermissionByRoleAndPermissionIDParams struct {
	RoleID       string `db:"role_id"`
	PermissionID string `db:"permission_id"`
}
```

### type FindRolesByNamesParams

```go
type FindRolesByNamesParams struct {
	WorkspaceID string   `db:"workspace_id"`
	Names       []string `db:"names"`
}
```

### type FindRolesByNamesRow

```go
type FindRolesByNamesRow struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}
```

### type FrontlineRoute

```go
type FrontlineRoute struct {
	Pk                       uint64                `db:"pk"`
	ID                       string                `db:"id"`
	ProjectID                string                `db:"project_id"`
	DeploymentID             string                `db:"deployment_id"`
	EnvironmentID            string                `db:"environment_id"`
	FullyQualifiedDomainName string                `db:"fully_qualified_domain_name"`
	Sticky                   FrontlineRoutesSticky `db:"sticky"`
	CreatedAt                int64                 `db:"created_at"`
	UpdatedAt                sql.NullInt64         `db:"updated_at"`
}
```

### type FrontlineRoutesSticky

```go
type FrontlineRoutesSticky string
```

```go
const (
	FrontlineRoutesStickyNone        FrontlineRoutesSticky = "none"
	FrontlineRoutesStickyBranch      FrontlineRoutesSticky = "branch"
	FrontlineRoutesStickyEnvironment FrontlineRoutesSticky = "environment"
	FrontlineRoutesStickyLive        FrontlineRoutesSticky = "live"
)
```

#### func (FrontlineRoutesSticky) Scan

```go
func (e *FrontlineRoutesSticky) Scan(src interface{}) error
```

### type GetKeyAuthByIDRow

```go
type GetKeyAuthByIDRow struct {
	ID                 string         `db:"id"`
	WorkspaceID        string         `db:"workspace_id"`
	CreatedAtM         int64          `db:"created_at_m"`
	DefaultPrefix      sql.NullString `db:"default_prefix"`
	DefaultBytes       sql.NullInt32  `db:"default_bytes"`
	StoreEncryptedKeys bool           `db:"store_encrypted_keys"`
}
```

### type GetWorkspacesForQuotaCheckByIDsRow

```go
type GetWorkspacesForQuotaCheckByIDsRow struct {
	ID               string         `db:"id"`
	OrgID            string         `db:"org_id"`
	Name             string         `db:"name"`
	StripeCustomerID sql.NullString `db:"stripe_customer_id"`
	Tier             sql.NullString `db:"tier"`
	Enabled          bool           `db:"enabled"`
	RequestsPerMonth sql.NullInt64  `db:"requests_per_month"`
}
```

### type GithubAppInstallation

```go
type GithubAppInstallation struct {
	Pk             uint64        `db:"pk"`
	WorkspaceID    string        `db:"workspace_id"`
	InstallationID int64         `db:"installation_id"`
	CreatedAt      int64         `db:"created_at"`
	UpdatedAt      sql.NullInt64 `db:"updated_at"`
}
```

### type GithubRepoConnection

```go
type GithubRepoConnection struct {
	Pk                 uint64        `db:"pk"`
	ProjectID          string        `db:"project_id"`
	InstallationID     int64         `db:"installation_id"`
	RepositoryID       int64         `db:"repository_id"`
	RepositoryFullName string        `db:"repository_full_name"`
	CreatedAt          int64         `db:"created_at"`
	UpdatedAt          sql.NullInt64 `db:"updated_at"`
}
```

### type Identity

```go
type Identity struct {
	Pk          uint64        `db:"pk"`
	ID          string        `db:"id"`
	ExternalID  string        `db:"external_id"`
	WorkspaceID string        `db:"workspace_id"`
	Environment string        `db:"environment"`
	Meta        []byte        `db:"meta"`
	Deleted     bool          `db:"deleted"`
	CreatedAt   int64         `db:"created_at"`
	UpdatedAt   sql.NullInt64 `db:"updated_at"`
}
```

### type InsertAcmeChallengeParams

```go
type InsertAcmeChallengeParams struct {
	WorkspaceID   string                      `db:"workspace_id"`
	DomainID      string                      `db:"domain_id"`
	Token         string                      `db:"token"`
	Authorization string                      `db:"authorization"`
	Status        AcmeChallengesStatus        `db:"status"`
	ChallengeType AcmeChallengesChallengeType `db:"challenge_type"`
	CreatedAt     int64                       `db:"created_at"`
	UpdatedAt     sql.NullInt64               `db:"updated_at"`
	ExpiresAt     int64                       `db:"expires_at"`
}
```

### type InsertAcmeUserParams

```go
type InsertAcmeUserParams struct {
	ID           string `db:"id"`
	WorkspaceID  string `db:"workspace_id"`
	EncryptedKey string `db:"encrypted_key"`
	CreatedAt    int64  `db:"created_at"`
}
```

### type InsertApiParams

```go
type InsertApiParams struct {
	ID          string           `db:"id"`
	Name        string           `db:"name"`
	WorkspaceID string           `db:"workspace_id"`
	AuthType    NullApisAuthType `db:"auth_type"`
	IpWhitelist sql.NullString   `db:"ip_whitelist"`
	KeyAuthID   sql.NullString   `db:"key_auth_id"`
	CreatedAtM  int64            `db:"created_at_m"`
}
```

### type InsertAuditLogParams

```go
type InsertAuditLogParams struct {
	ID          string          `db:"id"`
	WorkspaceID string          `db:"workspace_id"`
	BucketID    string          `db:"bucket_id"`
	Bucket      string          `db:"bucket"`
	Event       string          `db:"event"`
	Time        int64           `db:"time"`
	Display     string          `db:"display"`
	RemoteIp    sql.NullString  `db:"remote_ip"`
	UserAgent   sql.NullString  `db:"user_agent"`
	ActorType   string          `db:"actor_type"`
	ActorID     string          `db:"actor_id"`
	ActorName   sql.NullString  `db:"actor_name"`
	ActorMeta   json.RawMessage `db:"actor_meta"`
	CreatedAt   int64           `db:"created_at"`
}
```

### type InsertAuditLogTargetParams

```go
type InsertAuditLogTargetParams struct {
	WorkspaceID string          `db:"workspace_id"`
	BucketID    string          `db:"bucket_id"`
	Bucket      string          `db:"bucket"`
	AuditLogID  string          `db:"audit_log_id"`
	DisplayName string          `db:"display_name"`
	Type        string          `db:"type"`
	ID          string          `db:"id"`
	Name        sql.NullString  `db:"name"`
	Meta        json.RawMessage `db:"meta"`
	CreatedAt   int64           `db:"created_at"`
}
```

### type InsertCertificateParams

```go
type InsertCertificateParams struct {
	ID                  string        `db:"id"`
	WorkspaceID         string        `db:"workspace_id"`
	Hostname            string        `db:"hostname"`
	Certificate         string        `db:"certificate"`
	EncryptedPrivateKey string        `db:"encrypted_private_key"`
	CreatedAt           int64         `db:"created_at"`
	UpdatedAt           sql.NullInt64 `db:"updated_at"`
}
```

### type InsertCiliumNetworkPolicyParams

```go
type InsertCiliumNetworkPolicyParams struct {
	ID            string          `db:"id"`
	WorkspaceID   string          `db:"workspace_id"`
	ProjectID     string          `db:"project_id"`
	EnvironmentID string          `db:"environment_id"`
	K8sName       string          `db:"k8s_name"`
	Region        string          `db:"region"`
	Policy        json.RawMessage `db:"policy"`
	Version       uint64          `db:"version"`
	CreatedAt     int64           `db:"created_at"`
}
```

### type InsertClickhouseWorkspaceSettingsParams

```go
type InsertClickhouseWorkspaceSettingsParams struct {
	WorkspaceID               string        `db:"workspace_id"`
	Username                  string        `db:"username"`
	PasswordEncrypted         string        `db:"password_encrypted"`
	QuotaDurationSeconds      int32         `db:"quota_duration_seconds"`
	MaxQueriesPerWindow       int32         `db:"max_queries_per_window"`
	MaxExecutionTimePerWindow int32         `db:"max_execution_time_per_window"`
	MaxQueryExecutionTime     int32         `db:"max_query_execution_time"`
	MaxQueryMemoryBytes       int64         `db:"max_query_memory_bytes"`
	MaxQueryResultRows        int32         `db:"max_query_result_rows"`
	CreatedAt                 int64         `db:"created_at"`
	UpdatedAt                 sql.NullInt64 `db:"updated_at"`
}
```

### type InsertCustomDomainParams

```go
type InsertCustomDomainParams struct {
	ID                 string                          `db:"id"`
	WorkspaceID        string                          `db:"workspace_id"`
	ProjectID          string                          `db:"project_id"`
	EnvironmentID      string                          `db:"environment_id"`
	Domain             string                          `db:"domain"`
	ChallengeType      CustomDomainsChallengeType      `db:"challenge_type"`
	VerificationStatus CustomDomainsVerificationStatus `db:"verification_status"`
	VerificationToken  string                          `db:"verification_token"`
	TargetCname        string                          `db:"target_cname"`
	InvocationID       sql.NullString                  `db:"invocation_id"`
	CreatedAt          int64                           `db:"created_at"`
}
```

### type InsertDeploymentParams

```go
type InsertDeploymentParams struct {
	ID                            string             `db:"id"`
	K8sName                       string             `db:"k8s_name"`
	WorkspaceID                   string             `db:"workspace_id"`
	ProjectID                     string             `db:"project_id"`
	EnvironmentID                 string             `db:"environment_id"`
	GitCommitSha                  sql.NullString     `db:"git_commit_sha"`
	GitBranch                     sql.NullString     `db:"git_branch"`
	SentinelConfig                []byte             `db:"sentinel_config"`
	GitCommitMessage              sql.NullString     `db:"git_commit_message"`
	GitCommitAuthorHandle         sql.NullString     `db:"git_commit_author_handle"`
	GitCommitAuthorAvatarUrl      sql.NullString     `db:"git_commit_author_avatar_url"`
	GitCommitTimestamp            sql.NullInt64      `db:"git_commit_timestamp"`
	OpenapiSpec                   sql.NullString     `db:"openapi_spec"`
	EncryptedEnvironmentVariables []byte             `db:"encrypted_environment_variables"`
	Command                       dbtype.StringSlice `db:"command"`
	Status                        DeploymentsStatus  `db:"status"`
	CpuMillicores                 int32              `db:"cpu_millicores"`
	MemoryMib                     int32              `db:"memory_mib"`
	CreatedAt                     int64              `db:"created_at"`
	UpdatedAt                     sql.NullInt64      `db:"updated_at"`
}
```

### type InsertDeploymentTopologyParams

```go
type InsertDeploymentTopologyParams struct {
	WorkspaceID     string                          `db:"workspace_id"`
	DeploymentID    string                          `db:"deployment_id"`
	Region          string                          `db:"region"`
	DesiredReplicas int32                           `db:"desired_replicas"`
	DesiredStatus   DeploymentTopologyDesiredStatus `db:"desired_status"`
	Version         uint64                          `db:"version"`
	CreatedAt       int64                           `db:"created_at"`
}
```

### type InsertEnvironmentParams

```go
type InsertEnvironmentParams struct {
	ID             string        `db:"id"`
	WorkspaceID    string        `db:"workspace_id"`
	ProjectID      string        `db:"project_id"`
	Slug           string        `db:"slug"`
	Description    string        `db:"description"`
	CreatedAt      int64         `db:"created_at"`
	UpdatedAt      sql.NullInt64 `db:"updated_at"`
	SentinelConfig []byte        `db:"sentinel_config"`
}
```

### type InsertFrontlineRouteParams

```go
type InsertFrontlineRouteParams struct {
	ID                       string                `db:"id"`
	ProjectID                string                `db:"project_id"`
	DeploymentID             string                `db:"deployment_id"`
	EnvironmentID            string                `db:"environment_id"`
	FullyQualifiedDomainName string                `db:"fully_qualified_domain_name"`
	Sticky                   FrontlineRoutesSticky `db:"sticky"`
	CreatedAt                int64                 `db:"created_at"`
	UpdatedAt                sql.NullInt64         `db:"updated_at"`
}
```

### type InsertGithubRepoConnectionParams

```go
type InsertGithubRepoConnectionParams struct {
	ProjectID          string        `db:"project_id"`
	InstallationID     int64         `db:"installation_id"`
	RepositoryID       int64         `db:"repository_id"`
	RepositoryFullName string        `db:"repository_full_name"`
	CreatedAt          int64         `db:"created_at"`
	UpdatedAt          sql.NullInt64 `db:"updated_at"`
}
```

### type InsertIdentityParams

```go
type InsertIdentityParams struct {
	ID          string          `db:"id"`
	ExternalID  string          `db:"external_id"`
	WorkspaceID string          `db:"workspace_id"`
	Environment string          `db:"environment"`
	CreatedAt   int64           `db:"created_at"`
	Meta        json.RawMessage `db:"meta"`
}
```

### type InsertIdentityRatelimitParams

```go
type InsertIdentityRatelimitParams struct {
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	IdentityID  sql.NullString `db:"identity_id"`
	Name        string         `db:"name"`
	Limit       int32          `db:"limit"`
	Duration    int64          `db:"duration"`
	CreatedAt   int64          `db:"created_at"`
	AutoApply   bool           `db:"auto_apply"`
}
```

### type InsertKeyAuthParams

```go
type InsertKeyAuthParams struct {
	ID            string         `db:"id"`
	WorkspaceID   string         `db:"workspace_id"`
	CreatedAtM    int64          `db:"created_at_m"`
	DefaultPrefix sql.NullString `db:"default_prefix"`
	DefaultBytes  sql.NullInt32  `db:"default_bytes"`
}
```

### type InsertKeyEncryptionParams

```go
type InsertKeyEncryptionParams struct {
	WorkspaceID     string `db:"workspace_id"`
	KeyID           string `db:"key_id"`
	Encrypted       string `db:"encrypted"`
	EncryptionKeyID string `db:"encryption_key_id"`
	CreatedAt       int64  `db:"created_at"`
}
```

### type InsertKeyMigrationParams

```go
type InsertKeyMigrationParams struct {
	ID          string                 `db:"id"`
	WorkspaceID string                 `db:"workspace_id"`
	Algorithm   KeyMigrationsAlgorithm `db:"algorithm"`
}
```

### type InsertKeyParams

```go
type InsertKeyParams struct {
	ID                 string         `db:"id"`
	KeySpaceID         string         `db:"key_space_id"`
	Hash               string         `db:"hash"`
	Start              string         `db:"start"`
	WorkspaceID        string         `db:"workspace_id"`
	ForWorkspaceID     sql.NullString `db:"for_workspace_id"`
	Name               sql.NullString `db:"name"`
	IdentityID         sql.NullString `db:"identity_id"`
	Meta               sql.NullString `db:"meta"`
	Expires            sql.NullTime   `db:"expires"`
	CreatedAtM         int64          `db:"created_at_m"`
	Enabled            bool           `db:"enabled"`
	RemainingRequests  sql.NullInt32  `db:"remaining_requests"`
	RefillDay          sql.NullInt16  `db:"refill_day"`
	RefillAmount       sql.NullInt32  `db:"refill_amount"`
	PendingMigrationID sql.NullString `db:"pending_migration_id"`
}
```

### type InsertKeyPermissionParams

```go
type InsertKeyPermissionParams struct {
	KeyID        string        `db:"key_id"`
	PermissionID string        `db:"permission_id"`
	WorkspaceID  string        `db:"workspace_id"`
	CreatedAt    int64         `db:"created_at"`
	UpdatedAt    sql.NullInt64 `db:"updated_at"`
}
```

### type InsertKeyRatelimitParams

```go
type InsertKeyRatelimitParams struct {
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	KeyID       sql.NullString `db:"key_id"`
	Name        string         `db:"name"`
	Limit       int32          `db:"limit"`
	Duration    int64          `db:"duration"`
	AutoApply   bool           `db:"auto_apply"`
	CreatedAt   int64          `db:"created_at"`
	UpdatedAt   sql.NullInt64  `db:"updated_at"`
}
```

### type InsertKeyRoleParams

```go
type InsertKeyRoleParams struct {
	KeyID       string `db:"key_id"`
	RoleID      string `db:"role_id"`
	WorkspaceID string `db:"workspace_id"`
	CreatedAtM  int64  `db:"created_at_m"`
}
```

### type InsertKeySpaceParams

```go
type InsertKeySpaceParams struct {
	ID                 string         `db:"id"`
	WorkspaceID        string         `db:"workspace_id"`
	CreatedAtM         int64          `db:"created_at_m"`
	StoreEncryptedKeys bool           `db:"store_encrypted_keys"`
	DefaultPrefix      sql.NullString `db:"default_prefix"`
	DefaultBytes       sql.NullInt32  `db:"default_bytes"`
}
```

### type InsertPermissionParams

```go
type InsertPermissionParams struct {
	PermissionID string            `db:"permission_id"`
	WorkspaceID  string            `db:"workspace_id"`
	Name         string            `db:"name"`
	Slug         string            `db:"slug"`
	Description  dbtype.NullString `db:"description"`
	CreatedAtM   int64             `db:"created_at_m"`
}
```

### type InsertProjectParams

```go
type InsertProjectParams struct {
	ID               string         `db:"id"`
	WorkspaceID      string         `db:"workspace_id"`
	Name             string         `db:"name"`
	Slug             string         `db:"slug"`
	GitRepositoryUrl sql.NullString `db:"git_repository_url"`
	DefaultBranch    sql.NullString `db:"default_branch"`
	DeleteProtection sql.NullBool   `db:"delete_protection"`
	CreatedAt        int64          `db:"created_at"`
	UpdatedAt        sql.NullInt64  `db:"updated_at"`
}
```

### type InsertRatelimitNamespaceParams

```go
type InsertRatelimitNamespaceParams struct {
	ID          string `db:"id"`
	WorkspaceID string `db:"workspace_id"`
	Name        string `db:"name"`
	CreatedAt   int64  `db:"created_at"`
}
```

### type InsertRatelimitOverrideParams

```go
type InsertRatelimitOverrideParams struct {
	ID          string        `db:"id"`
	WorkspaceID string        `db:"workspace_id"`
	NamespaceID string        `db:"namespace_id"`
	Identifier  string        `db:"identifier"`
	Limit       int32         `db:"limit"`
	Duration    int32         `db:"duration"`
	CreatedAt   int64         `db:"created_at"`
	UpdatedAt   sql.NullInt64 `db:"updated_at"`
}
```

### type InsertRoleParams

```go
type InsertRoleParams struct {
	RoleID      string         `db:"role_id"`
	WorkspaceID string         `db:"workspace_id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAt   int64          `db:"created_at"`
}
```

### type InsertRolePermissionParams

```go
type InsertRolePermissionParams struct {
	RoleID       string `db:"role_id"`
	PermissionID string `db:"permission_id"`
	WorkspaceID  string `db:"workspace_id"`
	CreatedAtM   int64  `db:"created_at_m"`
}
```

### type InsertSentinelParams

```go
type InsertSentinelParams struct {
	ID                string          `db:"id"`
	WorkspaceID       string          `db:"workspace_id"`
	EnvironmentID     string          `db:"environment_id"`
	ProjectID         string          `db:"project_id"`
	K8sAddress        string          `db:"k8s_address"`
	K8sName           string          `db:"k8s_name"`
	Region            string          `db:"region"`
	Image             string          `db:"image"`
	Health            SentinelsHealth `db:"health"`
	DesiredReplicas   int32           `db:"desired_replicas"`
	AvailableReplicas int32           `db:"available_replicas"`
	CpuMillicores     int32           `db:"cpu_millicores"`
	MemoryMib         int32           `db:"memory_mib"`
	Version           uint64          `db:"version"`
	CreatedAt         int64           `db:"created_at"`
}
```

### type InsertWorkspaceParams

```go
type InsertWorkspaceParams struct {
	ID        string `db:"id"`
	OrgID     string `db:"org_id"`
	Name      string `db:"name"`
	Slug      string `db:"slug"`
	CreatedAt int64  `db:"created_at"`
}
```

### type Instance

```go
type Instance struct {
	Pk            uint64          `db:"pk"`
	ID            string          `db:"id"`
	DeploymentID  string          `db:"deployment_id"`
	WorkspaceID   string          `db:"workspace_id"`
	ProjectID     string          `db:"project_id"`
	Region        string          `db:"region"`
	K8sName       string          `db:"k8s_name"`
	Address       string          `db:"address"`
	CpuMillicores int32           `db:"cpu_millicores"`
	MemoryMib     int32           `db:"memory_mib"`
	Status        InstancesStatus `db:"status"`
}
```

### type InstancesStatus

```go
type InstancesStatus string
```

```go
const (
	InstancesStatusInactive InstancesStatus = "inactive"
	InstancesStatusPending  InstancesStatus = "pending"
	InstancesStatusRunning  InstancesStatus = "running"
	InstancesStatusFailed   InstancesStatus = "failed"
)
```

#### func (InstancesStatus) Scan

```go
func (e *InstancesStatus) Scan(src interface{}) error
```

### type Key

```go
type Key struct {
	Pk                 uint64         `db:"pk"`
	ID                 string         `db:"id"`
	KeyAuthID          string         `db:"key_auth_id"`
	Hash               string         `db:"hash"`
	Start              string         `db:"start"`
	WorkspaceID        string         `db:"workspace_id"`
	ForWorkspaceID     sql.NullString `db:"for_workspace_id"`
	Name               sql.NullString `db:"name"`
	OwnerID            sql.NullString `db:"owner_id"`
	IdentityID         sql.NullString `db:"identity_id"`
	Meta               sql.NullString `db:"meta"`
	Expires            sql.NullTime   `db:"expires"`
	CreatedAtM         int64          `db:"created_at_m"`
	UpdatedAtM         sql.NullInt64  `db:"updated_at_m"`
	DeletedAtM         sql.NullInt64  `db:"deleted_at_m"`
	RefillDay          sql.NullInt16  `db:"refill_day"`
	RefillAmount       sql.NullInt32  `db:"refill_amount"`
	LastRefillAt       sql.NullTime   `db:"last_refill_at"`
	Enabled            bool           `db:"enabled"`
	RemainingRequests  sql.NullInt32  `db:"remaining_requests"`
	RatelimitAsync     sql.NullBool   `db:"ratelimit_async"`
	RatelimitLimit     sql.NullInt32  `db:"ratelimit_limit"`
	RatelimitDuration  sql.NullInt64  `db:"ratelimit_duration"`
	Environment        sql.NullString `db:"environment"`
	PendingMigrationID sql.NullString `db:"pending_migration_id"`
}
```

### type KeyAuth

```go
type KeyAuth struct {
	Pk                 uint64         `db:"pk"`
	ID                 string         `db:"id"`
	WorkspaceID        string         `db:"workspace_id"`
	CreatedAtM         int64          `db:"created_at_m"`
	UpdatedAtM         sql.NullInt64  `db:"updated_at_m"`
	DeletedAtM         sql.NullInt64  `db:"deleted_at_m"`
	StoreEncryptedKeys bool           `db:"store_encrypted_keys"`
	DefaultPrefix      sql.NullString `db:"default_prefix"`
	DefaultBytes       sql.NullInt32  `db:"default_bytes"`
	SizeApprox         int32          `db:"size_approx"`
	SizeLastUpdatedAt  int64          `db:"size_last_updated_at"`
}
```

### type KeyData

```go
type KeyData struct {
	Key             Key
	Api             Api
	KeyAuth         KeyAuth
	Workspace       Workspace
	Identity        *Identity // Is optional
	EncryptedKey    sql.NullString
	EncryptionKeyID sql.NullString
	Roles           []RoleInfo
	Permissions     []PermissionInfo // Direct permissions attached to the key
	RolePermissions []PermissionInfo // Permissions inherited from roles
	Ratelimits      []RatelimitInfo
}
```

KeyData represents the complete data for a key including all relationships

#### func ToKeyData

```go
func ToKeyData[T KeyRow](row T) *KeyData
```

ToKeyData converts either query result into KeyData using generics

### type KeyFindForVerificationRatelimit

```go
type KeyFindForVerificationRatelimit struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Limit      int    `json:"limit"`
	Duration   int    `json:"duration"`
	AutoApply  int    `json:"auto_apply"`
	KeyID      string `json:"key_id"`
	IdentityID string `json:"identity_id"`
}
```

### type KeyMigration

```go
type KeyMigration struct {
	Pk          uint64                 `db:"pk"`
	ID          string                 `db:"id"`
	WorkspaceID string                 `db:"workspace_id"`
	Algorithm   KeyMigrationsAlgorithm `db:"algorithm"`
}
```

### type KeyMigrationError

```go
type KeyMigrationError struct {
	ID          string          `db:"id"`
	MigrationID string          `db:"migration_id"`
	CreatedAt   int64           `db:"created_at"`
	WorkspaceID string          `db:"workspace_id"`
	Message     json.RawMessage `db:"message"`
}
```

### type KeyMigrationsAlgorithm

```go
type KeyMigrationsAlgorithm string
```

```go
const (
	KeyMigrationsAlgorithmSha256                         KeyMigrationsAlgorithm = "sha256"
	KeyMigrationsAlgorithmGithubcomSeamapiPrefixedApiKey KeyMigrationsAlgorithm = "github.com/seamapi/prefixed-api-key"
)
```

#### func (KeyMigrationsAlgorithm) Scan

```go
func (e *KeyMigrationsAlgorithm) Scan(src interface{}) error
```

### type KeyRow

```go
type KeyRow interface {
	FindLiveKeyByHashRow | FindLiveKeyByIDRow | ListLiveKeysByKeySpaceIDRow
}
```

KeyRow constraint for types that can be converted to KeyData

### type KeysPermission

```go
type KeysPermission struct {
	Pk           uint64        `db:"pk"`
	TempID       sql.NullInt64 `db:"temp_id"`
	KeyID        string        `db:"key_id"`
	PermissionID string        `db:"permission_id"`
	WorkspaceID  string        `db:"workspace_id"`
	CreatedAtM   int64         `db:"created_at_m"`
	UpdatedAtM   sql.NullInt64 `db:"updated_at_m"`
}
```

### type KeysRole

```go
type KeysRole struct {
	Pk          uint64        `db:"pk"`
	KeyID       string        `db:"key_id"`
	RoleID      string        `db:"role_id"`
	WorkspaceID string        `db:"workspace_id"`
	CreatedAtM  int64         `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64 `db:"updated_at_m"`
}
```

### type ListCiliumNetworkPoliciesByRegionParams

```go
type ListCiliumNetworkPoliciesByRegionParams struct {
	Region       string `db:"region"`
	Afterversion uint64 `db:"afterversion"`
	Limit        int32  `db:"limit"`
}
```

### type ListCiliumNetworkPoliciesByRegionRow

```go
type ListCiliumNetworkPoliciesByRegionRow struct {
	CiliumNetworkPolicy CiliumNetworkPolicy `db:"cilium_network_policy"`
	K8sNamespace        sql.NullString      `db:"k8s_namespace"`
}
```

### type ListDeploymentTopologyByRegionParams

```go
type ListDeploymentTopologyByRegionParams struct {
	Region       string `db:"region"`
	Afterversion uint64 `db:"afterversion"`
	Limit        int32  `db:"limit"`
}
```

### type ListDeploymentTopologyByRegionRow

```go
type ListDeploymentTopologyByRegionRow struct {
	DeploymentTopology DeploymentTopology `db:"deployment_topology"`
	Deployment         Deployment         `db:"deployment"`
	K8sNamespace       sql.NullString     `db:"k8s_namespace"`
}
```

### type ListDesiredDeploymentTopologyParams

```go
type ListDesiredDeploymentTopologyParams struct {
	Region           string                  `db:"region"`
	DesiredState     DeploymentsDesiredState `db:"desired_state"`
	PaginationCursor string                  `db:"pagination_cursor"`
	Limit            int32                   `db:"limit"`
}
```

### type ListDesiredDeploymentTopologyRow

```go
type ListDesiredDeploymentTopologyRow struct {
	DeploymentTopology DeploymentTopology `db:"deployment_topology"`
	Deployment         Deployment         `db:"deployment"`
	K8sNamespace       sql.NullString     `db:"k8s_namespace"`
}
```

### type ListDesiredNetworkPoliciesParams

```go
type ListDesiredNetworkPoliciesParams struct {
	Region           string `db:"region"`
	PaginationCursor string `db:"pagination_cursor"`
	Limit            int32  `db:"limit"`
}
```

### type ListDesiredNetworkPoliciesRow

```go
type ListDesiredNetworkPoliciesRow struct {
	CiliumNetworkPolicy CiliumNetworkPolicy `db:"cilium_network_policy"`
	K8sNamespace        sql.NullString      `db:"k8s_namespace"`
}
```

### type ListDesiredSentinelsParams

```go
type ListDesiredSentinelsParams struct {
	Region           string                `db:"region"`
	DesiredState     SentinelsDesiredState `db:"desired_state"`
	PaginationCursor string                `db:"pagination_cursor"`
	Limit            int32                 `db:"limit"`
}
```

### type ListExecutableChallengesRow

```go
type ListExecutableChallengesRow struct {
	WorkspaceID   string                      `db:"workspace_id"`
	ChallengeType AcmeChallengesChallengeType `db:"challenge_type"`
	Domain        string                      `db:"domain"`
}
```

### type ListIdentitiesParams

```go
type ListIdentitiesParams struct {
	WorkspaceID string `db:"workspace_id"`
	Deleted     bool   `db:"deleted"`
	IDCursor    string `db:"id_cursor"`
	Limit       int32  `db:"limit"`
}
```

### type ListIdentitiesRow

```go
type ListIdentitiesRow struct {
	ID          string        `db:"id"`
	ExternalID  string        `db:"external_id"`
	WorkspaceID string        `db:"workspace_id"`
	Environment string        `db:"environment"`
	Meta        []byte        `db:"meta"`
	Deleted     bool          `db:"deleted"`
	CreatedAt   int64         `db:"created_at"`
	UpdatedAt   sql.NullInt64 `db:"updated_at"`
	Ratelimits  interface{}   `db:"ratelimits"`
}
```

### type ListKeysByKeySpaceIDParams

```go
type ListKeysByKeySpaceIDParams struct {
	KeySpaceID string         `db:"key_space_id"`
	IDCursor   string         `db:"id_cursor"`
	IdentityID sql.NullString `db:"identity_id"`
	Limit      int32          `db:"limit"`
}
```

### type ListKeysByKeySpaceIDRow

```go
type ListKeysByKeySpaceIDRow struct {
	Key             Key            `db:"key"`
	IdentityID      sql.NullString `db:"identity_id"`
	ExternalID      sql.NullString `db:"external_id"`
	IdentityMeta    []byte         `db:"identity_meta"`
	EncryptedKey    sql.NullString `db:"encrypted_key"`
	EncryptionKeyID sql.NullString `db:"encryption_key_id"`
}
```

### type ListLiveKeysByKeySpaceIDParams

```go
type ListLiveKeysByKeySpaceIDParams struct {
	KeySpaceID string         `db:"key_space_id"`
	IDCursor   string         `db:"id_cursor"`
	IdentityID sql.NullString `db:"identity_id"`
	Limit      int32          `db:"limit"`
}
```

### type ListLiveKeysByKeySpaceIDRow

```go
type ListLiveKeysByKeySpaceIDRow struct {
	Pk                 uint64         `db:"pk"`
	ID                 string         `db:"id"`
	KeyAuthID          string         `db:"key_auth_id"`
	Hash               string         `db:"hash"`
	Start              string         `db:"start"`
	WorkspaceID        string         `db:"workspace_id"`
	ForWorkspaceID     sql.NullString `db:"for_workspace_id"`
	Name               sql.NullString `db:"name"`
	OwnerID            sql.NullString `db:"owner_id"`
	IdentityID         sql.NullString `db:"identity_id"`
	Meta               sql.NullString `db:"meta"`
	Expires            sql.NullTime   `db:"expires"`
	CreatedAtM         int64          `db:"created_at_m"`
	UpdatedAtM         sql.NullInt64  `db:"updated_at_m"`
	DeletedAtM         sql.NullInt64  `db:"deleted_at_m"`
	RefillDay          sql.NullInt16  `db:"refill_day"`
	RefillAmount       sql.NullInt32  `db:"refill_amount"`
	LastRefillAt       sql.NullTime   `db:"last_refill_at"`
	Enabled            bool           `db:"enabled"`
	RemainingRequests  sql.NullInt32  `db:"remaining_requests"`
	RatelimitAsync     sql.NullBool   `db:"ratelimit_async"`
	RatelimitLimit     sql.NullInt32  `db:"ratelimit_limit"`
	RatelimitDuration  sql.NullInt64  `db:"ratelimit_duration"`
	Environment        sql.NullString `db:"environment"`
	PendingMigrationID sql.NullString `db:"pending_migration_id"`
	IdentityTableID    sql.NullString `db:"identity_table_id"`
	IdentityExternalID sql.NullString `db:"identity_external_id"`
	IdentityMeta       []byte         `db:"identity_meta"`
	EncryptedKey       sql.NullString `db:"encrypted_key"`
	EncryptionKeyID    sql.NullString `db:"encryption_key_id"`
	Roles              interface{}    `db:"roles"`
	Permissions        interface{}    `db:"permissions"`
	RolePermissions    interface{}    `db:"role_permissions"`
	Ratelimits         interface{}    `db:"ratelimits"`
}
```

### type ListNetworkPolicyByRegionParams

```go
type ListNetworkPolicyByRegionParams struct {
	Region       string `db:"region"`
	Afterversion uint64 `db:"afterversion"`
	Limit        int32  `db:"limit"`
}
```

### type ListNetworkPolicyByRegionRow

```go
type ListNetworkPolicyByRegionRow struct {
	CiliumNetworkPolicy CiliumNetworkPolicy `db:"cilium_network_policy"`
	K8sNamespace        sql.NullString      `db:"k8s_namespace"`
}
```

### type ListPermissionsByKeyIDParams

```go
type ListPermissionsByKeyIDParams struct {
	KeyID string `db:"key_id"`
}
```

### type ListPermissionsParams

```go
type ListPermissionsParams struct {
	WorkspaceID string `db:"workspace_id"`
	IDCursor    string `db:"id_cursor"`
	Limit       int32  `db:"limit"`
}
```

### type ListRatelimitOverridesByNamespaceIDParams

```go
type ListRatelimitOverridesByNamespaceIDParams struct {
	WorkspaceID string `db:"workspace_id"`
	NamespaceID string `db:"namespace_id"`
	CursorID    string `db:"cursor_id"`
	Limit       int32  `db:"limit"`
}
```

### type ListRatelimitsByKeyIDRow

```go
type ListRatelimitsByKeyIDRow struct {
	ID        string `db:"id"`
	Name      string `db:"name"`
	Limit     int32  `db:"limit"`
	Duration  int64  `db:"duration"`
	AutoApply bool   `db:"auto_apply"`
}
```

### type ListRatelimitsByKeyIDsRow

```go
type ListRatelimitsByKeyIDsRow struct {
	ID        string         `db:"id"`
	KeyID     sql.NullString `db:"key_id"`
	Name      string         `db:"name"`
	Limit     int32          `db:"limit"`
	Duration  int64          `db:"duration"`
	AutoApply bool           `db:"auto_apply"`
}
```

### type ListRolesByKeyIDRow

```go
type ListRolesByKeyIDRow struct {
	Pk          uint64         `db:"pk"`
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAtM  int64          `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64  `db:"updated_at_m"`
	Permissions interface{}    `db:"permissions"`
}
```

### type ListRolesParams

```go
type ListRolesParams struct {
	WorkspaceID string `db:"workspace_id"`
	IDCursor    string `db:"id_cursor"`
	Limit       int32  `db:"limit"`
}
```

### type ListRolesRow

```go
type ListRolesRow struct {
	Pk          uint64         `db:"pk"`
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAtM  int64          `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64  `db:"updated_at_m"`
	Permissions interface{}    `db:"permissions"`
}
```

### type ListSentinelsByRegionParams

```go
type ListSentinelsByRegionParams struct {
	Region       string `db:"region"`
	Afterversion uint64 `db:"afterversion"`
	Limit        int32  `db:"limit"`
}
```

### type ListWorkspacesForQuotaCheckRow

```go
type ListWorkspacesForQuotaCheckRow struct {
	ID               string         `db:"id"`
	OrgID            string         `db:"org_id"`
	Name             string         `db:"name"`
	StripeCustomerID sql.NullString `db:"stripe_customer_id"`
	Tier             sql.NullString `db:"tier"`
	Enabled          bool           `db:"enabled"`
	RequestsPerMonth sql.NullInt64  `db:"requests_per_month"`
}
```

### type ListWorkspacesRow

```go
type ListWorkspacesRow struct {
	Workspace Workspace `db:"workspace"`
	Quotas    Quotum    `db:"quotum"`
}
```

### type NullAcmeChallengesChallengeType

```go
type NullAcmeChallengesChallengeType struct {
	AcmeChallengesChallengeType AcmeChallengesChallengeType
	Valid                       bool // Valid is true if AcmeChallengesChallengeType is not NULL
}
```

#### func (NullAcmeChallengesChallengeType) Scan

```go
func (ns *NullAcmeChallengesChallengeType) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullAcmeChallengesChallengeType) Value

```go
func (ns NullAcmeChallengesChallengeType) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullAcmeChallengesStatus

```go
type NullAcmeChallengesStatus struct {
	AcmeChallengesStatus AcmeChallengesStatus
	Valid                bool // Valid is true if AcmeChallengesStatus is not NULL
}
```

#### func (NullAcmeChallengesStatus) Scan

```go
func (ns *NullAcmeChallengesStatus) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullAcmeChallengesStatus) Value

```go
func (ns NullAcmeChallengesStatus) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullApisAuthType

```go
type NullApisAuthType struct {
	ApisAuthType ApisAuthType
	Valid        bool // Valid is true if ApisAuthType is not NULL
}
```

#### func (NullApisAuthType) Scan

```go
func (ns *NullApisAuthType) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullApisAuthType) Value

```go
func (ns NullApisAuthType) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullCustomDomainsChallengeType

```go
type NullCustomDomainsChallengeType struct {
	CustomDomainsChallengeType CustomDomainsChallengeType
	Valid                      bool // Valid is true if CustomDomainsChallengeType is not NULL
}
```

#### func (NullCustomDomainsChallengeType) Scan

```go
func (ns *NullCustomDomainsChallengeType) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullCustomDomainsChallengeType) Value

```go
func (ns NullCustomDomainsChallengeType) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullCustomDomainsVerificationStatus

```go
type NullCustomDomainsVerificationStatus struct {
	CustomDomainsVerificationStatus CustomDomainsVerificationStatus
	Valid                           bool // Valid is true if CustomDomainsVerificationStatus is not NULL
}
```

#### func (NullCustomDomainsVerificationStatus) Scan

```go
func (ns *NullCustomDomainsVerificationStatus) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullCustomDomainsVerificationStatus) Value

```go
func (ns NullCustomDomainsVerificationStatus) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullDeploymentTopologyDesiredStatus

```go
type NullDeploymentTopologyDesiredStatus struct {
	DeploymentTopologyDesiredStatus DeploymentTopologyDesiredStatus
	Valid                           bool // Valid is true if DeploymentTopologyDesiredStatus is not NULL
}
```

#### func (NullDeploymentTopologyDesiredStatus) Scan

```go
func (ns *NullDeploymentTopologyDesiredStatus) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullDeploymentTopologyDesiredStatus) Value

```go
func (ns NullDeploymentTopologyDesiredStatus) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullDeploymentsDesiredState

```go
type NullDeploymentsDesiredState struct {
	DeploymentsDesiredState DeploymentsDesiredState
	Valid                   bool // Valid is true if DeploymentsDesiredState is not NULL
}
```

#### func (NullDeploymentsDesiredState) Scan

```go
func (ns *NullDeploymentsDesiredState) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullDeploymentsDesiredState) Value

```go
func (ns NullDeploymentsDesiredState) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullDeploymentsStatus

```go
type NullDeploymentsStatus struct {
	DeploymentsStatus DeploymentsStatus
	Valid             bool // Valid is true if DeploymentsStatus is not NULL
}
```

#### func (NullDeploymentsStatus) Scan

```go
func (ns *NullDeploymentsStatus) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullDeploymentsStatus) Value

```go
func (ns NullDeploymentsStatus) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullEnvironmentVariablesType

```go
type NullEnvironmentVariablesType struct {
	EnvironmentVariablesType EnvironmentVariablesType
	Valid                    bool // Valid is true if EnvironmentVariablesType is not NULL
}
```

#### func (NullEnvironmentVariablesType) Scan

```go
func (ns *NullEnvironmentVariablesType) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullEnvironmentVariablesType) Value

```go
func (ns NullEnvironmentVariablesType) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullFrontlineRoutesSticky

```go
type NullFrontlineRoutesSticky struct {
	FrontlineRoutesSticky FrontlineRoutesSticky
	Valid                 bool // Valid is true if FrontlineRoutesSticky is not NULL
}
```

#### func (NullFrontlineRoutesSticky) Scan

```go
func (ns *NullFrontlineRoutesSticky) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullFrontlineRoutesSticky) Value

```go
func (ns NullFrontlineRoutesSticky) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullInstancesStatus

```go
type NullInstancesStatus struct {
	InstancesStatus InstancesStatus
	Valid           bool // Valid is true if InstancesStatus is not NULL
}
```

#### func (NullInstancesStatus) Scan

```go
func (ns *NullInstancesStatus) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullInstancesStatus) Value

```go
func (ns NullInstancesStatus) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullKeyMigrationsAlgorithm

```go
type NullKeyMigrationsAlgorithm struct {
	KeyMigrationsAlgorithm KeyMigrationsAlgorithm
	Valid                  bool // Valid is true if KeyMigrationsAlgorithm is not NULL
}
```

#### func (NullKeyMigrationsAlgorithm) Scan

```go
func (ns *NullKeyMigrationsAlgorithm) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullKeyMigrationsAlgorithm) Value

```go
func (ns NullKeyMigrationsAlgorithm) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullRatelimitOverridesSharding

```go
type NullRatelimitOverridesSharding struct {
	RatelimitOverridesSharding RatelimitOverridesSharding
	Valid                      bool // Valid is true if RatelimitOverridesSharding is not NULL
}
```

#### func (NullRatelimitOverridesSharding) Scan

```go
func (ns *NullRatelimitOverridesSharding) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullRatelimitOverridesSharding) Value

```go
func (ns NullRatelimitOverridesSharding) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullSentinelsDesiredState

```go
type NullSentinelsDesiredState struct {
	SentinelsDesiredState SentinelsDesiredState
	Valid                 bool // Valid is true if SentinelsDesiredState is not NULL
}
```

#### func (NullSentinelsDesiredState) Scan

```go
func (ns *NullSentinelsDesiredState) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullSentinelsDesiredState) Value

```go
func (ns NullSentinelsDesiredState) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullSentinelsHealth

```go
type NullSentinelsHealth struct {
	SentinelsHealth SentinelsHealth
	Valid           bool // Valid is true if SentinelsHealth is not NULL
}
```

#### func (NullSentinelsHealth) Scan

```go
func (ns *NullSentinelsHealth) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullSentinelsHealth) Value

```go
func (ns NullSentinelsHealth) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullVercelBindingsEnvironment

```go
type NullVercelBindingsEnvironment struct {
	VercelBindingsEnvironment VercelBindingsEnvironment
	Valid                     bool // Valid is true if VercelBindingsEnvironment is not NULL
}
```

#### func (NullVercelBindingsEnvironment) Scan

```go
func (ns *NullVercelBindingsEnvironment) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullVercelBindingsEnvironment) Value

```go
func (ns NullVercelBindingsEnvironment) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullVercelBindingsResourceType

```go
type NullVercelBindingsResourceType struct {
	VercelBindingsResourceType VercelBindingsResourceType
	Valid                      bool // Valid is true if VercelBindingsResourceType is not NULL
}
```

#### func (NullVercelBindingsResourceType) Scan

```go
func (ns *NullVercelBindingsResourceType) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullVercelBindingsResourceType) Value

```go
func (ns NullVercelBindingsResourceType) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type NullWorkspacesPlan

```go
type NullWorkspacesPlan struct {
	WorkspacesPlan WorkspacesPlan
	Valid          bool // Valid is true if WorkspacesPlan is not NULL
}
```

#### func (NullWorkspacesPlan) Scan

```go
func (ns *NullWorkspacesPlan) Scan(value interface{}) error
```

Scan implements the Scanner interface.

#### func (NullWorkspacesPlan) Value

```go
func (ns NullWorkspacesPlan) Value() (driver.Value, error)
```

Value implements the driver Valuer interface.

### type Permission

```go
type Permission struct {
	Pk          uint64            `db:"pk"`
	ID          string            `db:"id"`
	WorkspaceID string            `db:"workspace_id"`
	Name        string            `db:"name"`
	Slug        string            `db:"slug"`
	Description dbtype.NullString `db:"description"`
	CreatedAtM  int64             `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64     `db:"updated_at_m"`
}
```

### type PermissionInfo

```go
type PermissionInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description dbtype.NullString `json:"description"`
}
```

### type Project

```go
type Project struct {
	Pk               uint64             `db:"pk"`
	ID               string             `db:"id"`
	WorkspaceID      string             `db:"workspace_id"`
	Name             string             `db:"name"`
	Slug             string             `db:"slug"`
	GitRepositoryUrl sql.NullString     `db:"git_repository_url"`
	LiveDeploymentID sql.NullString     `db:"live_deployment_id"`
	IsRolledBack     bool               `db:"is_rolled_back"`
	DefaultBranch    sql.NullString     `db:"default_branch"`
	DepotProjectID   sql.NullString     `db:"depot_project_id"`
	Command          dbtype.StringSlice `db:"command"`
	DeleteProtection sql.NullBool       `db:"delete_protection"`
	CreatedAt        int64              `db:"created_at"`
	UpdatedAt        sql.NullInt64      `db:"updated_at"`
}
```

### type Querier

```go
type Querier interface {
	//ClearAcmeChallengeTokens
	//
	//  UPDATE acme_challenges
	//  SET token = ?, authorization = ?, updated_at = ?
	//  WHERE domain_id = ?
	ClearAcmeChallengeTokens(ctx context.Context, db DBTX, arg ClearAcmeChallengeTokensParams) error
	//DeleteAcmeChallengeByDomainID
	//
	//  DELETE FROM acme_challenges WHERE domain_id = ?
	DeleteAcmeChallengeByDomainID(ctx context.Context, db DBTX, domainID string) error
	//DeleteAllKeyPermissionsByKeyID
	//
	//  DELETE FROM keys_permissions
	//  WHERE key_id = ?
	DeleteAllKeyPermissionsByKeyID(ctx context.Context, db DBTX, keyID string) error
	//DeleteAllKeyRolesByKeyID
	//
	//  DELETE FROM keys_roles
	//  WHERE key_id = ?
	DeleteAllKeyRolesByKeyID(ctx context.Context, db DBTX, keyID string) error
	//DeleteCustomDomainByID
	//
	//  DELETE FROM custom_domains WHERE id = ?
	DeleteCustomDomainByID(ctx context.Context, db DBTX, id string) error
	//DeleteDeploymentInstances
	//
	//  DELETE FROM instances
	//  WHERE deployment_id = ? AND region = ?
	DeleteDeploymentInstances(ctx context.Context, db DBTX, arg DeleteDeploymentInstancesParams) error
	//DeleteFrontlineRouteByFQDN
	//
	//  DELETE FROM frontline_routes WHERE fully_qualified_domain_name = ?
	DeleteFrontlineRouteByFQDN(ctx context.Context, db DBTX, fqdn string) error
	//DeleteIdentity
	//
	//  DELETE FROM identities
	//  WHERE id = ?
	//    AND workspace_id = ?
	DeleteIdentity(ctx context.Context, db DBTX, arg DeleteIdentityParams) error
	//DeleteInstance
	//
	//  DELETE FROM instances WHERE k8s_name = ? AND region = ?
	DeleteInstance(ctx context.Context, db DBTX, arg DeleteInstanceParams) error
	//DeleteKeyByID
	//
	//  DELETE k, kp, kr, rl, ek
	//  FROM `keys` k
	//  LEFT JOIN keys_permissions kp ON k.id = kp.key_id
	//  LEFT JOIN keys_roles kr ON k.id = kr.key_id
	//  LEFT JOIN ratelimits rl ON k.id = rl.key_id
	//  LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
	//  WHERE k.id = ?
	DeleteKeyByID(ctx context.Context, db DBTX, id string) error
	//DeleteKeyPermissionByKeyAndPermissionID
	//
	//  DELETE FROM keys_permissions
	//  WHERE key_id = ? AND permission_id = ?
	DeleteKeyPermissionByKeyAndPermissionID(ctx context.Context, db DBTX, arg DeleteKeyPermissionByKeyAndPermissionIDParams) error
	//DeleteManyKeyPermissionByKeyAndPermissionIDs
	//
	//  DELETE FROM keys_permissions
	//  WHERE key_id = ? AND permission_id IN (/*SLICE:ids*/?)
	DeleteManyKeyPermissionByKeyAndPermissionIDs(ctx context.Context, db DBTX, arg DeleteManyKeyPermissionByKeyAndPermissionIDsParams) error
	//DeleteManyKeyPermissionsByPermissionID
	//
	//  DELETE FROM keys_permissions
	//  WHERE permission_id = ?
	DeleteManyKeyPermissionsByPermissionID(ctx context.Context, db DBTX, permissionID string) error
	//DeleteManyKeyRolesByKeyAndRoleIDs
	//
	//  DELETE FROM keys_roles
	//  WHERE key_id = ? AND role_id IN(/*SLICE:role_ids*/?)
	DeleteManyKeyRolesByKeyAndRoleIDs(ctx context.Context, db DBTX, arg DeleteManyKeyRolesByKeyAndRoleIDsParams) error
	//DeleteManyKeyRolesByKeyID
	//
	//  DELETE FROM keys_roles
	//  WHERE key_id = ? AND role_id = ?
	DeleteManyKeyRolesByKeyID(ctx context.Context, db DBTX, arg DeleteManyKeyRolesByKeyIDParams) error
	//DeleteManyKeyRolesByRoleID
	//
	//  DELETE FROM keys_roles
	//  WHERE role_id = ?
	DeleteManyKeyRolesByRoleID(ctx context.Context, db DBTX, roleID string) error
	//DeleteManyRatelimitsByIDs
	//
	//  DELETE FROM ratelimits WHERE id IN (/*SLICE:ids*/?)
	DeleteManyRatelimitsByIDs(ctx context.Context, db DBTX, ids []string) error
	//DeleteManyRatelimitsByIdentityID
	//
	//  DELETE FROM ratelimits WHERE identity_id = ?
	DeleteManyRatelimitsByIdentityID(ctx context.Context, db DBTX, identityID sql.NullString) error
	//DeleteManyRolePermissionsByPermissionID
	//
	//  DELETE FROM roles_permissions
	//  WHERE permission_id = ?
	DeleteManyRolePermissionsByPermissionID(ctx context.Context, db DBTX, permissionID string) error
	//DeleteManyRolePermissionsByRoleID
	//
	//  DELETE FROM roles_permissions
	//  WHERE role_id = ?
	DeleteManyRolePermissionsByRoleID(ctx context.Context, db DBTX, roleID string) error
	//DeleteOldIdentityByExternalID
	//
	//  DELETE i, rl
	//  FROM identities i
	//  LEFT JOIN ratelimits rl ON rl.identity_id = i.id
	//  WHERE i.workspace_id = ?
	//    AND i.external_id = ?
	//    AND i.id != ?
	//    AND i.deleted = true
	DeleteOldIdentityByExternalID(ctx context.Context, db DBTX, arg DeleteOldIdentityByExternalIDParams) error
	//DeleteOldIdentityWithRatelimits
	//
	//  DELETE i, rl
	//  FROM identities i
	//  LEFT JOIN ratelimits rl ON rl.identity_id = i.id
	//  WHERE i.workspace_id = ?
	//    AND (i.id = ? OR i.external_id = ?)
	//    AND i.deleted = true
	DeleteOldIdentityWithRatelimits(ctx context.Context, db DBTX, arg DeleteOldIdentityWithRatelimitsParams) error
	//DeletePermission
	//
	//  DELETE FROM permissions
	//  WHERE id = ?
	DeletePermission(ctx context.Context, db DBTX, permissionID string) error
	//DeleteRatelimit
	//
	//  DELETE FROM `ratelimits` WHERE id = ?
	DeleteRatelimit(ctx context.Context, db DBTX, id string) error
	//DeleteRatelimitNamespace
	//
	//  UPDATE `ratelimit_namespaces`
	//  SET deleted_at_m = ?
	//  WHERE id = ?
	DeleteRatelimitNamespace(ctx context.Context, db DBTX, arg DeleteRatelimitNamespaceParams) (sql.Result, error)
	//DeleteRoleByID
	//
	//  DELETE FROM roles
	//  WHERE id = ?
	DeleteRoleByID(ctx context.Context, db DBTX, roleID string) error
	//FindAcmeChallengeByToken
	//
	//  SELECT pk, domain_id, workspace_id, token, challenge_type, authorization, status, expires_at, created_at, updated_at FROM acme_challenges WHERE workspace_id = ? AND domain_id = ? AND token = ?
	FindAcmeChallengeByToken(ctx context.Context, db DBTX, arg FindAcmeChallengeByTokenParams) (AcmeChallenge, error)
	//FindAcmeUserByWorkspaceID
	//
	//  SELECT pk, id, workspace_id, encrypted_key, registration_uri, created_at, updated_at FROM acme_users WHERE workspace_id = ? LIMIT 1
	FindAcmeUserByWorkspaceID(ctx context.Context, db DBTX, workspaceID string) (AcmeUser, error)
	//FindApiByID
	//
	//  SELECT pk, id, name, workspace_id, ip_whitelist, auth_type, key_auth_id, created_at_m, updated_at_m, deleted_at_m, delete_protection FROM apis WHERE id = ?
	FindApiByID(ctx context.Context, db DBTX, id string) (Api, error)
	//FindAuditLogTargetByID
	//
	//  SELECT audit_log_target.pk, audit_log_target.workspace_id, audit_log_target.bucket_id, audit_log_target.bucket, audit_log_target.audit_log_id, audit_log_target.display_name, audit_log_target.type, audit_log_target.id, audit_log_target.name, audit_log_target.meta, audit_log_target.created_at, audit_log_target.updated_at, audit_log.pk, audit_log.id, audit_log.workspace_id, audit_log.bucket, audit_log.bucket_id, audit_log.event, audit_log.time, audit_log.display, audit_log.remote_ip, audit_log.user_agent, audit_log.actor_type, audit_log.actor_id, audit_log.actor_name, audit_log.actor_meta, audit_log.created_at, audit_log.updated_at
	//  FROM audit_log_target
	//  JOIN audit_log ON audit_log.id = audit_log_target.audit_log_id
	//  WHERE audit_log_target.id = ?
	FindAuditLogTargetByID(ctx context.Context, db DBTX, id string) ([]FindAuditLogTargetByIDRow, error)
	//FindCertificateByHostname
	//
	//  SELECT pk, id, workspace_id, hostname, certificate, encrypted_private_key, created_at, updated_at FROM certificates WHERE hostname = ?
	FindCertificateByHostname(ctx context.Context, db DBTX, hostname string) (Certificate, error)
	//FindCertificatesByHostnames
	//
	//  SELECT pk, id, workspace_id, hostname, certificate, encrypted_private_key, created_at, updated_at FROM certificates WHERE hostname IN (/*SLICE:hostnames*/?)
	FindCertificatesByHostnames(ctx context.Context, db DBTX, hostnames []string) ([]Certificate, error)
	//FindCiliumNetworkPoliciesByEnvironmentID
	//
	//  SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, region, policy, version, created_at, updated_at FROM cilium_network_policies WHERE environment_id = ?
	FindCiliumNetworkPoliciesByEnvironmentID(ctx context.Context, db DBTX, environmentID string) ([]CiliumNetworkPolicy, error)
	//FindCiliumNetworkPolicyByIDAndRegion
	//
	//  SELECT
	//      n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
	//      w.k8s_namespace
	//  FROM `cilium_network_policies` n
	//  JOIN `workspaces` w ON w.id = n.workspace_id
	//  WHERE n.region = ? AND n.id = ?
	//  LIMIT 1
	FindCiliumNetworkPolicyByIDAndRegion(ctx context.Context, db DBTX, arg FindCiliumNetworkPolicyByIDAndRegionParams) (FindCiliumNetworkPolicyByIDAndRegionRow, error)
	//FindClickhouseWorkspaceSettingsByWorkspaceID
	//
	//  SELECT
	//      c.pk, c.workspace_id, c.username, c.password_encrypted, c.quota_duration_seconds, c.max_queries_per_window, c.max_execution_time_per_window, c.max_query_execution_time, c.max_query_memory_bytes, c.max_query_result_rows, c.created_at, c.updated_at,
	//      q.pk, q.workspace_id, q.requests_per_month, q.logs_retention_days, q.audit_logs_retention_days, q.team
	//  FROM `clickhouse_workspace_settings` c
	//  JOIN `quota` q ON c.workspace_id = q.workspace_id
	//  WHERE c.workspace_id = ?
	FindClickhouseWorkspaceSettingsByWorkspaceID(ctx context.Context, db DBTX, workspaceID string) (FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error)
	//FindCustomDomainByDomain
	//
	//  SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at
	//  FROM custom_domains
	//  WHERE domain = ?
	FindCustomDomainByDomain(ctx context.Context, db DBTX, domain string) (CustomDomain, error)
	//FindCustomDomainByDomainOrWildcard
	//
	//  SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at FROM custom_domains
	//  WHERE domain IN (?, ?)
	//  ORDER BY
	//      CASE WHEN domain = ? THEN 0 ELSE 1 END
	//  LIMIT 1
	FindCustomDomainByDomainOrWildcard(ctx context.Context, db DBTX, arg FindCustomDomainByDomainOrWildcardParams) (CustomDomain, error)
	//FindCustomDomainById
	//
	//  SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at
	//  FROM custom_domains
	//  WHERE id = ?
	FindCustomDomainById(ctx context.Context, db DBTX, id string) (CustomDomain, error)
	//FindCustomDomainWithCertByDomain
	//
	//  SELECT
	//      cd.pk, cd.id, cd.workspace_id, cd.project_id, cd.environment_id, cd.domain, cd.challenge_type, cd.verification_status, cd.verification_token, cd.ownership_verified, cd.cname_verified, cd.target_cname, cd.last_checked_at, cd.check_attempts, cd.verification_error, cd.invocation_id, cd.created_at, cd.updated_at,
	//      c.id AS certificate_id
	//  FROM custom_domains cd
	//  LEFT JOIN certificates c ON c.hostname = cd.domain
	//  WHERE cd.domain = ?
	FindCustomDomainWithCertByDomain(ctx context.Context, db DBTX, domain string) (FindCustomDomainWithCertByDomainRow, error)
	//FindDeploymentById
	//
	//  SELECT pk, id, k8s_name, workspace_id, project_id, environment_id, image, build_id, git_commit_sha, git_branch, git_commit_message, git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp, sentinel_config, openapi_spec, cpu_millicores, memory_mib, desired_state, encrypted_environment_variables, command, status, created_at, updated_at FROM `deployments` WHERE id = ?
	FindDeploymentById(ctx context.Context, db DBTX, id string) (Deployment, error)
	//FindDeploymentByK8sName
	//
	//  SELECT pk, id, k8s_name, workspace_id, project_id, environment_id, image, build_id, git_commit_sha, git_branch, git_commit_message, git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp, sentinel_config, openapi_spec, cpu_millicores, memory_mib, desired_state, encrypted_environment_variables, command, status, created_at, updated_at FROM `deployments` WHERE k8s_name = ?
	FindDeploymentByK8sName(ctx context.Context, db DBTX, k8sName string) (Deployment, error)
	// Returns all regions where a deployment is configured.
	// Used for fan-out: when a deployment changes, emit state_change to each region.
	//
	//  SELECT region
	//  FROM `deployment_topology`
	//  WHERE deployment_id = ?
	FindDeploymentRegions(ctx context.Context, db DBTX, deploymentID string) ([]string, error)
	//FindDeploymentTopologyByIDAndRegion
	//
	//  SELECT
	//      d.id,
	//      d.k8s_name,
	//      w.k8s_namespace,
	//      d.workspace_id,
	//      d.project_id,
	//      d.environment_id,
	//      d.build_id,
	//      d.image,
	//      dt.region,
	//      d.cpu_millicores,
	//      d.memory_mib,
	//      dt.desired_replicas,
	//      d.desired_state,
	//      d.encrypted_environment_variables
	//  FROM `deployment_topology` dt
	//  INNER JOIN `deployments` d ON dt.deployment_id = d.id
	//  INNER JOIN `workspaces` w ON d.workspace_id = w.id
	//  WHERE  dt.region = ?
	//      AND dt.deployment_id = ?
	//  LIMIT 1
	FindDeploymentTopologyByIDAndRegion(ctx context.Context, db DBTX, arg FindDeploymentTopologyByIDAndRegionParams) (FindDeploymentTopologyByIDAndRegionRow, error)
	//FindEnvironmentById
	//
	//  SELECT id, workspace_id, project_id, slug, description
	//  FROM environments
	//  WHERE id = ?
	FindEnvironmentById(ctx context.Context, db DBTX, id string) (FindEnvironmentByIdRow, error)
	//FindEnvironmentByProjectIdAndSlug
	//
	//  SELECT pk, id, workspace_id, project_id, slug, description, sentinel_config, delete_protection, created_at, updated_at
	//  FROM environments
	//  WHERE workspace_id = ?
	//    AND project_id = ?
	//    AND slug = ?
	FindEnvironmentByProjectIdAndSlug(ctx context.Context, db DBTX, arg FindEnvironmentByProjectIdAndSlugParams) (Environment, error)
	//FindEnvironmentVariablesByEnvironmentId
	//
	//  SELECT `key`, value
	//  FROM environment_variables
	//  WHERE environment_id = ?
	FindEnvironmentVariablesByEnvironmentId(ctx context.Context, db DBTX, environmentID string) ([]FindEnvironmentVariablesByEnvironmentIdRow, error)
	//FindFrontlineRouteByFQDN
	//
	//  SELECT pk, id, project_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at, updated_at FROM frontline_routes WHERE fully_qualified_domain_name = ?
	FindFrontlineRouteByFQDN(ctx context.Context, db DBTX, fullyQualifiedDomainName string) (FrontlineRoute, error)
	//FindFrontlineRouteForPromotion
	//
	//  SELECT
	//      id,
	//      project_id,
	//      environment_id,
	//      fully_qualified_domain_name,
	//      deployment_id,
	//      sticky,
	//      created_at,
	//      updated_at
	//  FROM frontline_routes
	//  WHERE
	//    environment_id = ?
	//    AND sticky IN (/*SLICE:sticky*/?)
	//  ORDER BY created_at ASC
	FindFrontlineRouteForPromotion(ctx context.Context, db DBTX, arg FindFrontlineRouteForPromotionParams) ([]FindFrontlineRouteForPromotionRow, error)
	//FindFrontlineRoutesByDeploymentID
	//
	//  SELECT pk, id, project_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at, updated_at FROM frontline_routes WHERE deployment_id = ?
	FindFrontlineRoutesByDeploymentID(ctx context.Context, db DBTX, deploymentID string) ([]FrontlineRoute, error)
	//FindFrontlineRoutesForRollback
	//
	//  SELECT
	//      id,
	//      project_id,
	//      environment_id,
	//      fully_qualified_domain_name,
	//      deployment_id,
	//      sticky,
	//      created_at,
	//      updated_at
	//  FROM frontline_routes
	//  WHERE
	//    environment_id = ?
	//    AND sticky IN (/*SLICE:sticky*/?)
	//  ORDER BY created_at ASC
	FindFrontlineRoutesForRollback(ctx context.Context, db DBTX, arg FindFrontlineRoutesForRollbackParams) ([]FindFrontlineRoutesForRollbackRow, error)
	//FindGithubRepoConnection
	//
	//  SELECT
	//      pk,
	//      project_id,
	//      installation_id,
	//      repository_id,
	//      repository_full_name,
	//      created_at,
	//      updated_at
	//  FROM github_repo_connections
	//  WHERE installation_id = ?
	//    AND repository_id = ?
	FindGithubRepoConnection(ctx context.Context, db DBTX, arg FindGithubRepoConnectionParams) (GithubRepoConnection, error)
	//FindIdentities
	//
	//  SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
	//  FROM identities
	//  WHERE workspace_id = ?
	//   AND deleted = ?
	//   AND (external_id IN(/*SLICE:identities*/?) OR id IN (/*SLICE:identities*/?))
	FindIdentities(ctx context.Context, db DBTX, arg FindIdentitiesParams) ([]Identity, error)
	//FindIdentitiesByExternalId
	//
	//  SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
	//  FROM identities
	//  WHERE workspace_id = ? AND external_id IN (/*SLICE:externalIds*/?) AND deleted = ?
	FindIdentitiesByExternalId(ctx context.Context, db DBTX, arg FindIdentitiesByExternalIdParams) ([]Identity, error)
	//FindIdentity
	//
	//  SELECT
	//      i.pk, i.id, i.external_id, i.workspace_id, i.environment, i.meta, i.deleted, i.created_at, i.updated_at,
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', rl.id,
	//                  'name', rl.name,
	//                  'key_id', rl.key_id,
	//                  'identity_id', rl.identity_id,
	//                  'limit', rl.`limit`,
	//                  'duration', rl.duration,
	//                  'auto_apply', rl.auto_apply = 1
	//              )
	//          )
	//          FROM ratelimits rl WHERE rl.identity_id = i.id),
	//          JSON_ARRAY()
	//      ) as ratelimits
	//  FROM identities i
	//  JOIN (
	//      SELECT id1.id FROM identities id1
	//      WHERE id1.id = ?
	//        AND id1.workspace_id = ?
	//        AND id1.deleted = ?
	//      UNION ALL
	//      SELECT id2.id FROM identities id2
	//      WHERE id2.workspace_id = ?
	//        AND id2.external_id = ?
	//        AND id2.deleted = ?
	//  ) AS identity_lookup ON i.id = identity_lookup.id
	//  LIMIT 1
	FindIdentity(ctx context.Context, db DBTX, arg FindIdentityParams) (FindIdentityRow, error)
	//FindIdentityByExternalID
	//
	//  SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
	//  FROM identities
	//  WHERE workspace_id = ?
	//    AND external_id = ?
	//    AND deleted = ?
	FindIdentityByExternalID(ctx context.Context, db DBTX, arg FindIdentityByExternalIDParams) (Identity, error)
	//FindIdentityByID
	//
	//  SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
	//  FROM identities
	//  WHERE workspace_id = ?
	//    AND id = ?
	//    AND deleted = ?
	FindIdentityByID(ctx context.Context, db DBTX, arg FindIdentityByIDParams) (Identity, error)
	//FindInstanceByPodName
	//
	//  SELECT
	//   pk, id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status
	//  FROM instances
	//    WHERE k8s_name = ? AND region = ?
	FindInstanceByPodName(ctx context.Context, db DBTX, arg FindInstanceByPodNameParams) (Instance, error)
	//FindInstancesByDeploymentId
	//
	//  SELECT
	//   pk, id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status
	//  FROM instances
	//  WHERE deployment_id = ?
	FindInstancesByDeploymentId(ctx context.Context, db DBTX, deploymentid string) ([]Instance, error)
	//FindInstancesByDeploymentIdAndRegion
	//
	//  SELECT
	//   pk, id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status
	//  FROM instances
	//  WHERE deployment_id = ? AND region = ?
	FindInstancesByDeploymentIdAndRegion(ctx context.Context, db DBTX, arg FindInstancesByDeploymentIdAndRegionParams) ([]Instance, error)
	//FindKeyAuthsByIds
	//
	//  SELECT ka.id as key_auth_id, a.id as api_id
	//  FROM apis a
	//  JOIN key_auth as ka ON ka.id = a.key_auth_id
	//  WHERE a.workspace_id = ?
	//      AND a.id IN (/*SLICE:api_ids*/?)
	//      AND ka.deleted_at_m IS NULL
	//      AND a.deleted_at_m IS NULL
	FindKeyAuthsByIds(ctx context.Context, db DBTX, arg FindKeyAuthsByIdsParams) ([]FindKeyAuthsByIdsRow, error)
	//FindKeyAuthsByKeyAuthIds
	//
	//  SELECT ka.id as key_auth_id, a.id as api_id
	//  FROM key_auth as ka
	//  JOIN apis a ON a.key_auth_id = ka.id
	//  WHERE a.workspace_id = ?
	//      AND ka.id IN (/*SLICE:key_auth_ids*/?)
	//      AND ka.deleted_at_m IS NULL
	//      AND a.deleted_at_m IS NULL
	FindKeyAuthsByKeyAuthIds(ctx context.Context, db DBTX, arg FindKeyAuthsByKeyAuthIdsParams) ([]FindKeyAuthsByKeyAuthIdsRow, error)
	//FindKeyByID
	//
	//  SELECT pk, id, key_auth_id, hash, start, workspace_id, for_workspace_id, name, owner_id, identity_id, meta, expires, created_at_m, updated_at_m, deleted_at_m, refill_day, refill_amount, last_refill_at, enabled, remaining_requests, ratelimit_async, ratelimit_limit, ratelimit_duration, environment, pending_migration_id FROM `keys` k
	//  WHERE k.id = ?
	FindKeyByID(ctx context.Context, db DBTX, id string) (Key, error)
	//FindKeyCredits
	//
	//  SELECT remaining_requests FROM `keys` k WHERE k.id = ?
	FindKeyCredits(ctx context.Context, db DBTX, id string) (sql.NullInt32, error)
	//FindKeyEncryptionByKeyID
	//
	//  SELECT pk, workspace_id, key_id, created_at, updated_at, encrypted, encryption_key_id FROM encrypted_keys WHERE key_id = ?
	FindKeyEncryptionByKeyID(ctx context.Context, db DBTX, keyID string) (EncryptedKey, error)
	//FindKeyForVerification
	//
	//  select k.id,
	//         k.key_auth_id,
	//         k.workspace_id,
	//         k.for_workspace_id,
	//         k.name,
	//         k.meta,
	//         k.expires,
	//         k.deleted_at_m,
	//         k.refill_day,
	//         k.refill_amount,
	//         k.last_refill_at,
	//         k.enabled,
	//         k.remaining_requests,
	//         k.pending_migration_id,
	//         a.ip_whitelist,
	//         a.workspace_id  as api_workspace_id,
	//         a.id            as api_id,
	//         a.deleted_at_m  as api_deleted_at_m,
	//
	//         COALESCE(
	//                 (SELECT JSON_ARRAYAGG(name)
	//                  FROM (SELECT name
	//                        FROM keys_roles kr
	//                                 JOIN roles r ON r.id = kr.role_id
	//                        WHERE kr.key_id = k.id) as roles),
	//                 JSON_ARRAY()
	//         )               as roles,
	//
	//         COALESCE(
	//                 (SELECT JSON_ARRAYAGG(slug)
	//                  FROM (SELECT slug
	//                        FROM keys_permissions kp
	//                                 JOIN permissions p ON kp.permission_id = p.id
	//                        WHERE kp.key_id = k.id
	//
	//                        UNION ALL
	//
	//                        SELECT slug
	//                        FROM keys_roles kr
	//                                 JOIN roles_permissions rp ON kr.role_id = rp.role_id
	//                                 JOIN permissions p ON rp.permission_id = p.id
	//                        WHERE kr.key_id = k.id) as combined_perms),
	//                 JSON_ARRAY()
	//         )               as permissions,
	//
	//         coalesce(
	//                 (select json_arrayagg(
	//                      json_object(
	//                         'id', rl.id,
	//                         'name', rl.name,
	//                         'key_id', rl.key_id,
	//                         'identity_id', rl.identity_id,
	//                         'limit', rl.limit,
	//                         'duration', rl.duration,
	//                         'auto_apply', rl.auto_apply
	//                      )
	//                  )
	//                  from `ratelimits` rl
	//                  where rl.key_id = k.id
	//                     OR rl.identity_id = i.id),
	//                 json_array()
	//         ) as ratelimits,
	//
	//         i.id as identity_id,
	//         i.external_id,
	//         i.meta          as identity_meta,
	//         ka.deleted_at_m as key_auth_deleted_at_m,
	//         ws.enabled      as workspace_enabled,
	//         fws.enabled     as for_workspace_enabled
	//  from `keys` k
	//           JOIN apis a USING (key_auth_id)
	//           JOIN key_auth ka ON ka.id = k.key_auth_id
	//           JOIN workspaces ws ON ws.id = k.workspace_id
	//           LEFT JOIN workspaces fws ON fws.id = k.for_workspace_id
	//           LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = 0
	//  where k.hash = ?
	//    and k.deleted_at_m is null
	FindKeyForVerification(ctx context.Context, db DBTX, hash string) (FindKeyForVerificationRow, error)
	//FindKeyMigrationByID
	//
	//  SELECT
	//      id,
	//      workspace_id,
	//      algorithm
	//  FROM key_migrations
	//  WHERE id = ?
	//  and workspace_id = ?
	FindKeyMigrationByID(ctx context.Context, db DBTX, arg FindKeyMigrationByIDParams) (FindKeyMigrationByIDRow, error)
	//FindKeyRoleByKeyAndRoleID
	//
	//  SELECT pk, key_id, role_id, workspace_id, created_at_m, updated_at_m
	//  FROM keys_roles
	//  WHERE key_id = ?
	//    AND role_id = ?
	FindKeyRoleByKeyAndRoleID(ctx context.Context, db DBTX, arg FindKeyRoleByKeyAndRoleIDParams) ([]KeysRole, error)
	//FindKeySpaceByID
	//
	//  SELECT pk, id, workspace_id, created_at_m, updated_at_m, deleted_at_m, store_encrypted_keys, default_prefix, default_bytes, size_approx, size_last_updated_at FROM `key_auth` WHERE id = ?
	FindKeySpaceByID(ctx context.Context, db DBTX, id string) (KeyAuth, error)
	//FindKeysByHash
	//
	//  SELECT id, hash FROM `keys` WHERE hash IN (/*SLICE:hashes*/?)
	FindKeysByHash(ctx context.Context, db DBTX, hashes []string) ([]FindKeysByHashRow, error)
	//FindLiveApiByID
	//
	//  SELECT apis.pk, apis.id, apis.name, apis.workspace_id, apis.ip_whitelist, apis.auth_type, apis.key_auth_id, apis.created_at_m, apis.updated_at_m, apis.deleted_at_m, apis.delete_protection, ka.pk, ka.id, ka.workspace_id, ka.created_at_m, ka.updated_at_m, ka.deleted_at_m, ka.store_encrypted_keys, ka.default_prefix, ka.default_bytes, ka.size_approx, ka.size_last_updated_at
	//  FROM apis
	//  JOIN key_auth as ka ON ka.id = apis.key_auth_id
	//  WHERE apis.id = ?
	//      AND ka.deleted_at_m IS NULL
	//      AND apis.deleted_at_m IS NULL
	//  LIMIT 1
	FindLiveApiByID(ctx context.Context, db DBTX, id string) (FindLiveApiByIDRow, error)
	//FindLiveKeyByHash
	//
	//  SELECT
	//      k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
	//      a.pk, a.id, a.name, a.workspace_id, a.ip_whitelist, a.auth_type, a.key_auth_id, a.created_at_m, a.updated_at_m, a.deleted_at_m, a.delete_protection,
	//      ka.pk, ka.id, ka.workspace_id, ka.created_at_m, ka.updated_at_m, ka.deleted_at_m, ka.store_encrypted_keys, ka.default_prefix, ka.default_bytes, ka.size_approx, ka.size_last_updated_at,
	//      ws.pk, ws.id, ws.org_id, ws.name, ws.slug, ws.k8s_namespace, ws.partition_id, ws.plan, ws.tier, ws.stripe_customer_id, ws.stripe_subscription_id, ws.beta_features, ws.features, ws.subscriptions, ws.enabled, ws.delete_protection, ws.created_at_m, ws.updated_at_m, ws.deleted_at_m,
	//      i.id as identity_table_id,
	//      i.external_id as identity_external_id,
	//      i.meta as identity_meta,
	//      ek.encrypted as encrypted_key,
	//      ek.encryption_key_id as encryption_key_id,
	//
	//      -- Roles with both IDs and names
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', r.id,
	//                  'name', r.name,
	//                  'description', r.description
	//              )
	//          )
	//          FROM keys_roles kr
	//          JOIN roles r ON r.id = kr.role_id
	//          WHERE kr.key_id = k.id),
	//          JSON_ARRAY()
	//      ) as roles,
	//
	//      -- Direct permissions attached to the key
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', p.id,
	//                  'name', p.name,
	//                  'slug', p.slug,
	//                  'description', p.description
	//              )
	//          )
	//          FROM keys_permissions kp
	//          JOIN permissions p ON kp.permission_id = p.id
	//          WHERE kp.key_id = k.id),
	//          JSON_ARRAY()
	//      ) as permissions,
	//
	//      -- Permissions from roles
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', p.id,
	//                  'name', p.name,
	//                  'slug', p.slug,
	//                  'description', p.description
	//              )
	//          )
	//          FROM keys_roles kr
	//          JOIN roles_permissions rp ON kr.role_id = rp.role_id
	//          JOIN permissions p ON rp.permission_id = p.id
	//          WHERE kr.key_id = k.id),
	//          JSON_ARRAY()
	//      ) as role_permissions,
	//
	//      -- Rate limits
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', rl.id,
	//                  'name', rl.name,
	//                  'key_id', rl.key_id,
	//                  'identity_id', rl.identity_id,
	//                  'limit', rl.`limit`,
	//                  'duration', rl.duration,
	//                  'auto_apply', rl.auto_apply = 1
	//              )
	//          )
	//          FROM ratelimits rl
	//          WHERE rl.key_id = k.id OR rl.identity_id = i.id),
	//          JSON_ARRAY()
	//      ) as ratelimits
	//
	//  FROM `keys` k
	//  JOIN apis a ON a.key_auth_id = k.key_auth_id
	//  JOIN key_auth ka ON ka.id = k.key_auth_id
	//  JOIN workspaces ws ON ws.id = k.workspace_id
	//  LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
	//  LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
	//  WHERE k.hash = ?
	//      AND k.deleted_at_m IS NULL
	//      AND a.deleted_at_m IS NULL
	//      AND ka.deleted_at_m IS NULL
	//      AND ws.deleted_at_m IS NULL
	FindLiveKeyByHash(ctx context.Context, db DBTX, hash string) (FindLiveKeyByHashRow, error)
	//FindLiveKeyByID
	//
	//  SELECT
	//      k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
	//      a.pk, a.id, a.name, a.workspace_id, a.ip_whitelist, a.auth_type, a.key_auth_id, a.created_at_m, a.updated_at_m, a.deleted_at_m, a.delete_protection,
	//      ka.pk, ka.id, ka.workspace_id, ka.created_at_m, ka.updated_at_m, ka.deleted_at_m, ka.store_encrypted_keys, ka.default_prefix, ka.default_bytes, ka.size_approx, ka.size_last_updated_at,
	//      ws.pk, ws.id, ws.org_id, ws.name, ws.slug, ws.k8s_namespace, ws.partition_id, ws.plan, ws.tier, ws.stripe_customer_id, ws.stripe_subscription_id, ws.beta_features, ws.features, ws.subscriptions, ws.enabled, ws.delete_protection, ws.created_at_m, ws.updated_at_m, ws.deleted_at_m,
	//      i.id as identity_table_id,
	//      i.external_id as identity_external_id,
	//      i.meta as identity_meta,
	//      ek.encrypted as encrypted_key,
	//      ek.encryption_key_id as encryption_key_id,
	//
	//      -- Roles with both IDs and names
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', r.id,
	//                  'name', r.name,
	//                  'description', r.description
	//              )
	//          )
	//          FROM keys_roles kr
	//          JOIN roles r ON r.id = kr.role_id
	//          WHERE kr.key_id = k.id),
	//          JSON_ARRAY()
	//      ) as roles,
	//
	//      -- Direct permissions attached to the key
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', p.id,
	//                  'name', p.name,
	//                  'slug', p.slug,
	//                  'description', p.description
	//              )
	//          )
	//          FROM keys_permissions kp
	//          JOIN permissions p ON kp.permission_id = p.id
	//          WHERE kp.key_id = k.id),
	//          JSON_ARRAY()
	//      ) as permissions,
	//
	//      -- Permissions from roles
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', p.id,
	//                  'name', p.name,
	//                  'slug', p.slug,
	//                  'description', p.description
	//              )
	//          )
	//          FROM keys_roles kr
	//          JOIN roles_permissions rp ON kr.role_id = rp.role_id
	//          JOIN permissions p ON rp.permission_id = p.id
	//          WHERE kr.key_id = k.id),
	//          JSON_ARRAY()
	//      ) as role_permissions,
	//
	//      -- Rate limits
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', rl.id,
	//                  'name', rl.name,
	//                  'key_id', rl.key_id,
	//                  'identity_id', rl.identity_id,
	//                  'limit', rl.`limit`,
	//                  'duration', rl.duration,
	//                  'auto_apply', rl.auto_apply = 1
	//              )
	//          )
	//          FROM ratelimits rl
	//          WHERE rl.key_id = k.id
	//              OR rl.identity_id = i.id),
	//          JSON_ARRAY()
	//      ) as ratelimits
	//
	//  FROM `keys` k
	//  JOIN apis a ON a.key_auth_id = k.key_auth_id
	//  JOIN key_auth ka ON ka.id = k.key_auth_id
	//  JOIN workspaces ws ON ws.id = k.workspace_id
	//  LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
	//  LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
	//  WHERE k.id = ?
	//      AND k.deleted_at_m IS NULL
	//      AND a.deleted_at_m IS NULL
	//      AND ka.deleted_at_m IS NULL
	//      AND ws.deleted_at_m IS NULL
	FindLiveKeyByID(ctx context.Context, db DBTX, id string) (FindLiveKeyByIDRow, error)
	//FindManyRatelimitNamespaces
	//
	//  SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m,
	//         coalesce(
	//                 (select json_arrayagg(
	//                                 json_object(
	//                                         'id', ro.id,
	//                                         'identifier', ro.identifier,
	//                                         'limit', ro.limit,
	//                                         'duration', ro.duration
	//                                 )
	//                         )
	//                  from ratelimit_overrides ro where ro.namespace_id = ns.id AND ro.deleted_at_m IS NULL),
	//                 json_array()
	//         ) as overrides
	//  FROM `ratelimit_namespaces` ns
	//  WHERE ns.workspace_id = ?
	//    AND (ns.id IN (/*SLICE:namespaces*/?) OR ns.name IN (/*SLICE:namespaces*/?))
	FindManyRatelimitNamespaces(ctx context.Context, db DBTX, arg FindManyRatelimitNamespacesParams) ([]FindManyRatelimitNamespacesRow, error)
	//FindManyRolesByIdOrNameWithPerms
	//
	//  SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              json_object(
	//                  'id', permission.id,
	//                  'name', permission.name,
	//                  'slug', permission.slug,
	//                  'description', permission.description
	//             )
	//          )
	//           FROM (SELECT name, id, slug, description
	//                 FROM roles_permissions rp
	//                          JOIN permissions p ON p.id = rp.permission_id
	//                 WHERE rp.role_id = r.id) as permission),
	//          JSON_ARRAY()
	//  ) as permissions
	//  FROM roles r
	//  WHERE r.workspace_id = ? AND (
	//      r.id IN (/*SLICE:search*/?)
	//      OR r.name IN (/*SLICE:search*/?)
	//  )
	FindManyRolesByIdOrNameWithPerms(ctx context.Context, db DBTX, arg FindManyRolesByIdOrNameWithPermsParams) ([]FindManyRolesByIdOrNameWithPermsRow, error)
	//FindManyRolesByNamesWithPerms
	//
	//  SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              json_object(
	//                  'id', permission.id,
	//                  'name', permission.name,
	//                  'slug', permission.slug,
	//                  'description', permission.description
	//             )
	//          )
	//           FROM (SELECT name, id, slug, description
	//                 FROM roles_permissions rp
	//                          JOIN permissions p ON p.id = rp.permission_id
	//                 WHERE rp.role_id = r.id) as permission),
	//          JSON_ARRAY()
	//  ) as permissions
	//  FROM roles r
	//  WHERE r.workspace_id = ? AND r.name IN (/*SLICE:names*/?)
	FindManyRolesByNamesWithPerms(ctx context.Context, db DBTX, arg FindManyRolesByNamesWithPermsParams) ([]FindManyRolesByNamesWithPermsRow, error)
	// Finds a permission record by its ID
	// Returns: The permission record if found
	//
	//  SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
	//  FROM permissions
	//  WHERE id = ?
	//  LIMIT 1
	FindPermissionByID(ctx context.Context, db DBTX, permissionID string) (Permission, error)
	//FindPermissionByIdOrSlug
	//
	//  SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
	//  FROM permissions
	//  WHERE workspace_id = ? AND (id = ? OR slug = ?)
	FindPermissionByIdOrSlug(ctx context.Context, db DBTX, arg FindPermissionByIdOrSlugParams) (Permission, error)
	//FindPermissionByNameAndWorkspaceID
	//
	//  SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
	//  FROM permissions
	//  WHERE name = ?
	//  AND workspace_id = ?
	//  LIMIT 1
	FindPermissionByNameAndWorkspaceID(ctx context.Context, db DBTX, arg FindPermissionByNameAndWorkspaceIDParams) (Permission, error)
	//FindPermissionBySlugAndWorkspaceID
	//
	//  SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
	//  FROM permissions
	//  WHERE slug = ?
	//  AND workspace_id = ?
	//  LIMIT 1
	FindPermissionBySlugAndWorkspaceID(ctx context.Context, db DBTX, arg FindPermissionBySlugAndWorkspaceIDParams) (Permission, error)
	//FindPermissionsBySlugs
	//
	//  SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m FROM permissions WHERE workspace_id = ? AND slug IN (/*SLICE:slugs*/?)
	FindPermissionsBySlugs(ctx context.Context, db DBTX, arg FindPermissionsBySlugsParams) ([]Permission, error)
	//FindProjectById
	//
	//  SELECT
	//      id,
	//      workspace_id,
	//      name,
	//      slug,
	//      git_repository_url,
	//      default_branch,
	//      delete_protection,
	//      live_deployment_id,
	//      is_rolled_back,
	//      created_at,
	//      updated_at,
	//      depot_project_id,
	//      command
	//  FROM projects
	//  WHERE id = ?
	FindProjectById(ctx context.Context, db DBTX, id string) (FindProjectByIdRow, error)
	//FindProjectByWorkspaceSlug
	//
	//  SELECT
	//      id,
	//      workspace_id,
	//      name,
	//      slug,
	//      git_repository_url,
	//      default_branch,
	//      delete_protection,
	//      created_at,
	//      updated_at
	//  FROM projects
	//  WHERE workspace_id = ? AND slug = ?
	//  LIMIT 1
	FindProjectByWorkspaceSlug(ctx context.Context, db DBTX, arg FindProjectByWorkspaceSlugParams) (FindProjectByWorkspaceSlugRow, error)
	//FindQuotaByWorkspaceID
	//
	//  SELECT pk, workspace_id, requests_per_month, logs_retention_days, audit_logs_retention_days, team
	//  FROM `quota`
	//  WHERE workspace_id = ?
	FindQuotaByWorkspaceID(ctx context.Context, db DBTX, workspaceID string) (Quotum, error)
	//FindRatelimitNamespace
	//
	//  SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m,
	//         coalesce(
	//                 (select json_arrayagg(
	//                                 json_object(
	//                                         'id', ro.id,
	//                                         'identifier', ro.identifier,
	//                                         'limit', ro.limit,
	//                                         'duration', ro.duration
	//                                 )
	//                         )
	//                  from ratelimit_overrides ro where ro.namespace_id = ns.id AND ro.deleted_at_m IS NULL),
	//                 json_array()
	//         ) as overrides
	//  FROM `ratelimit_namespaces` ns
	//  WHERE ns.workspace_id = ?
	//  AND (ns.id = ? OR ns.name = ?)
	FindRatelimitNamespace(ctx context.Context, db DBTX, arg FindRatelimitNamespaceParams) (FindRatelimitNamespaceRow, error)
	//FindRatelimitNamespaceByID
	//
	//  SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m FROM `ratelimit_namespaces`
	//  WHERE id = ?
	FindRatelimitNamespaceByID(ctx context.Context, db DBTX, id string) (RatelimitNamespace, error)
	//FindRatelimitNamespaceByName
	//
	//  SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m FROM `ratelimit_namespaces`
	//  WHERE name = ?
	//  AND workspace_id = ?
	FindRatelimitNamespaceByName(ctx context.Context, db DBTX, arg FindRatelimitNamespaceByNameParams) (RatelimitNamespace, error)
	//FindRatelimitOverrideByID
	//
	//  SELECT pk, id, workspace_id, namespace_id, identifier, `limit`, duration, async, sharding, created_at_m, updated_at_m, deleted_at_m FROM ratelimit_overrides
	//  WHERE
	//      workspace_id = ?
	//      AND id = ?
	FindRatelimitOverrideByID(ctx context.Context, db DBTX, arg FindRatelimitOverrideByIDParams) (RatelimitOverride, error)
	//FindRatelimitOverrideByIdentifier
	//
	//  SELECT pk, id, workspace_id, namespace_id, identifier, `limit`, duration, async, sharding, created_at_m, updated_at_m, deleted_at_m FROM ratelimit_overrides
	//  WHERE
	//      workspace_id = ?
	//      AND namespace_id = ?
	//      AND identifier = ?
	FindRatelimitOverrideByIdentifier(ctx context.Context, db DBTX, arg FindRatelimitOverrideByIdentifierParams) (RatelimitOverride, error)
	// Finds a role record by its ID
	// Returns: The role record if found
	//
	//  SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m
	//  FROM roles
	//  WHERE id = ?
	//  LIMIT 1
	FindRoleByID(ctx context.Context, db DBTX, roleID string) (Role, error)
	//FindRoleByIdOrNameWithPerms
	//
	//  SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              json_object(
	//                  'id', permission.id,
	//                  'name', permission.name,
	//                  'slug', permission.slug,
	//                  'description', permission.description
	//             )
	//          )
	//           FROM (SELECT name, id, slug, description
	//                 FROM roles_permissions rp
	//                          JOIN permissions p ON p.id = rp.permission_id
	//                 WHERE rp.role_id = r.id) as permission),
	//          JSON_ARRAY()
	//  ) as permissions
	//  FROM roles r
	//  WHERE r.workspace_id = ? AND (
	//      r.id = ?
	//      OR r.name = ?
	//  )
	FindRoleByIdOrNameWithPerms(ctx context.Context, db DBTX, arg FindRoleByIdOrNameWithPermsParams) (FindRoleByIdOrNameWithPermsRow, error)
	// Finds a role record by its name within a specific workspace
	// Returns: The role record if found
	//
	//  SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m
	//  FROM roles
	//  WHERE name = ?
	//  AND workspace_id = ?
	//  LIMIT 1
	FindRoleByNameAndWorkspaceID(ctx context.Context, db DBTX, arg FindRoleByNameAndWorkspaceIDParams) (Role, error)
	//FindRolePermissionByRoleAndPermissionID
	//
	//  SELECT pk, role_id, permission_id, workspace_id, created_at_m, updated_at_m
	//  FROM roles_permissions
	//  WHERE role_id = ?
	//    AND permission_id = ?
	FindRolePermissionByRoleAndPermissionID(ctx context.Context, db DBTX, arg FindRolePermissionByRoleAndPermissionIDParams) ([]RolesPermission, error)
	//FindRolesByNames
	//
	//  SELECT id, name FROM roles WHERE workspace_id = ? AND name IN (/*SLICE:names*/?)
	FindRolesByNames(ctx context.Context, db DBTX, arg FindRolesByNamesParams) ([]FindRolesByNamesRow, error)
	//FindSentinelByID
	//
	//  SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at FROM sentinels s
	//  WHERE id = ? LIMIT 1
	FindSentinelByID(ctx context.Context, db DBTX, id string) (Sentinel, error)
	//FindSentinelsByEnvironmentID
	//
	//  SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at FROM sentinels WHERE environment_id = ?
	FindSentinelsByEnvironmentID(ctx context.Context, db DBTX, environmentID string) ([]Sentinel, error)
	//FindWorkspaceByID
	//
	//  SELECT pk, id, org_id, name, slug, k8s_namespace, partition_id, plan, tier, stripe_customer_id, stripe_subscription_id, beta_features, features, subscriptions, enabled, delete_protection, created_at_m, updated_at_m, deleted_at_m FROM `workspaces`
	//  WHERE id = ?
	FindWorkspaceByID(ctx context.Context, db DBTX, id string) (Workspace, error)
	//GetKeyAuthByID
	//
	//  SELECT
	//      id,
	//      workspace_id,
	//      created_at_m,
	//      default_prefix,
	//      default_bytes,
	//      store_encrypted_keys
	//  FROM key_auth
	//  WHERE id = ?
	//    AND deleted_at_m IS NULL
	GetKeyAuthByID(ctx context.Context, db DBTX, id string) (GetKeyAuthByIDRow, error)
	//GetWorkspacesForQuotaCheckByIDs
	//
	//  SELECT
	//     w.id,
	//     w.org_id,
	//     w.name,
	//     w.stripe_customer_id,
	//     w.tier,
	//     w.enabled,
	//     q.requests_per_month
	//  FROM `workspaces` w
	//  LEFT JOIN quota q ON w.id = q.workspace_id
	//  WHERE w.id IN (/*SLICE:workspace_ids*/?)
	GetWorkspacesForQuotaCheckByIDs(ctx context.Context, db DBTX, workspaceIds []string) ([]GetWorkspacesForQuotaCheckByIDsRow, error)
	//HardDeleteWorkspace
	//
	//  DELETE FROM `workspaces`
	//  WHERE id = ?
	//  AND delete_protection = false
	HardDeleteWorkspace(ctx context.Context, db DBTX, id string) (sql.Result, error)
	//InsertAcmeChallenge
	//
	//  INSERT INTO acme_challenges (
	//      workspace_id,
	//      domain_id,
	//      token,
	//      authorization,
	//      status,
	//      challenge_type,
	//      created_at,
	//      updated_at,
	//      expires_at
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertAcmeChallenge(ctx context.Context, db DBTX, arg InsertAcmeChallengeParams) error
	//InsertAcmeUser
	//
	//
	//  INSERT INTO acme_users (id, workspace_id, encrypted_key, created_at)
	//  VALUES (?,?,?,?)
	InsertAcmeUser(ctx context.Context, db DBTX, arg InsertAcmeUserParams) error
	//InsertApi
	//
	//  INSERT INTO apis (
	//      id,
	//      name,
	//      workspace_id,
	//      auth_type,
	//      ip_whitelist,
	//      key_auth_id,
	//      created_at_m,
	//      deleted_at_m
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      NULL
	//  )
	InsertApi(ctx context.Context, db DBTX, arg InsertApiParams) error
	//InsertAuditLog
	//
	//  INSERT INTO `audit_log` (
	//      id,
	//      workspace_id,
	//      bucket_id,
	//      bucket,
	//      event,
	//      time,
	//      display,
	//      remote_ip,
	//      user_agent,
	//      actor_type,
	//      actor_id,
	//      actor_name,
	//      actor_meta,
	//      created_at
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      CAST(? AS JSON),
	//      ?
	//  )
	InsertAuditLog(ctx context.Context, db DBTX, arg InsertAuditLogParams) error
	//InsertAuditLogTarget
	//
	//  INSERT INTO `audit_log_target` (
	//      workspace_id,
	//      bucket_id,
	//      bucket,
	//      audit_log_id,
	//      display_name,
	//      type,
	//      id,
	//      name,
	//      meta,
	//      created_at
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      CAST(? AS JSON),
	//      ?
	//  )
	InsertAuditLogTarget(ctx context.Context, db DBTX, arg InsertAuditLogTargetParams) error
	//InsertCertificate
	//
	//  INSERT INTO certificates (id, workspace_id, hostname, certificate, encrypted_private_key, created_at)
	//  VALUES (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE
	//  workspace_id = VALUES(workspace_id),
	//  hostname = VALUES(hostname),
	//  certificate = VALUES(certificate),
	//  encrypted_private_key = VALUES(encrypted_private_key),
	//  updated_at = ?
	InsertCertificate(ctx context.Context, db DBTX, arg InsertCertificateParams) error
	//InsertCiliumNetworkPolicy
	//
	//  INSERT INTO cilium_network_policies (
	//      id,
	//      workspace_id,
	//      project_id,
	//      environment_id,
	//      k8s_name,
	//      region,
	//      policy,
	//      version,
	//      created_at
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertCiliumNetworkPolicy(ctx context.Context, db DBTX, arg InsertCiliumNetworkPolicyParams) error
	//InsertClickhouseWorkspaceSettings
	//
	//  INSERT INTO `clickhouse_workspace_settings` (
	//      workspace_id,
	//      username,
	//      password_encrypted,
	//      quota_duration_seconds,
	//      max_queries_per_window,
	//      max_execution_time_per_window,
	//      max_query_execution_time,
	//      max_query_memory_bytes,
	//      max_query_result_rows,
	//      created_at,
	//      updated_at
	//  )
	//  VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertClickhouseWorkspaceSettings(ctx context.Context, db DBTX, arg InsertClickhouseWorkspaceSettingsParams) error
	//InsertCustomDomain
	//
	//  INSERT INTO custom_domains (
	//      id, workspace_id, project_id, environment_id, domain,
	//      challenge_type, verification_status, verification_token, target_cname, invocation_id, created_at
	//  ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	InsertCustomDomain(ctx context.Context, db DBTX, arg InsertCustomDomainParams) error
	//InsertDeployment
	//
	//  INSERT INTO `deployments` (
	//      id,
	//      k8s_name,
	//      workspace_id,
	//      project_id,
	//      environment_id,
	//      git_commit_sha,
	//      git_branch,
	//      sentinel_config,
	//      git_commit_message,
	//      git_commit_author_handle,
	//      git_commit_author_avatar_url,
	//      git_commit_timestamp,
	//      openapi_spec,
	//      encrypted_environment_variables,
	//      command,
	//      status,
	//      cpu_millicores,
	//      memory_mib,
	//      created_at,
	//      updated_at
	//  )
	//  VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertDeployment(ctx context.Context, db DBTX, arg InsertDeploymentParams) error
	//InsertDeploymentTopology
	//
	//  INSERT INTO `deployment_topology` (
	//      workspace_id,
	//      deployment_id,
	//      region,
	//      desired_replicas,
	//      desired_status,
	//      version,
	//      created_at
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertDeploymentTopology(ctx context.Context, db DBTX, arg InsertDeploymentTopologyParams) error
	//InsertEnvironment
	//
	//  INSERT INTO environments (
	//      id,
	//      workspace_id,
	//      project_id,
	//      slug,
	//      description,
	//      created_at,
	//      updated_at,
	//      sentinel_config
	//  ) VALUES (
	//      ?, ?, ?, ?, ?, ?, ?, ?
	//  )
	InsertEnvironment(ctx context.Context, db DBTX, arg InsertEnvironmentParams) error
	//InsertFrontlineRoute
	//
	//  INSERT INTO frontline_routes (
	//      id,
	//      project_id,
	//      deployment_id,
	//      environment_id,
	//      fully_qualified_domain_name,
	//      sticky,
	//      created_at,
	//      updated_at
	//  )
	//  VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertFrontlineRoute(ctx context.Context, db DBTX, arg InsertFrontlineRouteParams) error
	//InsertGithubRepoConnection
	//
	//  INSERT INTO github_repo_connections (
	//      project_id,
	//      installation_id,
	//      repository_id,
	//      repository_full_name,
	//      created_at,
	//      updated_at
	//  )
	//  VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertGithubRepoConnection(ctx context.Context, db DBTX, arg InsertGithubRepoConnectionParams) error
	//InsertIdentity
	//
	//  INSERT INTO `identities` (
	//      id,
	//      external_id,
	//      workspace_id,
	//      environment,
	//      created_at,
	//      meta
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      CAST(? AS JSON)
	//  )
	InsertIdentity(ctx context.Context, db DBTX, arg InsertIdentityParams) error
	//InsertIdentityRatelimit
	//
	//  INSERT INTO `ratelimits` (
	//      id,
	//      workspace_id,
	//      identity_id,
	//      name,
	//      `limit`,
	//      duration,
	//      created_at,
	//      auto_apply
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  ) ON DUPLICATE KEY UPDATE
	//      name = VALUES(name),
	//      `limit` = VALUES(`limit`),
	//      duration = VALUES(duration),
	//      auto_apply = VALUES(auto_apply),
	//      updated_at = VALUES(created_at)
	InsertIdentityRatelimit(ctx context.Context, db DBTX, arg InsertIdentityRatelimitParams) error
	//InsertKey
	//
	//  INSERT INTO `keys` (
	//      id,
	//      key_auth_id,
	//      hash,
	//      start,
	//      workspace_id,
	//      for_workspace_id,
	//      name,
	//      owner_id,
	//      identity_id,
	//      meta,
	//      expires,
	//      created_at_m,
	//      enabled,
	//      remaining_requests,
	//      refill_day,
	//      refill_amount,
	//      pending_migration_id
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      null,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertKey(ctx context.Context, db DBTX, arg InsertKeyParams) error
	//InsertKeyAuth
	//
	//  INSERT INTO key_auth (
	//      id,
	//      workspace_id,
	//      created_at_m,
	//      default_prefix,
	//      default_bytes,
	//      store_encrypted_keys
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      false
	//  )
	InsertKeyAuth(ctx context.Context, db DBTX, arg InsertKeyAuthParams) error
	//InsertKeyEncryption
	//
	//  INSERT INTO encrypted_keys
	//  (workspace_id, key_id, encrypted, encryption_key_id, created_at)
	//  VALUES (?, ?, ?, ?, ?)
	InsertKeyEncryption(ctx context.Context, db DBTX, arg InsertKeyEncryptionParams) error
	//InsertKeyMigration
	//
	//  INSERT INTO key_migrations (
	//      id,
	//      workspace_id,
	//      algorithm
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertKeyMigration(ctx context.Context, db DBTX, arg InsertKeyMigrationParams) error
	//InsertKeyPermission
	//
	//  INSERT INTO `keys_permissions` (
	//      key_id,
	//      permission_id,
	//      workspace_id,
	//      created_at_m
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  ) ON DUPLICATE KEY UPDATE updated_at_m = ?
	InsertKeyPermission(ctx context.Context, db DBTX, arg InsertKeyPermissionParams) error
	//InsertKeyRatelimit
	//
	//  INSERT INTO `ratelimits` (
	//      id,
	//      workspace_id,
	//      key_id,
	//      name,
	//      `limit`,
	//      duration,
	//      auto_apply,
	//      created_at
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  ) ON DUPLICATE KEY UPDATE
	//  `limit` = VALUES(`limit`),
	//  duration = VALUES(duration),
	//  auto_apply = VALUES(auto_apply),
	//  updated_at = ?
	InsertKeyRatelimit(ctx context.Context, db DBTX, arg InsertKeyRatelimitParams) error
	//InsertKeyRole
	//
	//  INSERT INTO keys_roles (
	//    key_id,
	//    role_id,
	//    workspace_id,
	//    created_at_m
	//  )
	//  VALUES (
	//    ?,
	//    ?,
	//    ?,
	//    ?
	//  )
	InsertKeyRole(ctx context.Context, db DBTX, arg InsertKeyRoleParams) error
	//InsertKeySpace
	//
	//  INSERT INTO `key_auth` (
	//      id,
	//      workspace_id,
	//      created_at_m,
	//      store_encrypted_keys,
	//      default_prefix,
	//      default_bytes,
	//      size_approx,
	//      size_last_updated_at
	//  ) VALUES (
	//      ?,
	//      ?,
	//        ?,
	//      ?,
	//      ?,
	//      ?,
	//      0,
	//      0
	//  )
	InsertKeySpace(ctx context.Context, db DBTX, arg InsertKeySpaceParams) error
	//InsertPermission
	//
	//  INSERT INTO permissions (
	//    id,
	//    workspace_id,
	//    name,
	//    slug,
	//    description,
	//    created_at_m
	//  )
	//  VALUES (
	//    ?,
	//    ?,
	//    ?,
	//    ?,
	//    ?,
	//    ?
	//  )
	InsertPermission(ctx context.Context, db DBTX, arg InsertPermissionParams) error
	//InsertProject
	//
	//  INSERT INTO projects (
	//      id,
	//      workspace_id,
	//      name,
	//      slug,
	//      git_repository_url,
	//      default_branch,
	//      delete_protection,
	//      created_at,
	//      updated_at
	//  ) VALUES (
	//      ?, ?, ?, ?, ?, ?, ?, ?, ?
	//  )
	InsertProject(ctx context.Context, db DBTX, arg InsertProjectParams) error
	//InsertRatelimitNamespace
	//
	//  INSERT INTO
	//      `ratelimit_namespaces` (
	//          id,
	//          workspace_id,
	//          name,
	//          created_at_m,
	//          updated_at_m,
	//          deleted_at_m
	//          )
	//  VALUES
	//      (
	//          ?,
	//          ?,
	//          ?,
	//           ?,
	//          NULL,
	//          NULL
	//      )
	InsertRatelimitNamespace(ctx context.Context, db DBTX, arg InsertRatelimitNamespaceParams) error
	//InsertRatelimitOverride
	//
	//  INSERT INTO ratelimit_overrides (
	//      id,
	//      workspace_id,
	//      namespace_id,
	//      identifier,
	//      `limit`,
	//      duration,
	//      async,
	//      created_at_m
	//  )
	//  VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      false,
	//      ?
	//  )
	//  ON DUPLICATE KEY UPDATE
	//      `limit` = VALUES(`limit`),
	//      duration = VALUES(duration),
	//      async = VALUES(async),
	//      updated_at_m = ?
	InsertRatelimitOverride(ctx context.Context, db DBTX, arg InsertRatelimitOverrideParams) error
	//InsertRole
	//
	//  INSERT INTO roles (
	//    id,
	//    workspace_id,
	//    name,
	//    description,
	//    created_at_m
	//  )
	//  VALUES (
	//    ?,
	//    ?,
	//    ?,
	//    ?,
	//    ?
	//  )
	InsertRole(ctx context.Context, db DBTX, arg InsertRoleParams) error
	//InsertRolePermission
	//
	//  INSERT INTO roles_permissions (
	//    role_id,
	//    permission_id,
	//    workspace_id,
	//    created_at_m
	//  )
	//  VALUES (
	//    ?,
	//    ?,
	//    ?,
	//    ?
	//  )
	InsertRolePermission(ctx context.Context, db DBTX, arg InsertRolePermissionParams) error
	//InsertSentinel
	//
	//  INSERT INTO sentinels (
	//      id,
	//      workspace_id,
	//      environment_id,
	//      project_id,
	//      k8s_address,
	//      k8s_name,
	//      region,
	//      image,
	//      health,
	//      desired_replicas,
	//      available_replicas,
	//      cpu_millicores,
	//      memory_mib,
	//      version,
	//      created_at
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?
	//  )
	InsertSentinel(ctx context.Context, db DBTX, arg InsertSentinelParams) error
	//InsertWorkspace
	//
	//  INSERT INTO `workspaces` (
	//      id,
	//      org_id,
	//      name,
	//      slug,
	//      created_at_m,
	//      tier,
	//      beta_features,
	//      features,
	//      enabled,
	//      delete_protection
	//  )
	//  VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      'Free',
	//      '{}',
	//      '{}',
	//      true,
	//      true
	//  )
	InsertWorkspace(ctx context.Context, db DBTX, arg InsertWorkspaceParams) error
	// ListCiliumNetworkPoliciesByRegion returns cilium network policies for a region with version > after_version.
	// Used by WatchCiliumNetworkPolicies to stream policy state changes to krane agents.
	//
	//  SELECT
	//      n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
	//      w.k8s_namespace
	//  FROM `cilium_network_policies` n
	//  JOIN `workspaces` w ON w.id = n.workspace_id
	//  WHERE n.region = ? AND n.version > ?
	//  ORDER BY n.version ASC
	//  LIMIT ?
	ListCiliumNetworkPoliciesByRegion(ctx context.Context, db DBTX, arg ListCiliumNetworkPoliciesByRegionParams) ([]ListCiliumNetworkPoliciesByRegionRow, error)
	//ListCustomDomainsByProjectID
	//
	//  SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at
	//  FROM custom_domains
	//  WHERE project_id = ?
	//  ORDER BY created_at DESC
	ListCustomDomainsByProjectID(ctx context.Context, db DBTX, projectID string) ([]CustomDomain, error)
	// ListDeploymentTopologyByRegion returns deployment topologies for a region with version > after_version.
	// Used by WatchDeployments to stream deployment state changes to krane agents.
	//
	//  SELECT
	//      dt.pk, dt.workspace_id, dt.deployment_id, dt.region, dt.desired_replicas, dt.version, dt.desired_status, dt.created_at, dt.updated_at,
	//      d.pk, d.id, d.k8s_name, d.workspace_id, d.project_id, d.environment_id, d.image, d.build_id, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.sentinel_config, d.openapi_spec, d.cpu_millicores, d.memory_mib, d.desired_state, d.encrypted_environment_variables, d.command, d.status, d.created_at, d.updated_at,
	//      w.k8s_namespace
	//  FROM `deployment_topology` dt
	//  INNER JOIN `deployments` d ON dt.deployment_id = d.id
	//  INNER JOIN `workspaces` w ON d.workspace_id = w.id
	//  WHERE dt.region = ? AND dt.version > ?
	//  ORDER BY dt.version ASC
	//  LIMIT ?
	ListDeploymentTopologyByRegion(ctx context.Context, db DBTX, arg ListDeploymentTopologyByRegionParams) ([]ListDeploymentTopologyByRegionRow, error)
	// ListDesiredDeploymentTopology returns all deployment topologies matching the desired state for a region.
	// Used during bootstrap to stream all running deployments to krane.
	//
	//  SELECT
	//      dt.pk, dt.workspace_id, dt.deployment_id, dt.region, dt.desired_replicas, dt.version, dt.desired_status, dt.created_at, dt.updated_at,
	//      d.pk, d.id, d.k8s_name, d.workspace_id, d.project_id, d.environment_id, d.image, d.build_id, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.sentinel_config, d.openapi_spec, d.cpu_millicores, d.memory_mib, d.desired_state, d.encrypted_environment_variables, d.command, d.status, d.created_at, d.updated_at,
	//      w.k8s_namespace
	//  FROM `deployment_topology` dt
	//  INNER JOIN `deployments` d ON dt.deployment_id = d.id
	//  INNER JOIN `workspaces` w ON d.workspace_id = w.id
	//  WHERE (? = '' OR dt.region = ?)
	//      AND d.desired_state = ?
	//      AND dt.deployment_id > ?
	//  ORDER BY dt.deployment_id ASC
	//  LIMIT ?
	ListDesiredDeploymentTopology(ctx context.Context, db DBTX, arg ListDesiredDeploymentTopologyParams) ([]ListDesiredDeploymentTopologyRow, error)
	//ListDesiredNetworkPolicies
	//
	//  SELECT
	//      n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
	//      w.k8s_namespace
	//  FROM `cilium_network_policies` n
	//  INNER JOIN `workspaces` w ON n.workspace_id = w.id
	//  WHERE (? = '' OR n.region = ?) AND n.id > ?
	//  ORDER BY n.id ASC
	//  LIMIT ?
	ListDesiredNetworkPolicies(ctx context.Context, db DBTX, arg ListDesiredNetworkPoliciesParams) ([]ListDesiredNetworkPoliciesRow, error)
	// ListDesiredSentinels returns all sentinels matching the desired state for a region.
	// Used during bootstrap to stream all running sentinels to krane.
	//
	//  SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at
	//  FROM `sentinels`
	//  WHERE (? = '' OR region = ?)
	//      AND desired_state = ?
	//      AND id > ?
	//  ORDER BY id ASC
	//  LIMIT ?
	ListDesiredSentinels(ctx context.Context, db DBTX, arg ListDesiredSentinelsParams) ([]Sentinel, error)
	//ListDirectPermissionsByKeyID
	//
	//  SELECT p.pk, p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
	//  FROM keys_permissions kp
	//  JOIN permissions p ON kp.permission_id = p.id
	//  WHERE kp.key_id = ?
	//  ORDER BY p.slug
	ListDirectPermissionsByKeyID(ctx context.Context, db DBTX, keyID string) ([]Permission, error)
	//ListExecutableChallenges
	//
	//  SELECT dc.workspace_id, dc.challenge_type, d.domain FROM acme_challenges dc
	//  JOIN custom_domains d ON dc.domain_id = d.id
	//  WHERE (dc.status = 'waiting' OR (dc.status = 'verified' AND dc.expires_at <= UNIX_TIMESTAMP(DATE_ADD(NOW(), INTERVAL 30 DAY)) * 1000))
	//  AND dc.challenge_type IN (/*SLICE:verification_types*/?)
	//  ORDER BY d.created_at ASC
	ListExecutableChallenges(ctx context.Context, db DBTX, verificationTypes []AcmeChallengesChallengeType) ([]ListExecutableChallengesRow, error)
	//ListIdentities
	//
	//  SELECT
	//      i.id,
	//      i.external_id,
	//      i.workspace_id,
	//      i.environment,
	//      i.meta,
	//      i.deleted,
	//      i.created_at,
	//      i.updated_at,
	//      COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              JSON_OBJECT(
	//                  'id', r.id,
	//                  'name', r.name,
	//                  'limit', r.`limit`,
	//                  'duration', r.duration,
	//                  'auto_apply', r.auto_apply = 1
	//              )
	//          )
	//          FROM ratelimits r
	//          WHERE r.identity_id = i.id),
	//          JSON_ARRAY()
	//      ) as ratelimits
	//  FROM identities i
	//  WHERE i.workspace_id = ?
	//  AND i.deleted = ?
	//  AND i.id >= ?
	//  ORDER BY i.id ASC
	//  LIMIT ?
	ListIdentities(ctx context.Context, db DBTX, arg ListIdentitiesParams) ([]ListIdentitiesRow, error)
	//ListIdentityRatelimits
	//
	//  SELECT pk, id, name, workspace_id, created_at, updated_at, key_id, identity_id, `limit`, duration, auto_apply
	//  FROM ratelimits
	//  WHERE identity_id = ?
	//  ORDER BY id ASC
	ListIdentityRatelimits(ctx context.Context, db DBTX, identityID sql.NullString) ([]Ratelimit, error)
	//ListIdentityRatelimitsByID
	//
	//  SELECT pk, id, name, workspace_id, created_at, updated_at, key_id, identity_id, `limit`, duration, auto_apply FROM ratelimits WHERE identity_id = ?
	ListIdentityRatelimitsByID(ctx context.Context, db DBTX, identityID sql.NullString) ([]Ratelimit, error)
	//ListIdentityRatelimitsByIDs
	//
	//  SELECT pk, id, name, workspace_id, created_at, updated_at, key_id, identity_id, `limit`, duration, auto_apply FROM ratelimits WHERE identity_id IN (/*SLICE:ids*/?)
	ListIdentityRatelimitsByIDs(ctx context.Context, db DBTX, ids []sql.NullString) ([]Ratelimit, error)
	//ListKeysByKeySpaceID
	//
	//  SELECT
	//    k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
	//    i.id as identity_id,
	//    i.external_id as external_id,
	//    i.meta as identity_meta,
	//    ek.encrypted as encrypted_key,
	//    ek.encryption_key_id as encryption_key_id
	//
	//  FROM `keys` k
	//  LEFT JOIN `identities` i ON k.identity_id = i.id
	//  LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
	//  WHERE k.key_auth_id = ?
	//  AND k.id >= ?
	//  AND (? IS NULL OR k.identity_id = ?)
	//  AND k.deleted_at_m IS NULL
	//  ORDER BY k.id ASC
	//  LIMIT ?
	ListKeysByKeySpaceID(ctx context.Context, db DBTX, arg ListKeysByKeySpaceIDParams) ([]ListKeysByKeySpaceIDRow, error)
	//ListLiveKeysByKeySpaceID
	//
	//  SELECT k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
	//         i.id                 as identity_table_id,
	//         i.external_id        as identity_external_id,
	//         i.meta               as identity_meta,
	//         ek.encrypted         as encrypted_key,
	//         ek.encryption_key_id as encryption_key_id,
	//         -- Roles with both IDs and names (sorted by name)
	//         COALESCE(
	//                 (SELECT JSON_ARRAYAGG(
	//                                 JSON_OBJECT(
	//                                         'id', r.id,
	//                                         'name', r.name,
	//                                         'description', r.description
	//                                 )
	//                         )
	//                  FROM keys_roles kr
	//                           JOIN roles r ON r.id = kr.role_id
	//                  WHERE kr.key_id = k.id
	//                  ORDER BY r.name),
	//                 JSON_ARRAY()
	//         )                    as roles,
	//         -- Direct permissions attached to the key (sorted by slug)
	//         COALESCE(
	//                 (SELECT JSON_ARRAYAGG(
	//                                 JSON_OBJECT(
	//                                         'id', p.id,
	//                                         'name', p.name,
	//                                         'slug', p.slug,
	//                                         'description', p.description
	//                                 )
	//                         )
	//                  FROM keys_permissions kp
	//                           JOIN permissions p ON kp.permission_id = p.id
	//                  WHERE kp.key_id = k.id
	//                  ORDER BY p.slug),
	//                 JSON_ARRAY()
	//         )                    as permissions,
	//         -- Permissions from roles (sorted by slug)
	//         COALESCE(
	//                 (SELECT JSON_ARRAYAGG(
	//                                 JSON_OBJECT(
	//                                         'id', p.id,
	//                                         'name', p.name,
	//                                         'slug', p.slug,
	//                                         'description', p.description
	//                                 )
	//                         )
	//                  FROM keys_roles kr
	//                           JOIN roles_permissions rp ON kr.role_id = rp.role_id
	//                           JOIN permissions p ON rp.permission_id = p.id
	//                  WHERE kr.key_id = k.id
	//                  ORDER BY p.slug),
	//                 JSON_ARRAY()
	//         )                    as role_permissions,
	//         -- Rate limits
	//         COALESCE(
	//                 (SELECT JSON_ARRAYAGG(
	//                                 JSON_OBJECT(
	//                                         'id', id,
	//                                         'name', name,
	//                                         'key_id', key_id,
	//                                         'identity_id', identity_id,
	//                                         'limit', `limit`,
	//                                         'duration', duration,
	//                                         'auto_apply', auto_apply = 1
	//                                 )
	//                         )
	//                  FROM (
	//                      SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.`limit`, rl.duration, rl.auto_apply
	//                      FROM ratelimits rl
	//                      WHERE rl.key_id = k.id
	//                      UNION ALL
	//                      SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.`limit`, rl.duration, rl.auto_apply
	//                      FROM ratelimits rl
	//                      WHERE rl.identity_id = i.id
	//                  ) AS combined_rl),
	//                 JSON_ARRAY()
	//         )                    AS ratelimits
	//  FROM `keys` k
	//           STRAIGHT_JOIN key_auth ka ON ka.id = k.key_auth_id
	//           LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
	//           LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
	//  WHERE k.key_auth_id = ?
	//    AND k.id >= ?
	//    AND (? IS NULL OR k.identity_id = ?)
	//    AND k.deleted_at_m IS NULL
	//    AND ka.deleted_at_m IS NULL
	//  ORDER BY k.id ASC
	//  LIMIT ?
	ListLiveKeysByKeySpaceID(ctx context.Context, db DBTX, arg ListLiveKeysByKeySpaceIDParams) ([]ListLiveKeysByKeySpaceIDRow, error)
	//ListNetworkPolicyByRegion
	//
	//  SELECT
	//      n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
	//      w.k8s_namespace
	//  FROM `cilium_network_policies` n
	//  INNER JOIN `workspaces` w ON n.workspace_id = w.id
	//  WHERE n.region = ? AND n.version > ?
	//  ORDER BY n.version ASC
	//  LIMIT ?
	ListNetworkPolicyByRegion(ctx context.Context, db DBTX, arg ListNetworkPolicyByRegionParams) ([]ListNetworkPolicyByRegionRow, error)
	//ListPermissions
	//
	//  SELECT p.pk, p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
	//  FROM permissions p
	//  WHERE p.workspace_id = ?
	//    AND p.id >= ?
	//  ORDER BY p.id
	//  LIMIT ?
	ListPermissions(ctx context.Context, db DBTX, arg ListPermissionsParams) ([]Permission, error)
	//ListPermissionsByKeyID
	//
	//  WITH direct_permissions AS (
	//      SELECT p.slug as permission_slug
	//      FROM keys_permissions kp
	//      JOIN permissions p ON kp.permission_id = p.id
	//      WHERE kp.key_id = ?
	//  ),
	//  role_permissions AS (
	//      SELECT p.slug as permission_slug
	//      FROM keys_roles kr
	//      JOIN roles_permissions rp ON kr.role_id = rp.role_id
	//      JOIN permissions p ON rp.permission_id = p.id
	//      WHERE kr.key_id = ?
	//  )
	//  SELECT DISTINCT permission_slug
	//  FROM (
	//      SELECT permission_slug FROM direct_permissions
	//      UNION ALL
	//      SELECT permission_slug FROM role_permissions
	//  ) all_permissions
	ListPermissionsByKeyID(ctx context.Context, db DBTX, arg ListPermissionsByKeyIDParams) ([]string, error)
	//ListPermissionsByRoleID
	//
	//  SELECT p.pk, p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
	//  FROM permissions p
	//  JOIN roles_permissions rp ON p.id = rp.permission_id
	//  WHERE rp.role_id = ?
	//  ORDER BY p.slug
	ListPermissionsByRoleID(ctx context.Context, db DBTX, roleID string) ([]Permission, error)
	//ListRatelimitOverridesByNamespaceID
	//
	//  SELECT pk, id, workspace_id, namespace_id, identifier, `limit`, duration, async, sharding, created_at_m, updated_at_m, deleted_at_m FROM ratelimit_overrides
	//  WHERE
	//  workspace_id = ?
	//  AND namespace_id = ?
	//  AND deleted_at_m IS NULL
	//  AND id >= ?
	//  ORDER BY id ASC
	//  LIMIT ?
	ListRatelimitOverridesByNamespaceID(ctx context.Context, db DBTX, arg ListRatelimitOverridesByNamespaceIDParams) ([]RatelimitOverride, error)
	//ListRatelimitsByKeyID
	//
	//  SELECT
	//    id,
	//    name,
	//    `limit`,
	//    duration,
	//    auto_apply
	//  FROM ratelimits
	//  WHERE key_id = ?
	ListRatelimitsByKeyID(ctx context.Context, db DBTX, keyID sql.NullString) ([]ListRatelimitsByKeyIDRow, error)
	//ListRatelimitsByKeyIDs
	//
	//  SELECT
	//    id,
	//    key_id,
	//    name,
	//    `limit`,
	//    duration,
	//    auto_apply
	//  FROM ratelimits
	//  WHERE key_id IN (/*SLICE:key_ids*/?)
	//  ORDER BY key_id, id
	ListRatelimitsByKeyIDs(ctx context.Context, db DBTX, keyIds []sql.NullString) ([]ListRatelimitsByKeyIDsRow, error)
	//ListRoles
	//
	//  SELECT r.pk, r.id, r.workspace_id, r.name, r.description, r.created_at_m, r.updated_at_m, COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              json_object(
	//                  'id', permission.id,
	//                  'name', permission.name,
	//                  'slug', permission.slug,
	//                  'description', permission.description
	//             )
	//          )
	//           FROM (SELECT name, id, slug, description
	//                 FROM roles_permissions rp
	//                          JOIN permissions p ON p.id = rp.permission_id
	//                 WHERE rp.role_id = r.id) as permission),
	//          JSON_ARRAY()
	//  ) as permissions
	//  FROM roles r
	//  WHERE r.workspace_id = ?
	//  AND r.id >= ?
	//  ORDER BY r.id
	//  LIMIT ?
	ListRoles(ctx context.Context, db DBTX, arg ListRolesParams) ([]ListRolesRow, error)
	//ListRolesByKeyID
	//
	//  SELECT r.pk, r.id, r.workspace_id, r.name, r.description, r.created_at_m, r.updated_at_m, COALESCE(
	//          (SELECT JSON_ARRAYAGG(
	//              json_object(
	//                  'id', permission.id,
	//                  'name', permission.name,
	//                  'slug', permission.slug,
	//                  'description', permission.description
	//             )
	//          )
	//           FROM (SELECT name, id, slug, description
	//                 FROM roles_permissions rp
	//                          JOIN permissions p ON p.id = rp.permission_id
	//                 WHERE rp.role_id = r.id) as permission),
	//          JSON_ARRAY()
	//  ) as permissions
	//  FROM keys_roles kr
	//  JOIN roles r ON kr.role_id = r.id
	//  WHERE kr.key_id = ?
	//  ORDER BY r.name
	ListRolesByKeyID(ctx context.Context, db DBTX, keyID string) ([]ListRolesByKeyIDRow, error)
	// ListSentinelsByRegion returns sentinels for a region with version > after_version.
	// Used by WatchSentinels to stream sentinel state changes to krane agents.
	//
	//  SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at FROM `sentinels`
	//  WHERE region = ? AND version > ?
	//  ORDER BY version ASC
	//  LIMIT ?
	ListSentinelsByRegion(ctx context.Context, db DBTX, arg ListSentinelsByRegionParams) ([]Sentinel, error)
	//ListWorkspaces
	//
	//  SELECT
	//     w.pk, w.id, w.org_id, w.name, w.slug, w.k8s_namespace, w.partition_id, w.plan, w.tier, w.stripe_customer_id, w.stripe_subscription_id, w.beta_features, w.features, w.subscriptions, w.enabled, w.delete_protection, w.created_at_m, w.updated_at_m, w.deleted_at_m,
	//     q.pk, q.workspace_id, q.requests_per_month, q.logs_retention_days, q.audit_logs_retention_days, q.team
	//  FROM `workspaces` w
	//  LEFT JOIN quota q ON w.id = q.workspace_id
	//  WHERE w.id > ?
	//  ORDER BY w.id ASC
	//  LIMIT 100
	ListWorkspaces(ctx context.Context, db DBTX, cursor string) ([]ListWorkspacesRow, error)
	//ListWorkspacesForQuotaCheck
	//
	//  SELECT
	//     w.id,
	//     w.org_id,
	//     w.name,
	//     w.stripe_customer_id,
	//     w.tier,
	//     w.enabled,
	//     q.requests_per_month
	//  FROM `workspaces` w
	//  LEFT JOIN quota q ON w.id = q.workspace_id
	//  WHERE w.id > ?
	//  ORDER BY w.id ASC
	//  LIMIT 100
	ListWorkspacesForQuotaCheck(ctx context.Context, db DBTX, cursor string) ([]ListWorkspacesForQuotaCheckRow, error)
	// Acquires an exclusive lock on the identity row to prevent concurrent modifications.
	// This should be called at the start of a transaction before modifying identity-related data.
	//
	//  SELECT id FROM identities
	//  WHERE id = ?
	//  FOR UPDATE
	LockIdentityForUpdate(ctx context.Context, db DBTX, id string) (string, error)
	// Acquires an exclusive lock on the key row to prevent concurrent modifications.
	// This is used to prevent deadlocks when updating key ratelimits concurrently.
	//
	//  SELECT id FROM `keys`
	//  WHERE id = ?
	//  FOR UPDATE
	LockKeyForUpdate(ctx context.Context, db DBTX, id string) (string, error)
	//ReassignFrontlineRoute
	//
	//  UPDATE frontline_routes
	//  SET
	//    deployment_id = ?,
	//    updated_at = ?
	//  WHERE id = ?
	ReassignFrontlineRoute(ctx context.Context, db DBTX, arg ReassignFrontlineRouteParams) error
	//ResetCustomDomainVerification
	//
	//  UPDATE custom_domains
	//  SET verification_status = ?,
	//      check_attempts = ?,
	//      verification_error = NULL,
	//      last_checked_at = NULL,
	//      invocation_id = ?,
	//      updated_at = ?
	//  WHERE domain = ?
	ResetCustomDomainVerification(ctx context.Context, db DBTX, arg ResetCustomDomainVerificationParams) error
	//SetWorkspaceK8sNamespace
	//
	//  UPDATE `workspaces`
	//  SET k8s_namespace = ?
	//  WHERE id = ? AND k8s_namespace IS NULL
	SetWorkspaceK8sNamespace(ctx context.Context, db DBTX, arg SetWorkspaceK8sNamespaceParams) error
	//SoftDeleteApi
	//
	//  UPDATE apis
	//  SET deleted_at_m = ?
	//  WHERE id = ?
	SoftDeleteApi(ctx context.Context, db DBTX, arg SoftDeleteApiParams) error
	//SoftDeleteIdentity
	//
	//  UPDATE identities
	//  SET deleted = 1
	//  WHERE id = ?
	//    AND workspace_id = ?
	SoftDeleteIdentity(ctx context.Context, db DBTX, arg SoftDeleteIdentityParams) error
	//SoftDeleteKeyByID
	//
	//  UPDATE `keys` SET deleted_at_m = ? WHERE id = ?
	SoftDeleteKeyByID(ctx context.Context, db DBTX, arg SoftDeleteKeyByIDParams) error
	//SoftDeleteManyKeysByKeySpaceID
	//
	//  UPDATE `keys`
	//  SET deleted_at_m = ?
	//  WHERE key_auth_id = ?
	//  AND deleted_at_m IS NULL
	SoftDeleteManyKeysByKeySpaceID(ctx context.Context, db DBTX, arg SoftDeleteManyKeysByKeySpaceIDParams) error
	//SoftDeleteRatelimitNamespace
	//
	//  UPDATE `ratelimit_namespaces`
	//  SET
	//      deleted_at_m =  ?
	//  WHERE id = ?
	SoftDeleteRatelimitNamespace(ctx context.Context, db DBTX, arg SoftDeleteRatelimitNamespaceParams) error
	//SoftDeleteRatelimitOverride
	//
	//  UPDATE `ratelimit_overrides`
	//  SET
	//      deleted_at_m =  ?
	//  WHERE id = ?
	SoftDeleteRatelimitOverride(ctx context.Context, db DBTX, arg SoftDeleteRatelimitOverrideParams) error
	//SoftDeleteWorkspace
	//
	//  UPDATE `workspaces`
	//  SET deleted_at_m = ?
	//  WHERE id = ?
	//  AND delete_protection = false
	SoftDeleteWorkspace(ctx context.Context, db DBTX, arg SoftDeleteWorkspaceParams) (sql.Result, error)
	//UpdateAcmeChallengePending
	//
	//  UPDATE acme_challenges
	//  SET status = ?, token = ?, authorization = ?, updated_at = ?
	//  WHERE domain_id = ?
	UpdateAcmeChallengePending(ctx context.Context, db DBTX, arg UpdateAcmeChallengePendingParams) error
	//UpdateAcmeChallengeStatus
	//
	//  UPDATE acme_challenges
	//  SET status = ?, updated_at = ?
	//  WHERE domain_id = ?
	UpdateAcmeChallengeStatus(ctx context.Context, db DBTX, arg UpdateAcmeChallengeStatusParams) error
	//UpdateAcmeChallengeTryClaiming
	//
	//  UPDATE acme_challenges
	//  SET status = ?, updated_at = ?
	//  WHERE domain_id = ? AND status = 'waiting'
	UpdateAcmeChallengeTryClaiming(ctx context.Context, db DBTX, arg UpdateAcmeChallengeTryClaimingParams) error
	//UpdateAcmeChallengeVerifiedWithExpiry
	//
	//  UPDATE acme_challenges
	//  SET status = ?, expires_at = ?, updated_at = ?
	//  WHERE domain_id = ?
	UpdateAcmeChallengeVerifiedWithExpiry(ctx context.Context, db DBTX, arg UpdateAcmeChallengeVerifiedWithExpiryParams) error
	//UpdateAcmeUserRegistrationURI
	//
	//  UPDATE acme_users SET registration_uri = ? WHERE id = ?
	UpdateAcmeUserRegistrationURI(ctx context.Context, db DBTX, arg UpdateAcmeUserRegistrationURIParams) error
	//UpdateApiDeleteProtection
	//
	//  UPDATE apis
	//  SET delete_protection = ?
	//  WHERE id = ?
	UpdateApiDeleteProtection(ctx context.Context, db DBTX, arg UpdateApiDeleteProtectionParams) error
	//UpdateClickhouseWorkspaceSettingsLimits
	//
	//  UPDATE `clickhouse_workspace_settings`
	//  SET
	//      quota_duration_seconds = ?,
	//      max_queries_per_window = ?,
	//      max_execution_time_per_window = ?,
	//      max_query_execution_time = ?,
	//      max_query_memory_bytes = ?,
	//      max_query_result_rows = ?,
	//      updated_at = ?
	//  WHERE workspace_id = ?
	UpdateClickhouseWorkspaceSettingsLimits(ctx context.Context, db DBTX, arg UpdateClickhouseWorkspaceSettingsLimitsParams) error
	//UpdateCustomDomainCheckAttempt
	//
	//  UPDATE custom_domains
	//  SET check_attempts = ?,
	//      last_checked_at = ?,
	//      updated_at = ?
	//  WHERE id = ?
	UpdateCustomDomainCheckAttempt(ctx context.Context, db DBTX, arg UpdateCustomDomainCheckAttemptParams) error
	//UpdateCustomDomainFailed
	//
	//  UPDATE custom_domains
	//  SET verification_status = ?,
	//      verification_error = ?,
	//      updated_at = ?
	//  WHERE id = ?
	UpdateCustomDomainFailed(ctx context.Context, db DBTX, arg UpdateCustomDomainFailedParams) error
	//UpdateCustomDomainInvocationID
	//
	//  UPDATE custom_domains
	//  SET invocation_id = ?,
	//      updated_at = ?
	//  WHERE id = ?
	UpdateCustomDomainInvocationID(ctx context.Context, db DBTX, arg UpdateCustomDomainInvocationIDParams) error
	//UpdateCustomDomainOwnership
	//
	//  UPDATE custom_domains
	//  SET ownership_verified = ?, cname_verified = ?, updated_at = ?
	//  WHERE id = ?
	UpdateCustomDomainOwnership(ctx context.Context, db DBTX, arg UpdateCustomDomainOwnershipParams) error
	//UpdateCustomDomainVerificationStatus
	//
	//  UPDATE custom_domains
	//  SET verification_status = ?,
	//      updated_at = ?
	//  WHERE id = ?
	UpdateCustomDomainVerificationStatus(ctx context.Context, db DBTX, arg UpdateCustomDomainVerificationStatusParams) error
	//UpdateDeploymentBuildID
	//
	//  UPDATE deployments
	//  SET build_id = ?, updated_at = ?
	//  WHERE id = ?
	UpdateDeploymentBuildID(ctx context.Context, db DBTX, arg UpdateDeploymentBuildIDParams) error
	//UpdateDeploymentImage
	//
	//  UPDATE deployments
	//  SET image = ?, updated_at = ?
	//  WHERE id = ?
	UpdateDeploymentImage(ctx context.Context, db DBTX, arg UpdateDeploymentImageParams) error
	//UpdateDeploymentOpenapiSpec
	//
	//  UPDATE deployments
	//  SET openapi_spec = ?, updated_at = ?
	//  WHERE id = ?
	UpdateDeploymentOpenapiSpec(ctx context.Context, db DBTX, arg UpdateDeploymentOpenapiSpecParams) error
	//UpdateDeploymentStatus
	//
	//  UPDATE deployments
	//  SET status = ?, updated_at = ?
	//  WHERE id = ?
	UpdateDeploymentStatus(ctx context.Context, db DBTX, arg UpdateDeploymentStatusParams) error
	//UpdateFrontlineRouteDeploymentId
	//
	//  UPDATE frontline_routes
	//  SET deployment_id = ?
	//  WHERE id = ?
	UpdateFrontlineRouteDeploymentId(ctx context.Context, db DBTX, arg UpdateFrontlineRouteDeploymentIdParams) error
	//UpdateIdentity
	//
	//  UPDATE `identities`
	//  SET
	//      meta = CAST(? AS JSON),
	//      updated_at = NOW()
	//  WHERE
	//      id = ?
	UpdateIdentity(ctx context.Context, db DBTX, arg UpdateIdentityParams) error
	//UpdateKey
	//
	//  UPDATE `keys` k SET
	//      name = CASE
	//          WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	//          ELSE k.name
	//      END,
	//      identity_id = CASE
	//          WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	//          ELSE k.identity_id
	//      END,
	//      enabled = CASE
	//          WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	//          ELSE k.enabled
	//      END,
	//      meta = CASE
	//          WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	//          ELSE k.meta
	//      END,
	//      expires = CASE
	//          WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	//          ELSE k.expires
	//      END,
	//      remaining_requests = CASE
	//          WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	//          ELSE k.remaining_requests
	//      END,
	//      refill_amount = CASE
	//          WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	//          ELSE k.refill_amount
	//      END,
	//      refill_day = CASE
	//          WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	//          ELSE k.refill_day
	//      END,
	//      updated_at_m = ?
	//  WHERE id = ?
	UpdateKey(ctx context.Context, db DBTX, arg UpdateKeyParams) error
	//UpdateKeyCreditsDecrement
	//
	//  UPDATE `keys`
	//  SET remaining_requests = CASE
	//      WHEN remaining_requests >= ? THEN remaining_requests - ?
	//      ELSE 0
	//  END
	//  WHERE id = ?
	UpdateKeyCreditsDecrement(ctx context.Context, db DBTX, arg UpdateKeyCreditsDecrementParams) error
	//UpdateKeyCreditsIncrement
	//
	//  UPDATE `keys`
	//  SET remaining_requests = remaining_requests + ?
	//  WHERE id = ?
	UpdateKeyCreditsIncrement(ctx context.Context, db DBTX, arg UpdateKeyCreditsIncrementParams) error
	//UpdateKeyCreditsRefill
	//
	//  UPDATE `keys` SET refill_amount = ?, refill_day = ? WHERE id = ?
	UpdateKeyCreditsRefill(ctx context.Context, db DBTX, arg UpdateKeyCreditsRefillParams) error
	//UpdateKeyCreditsSet
	//
	//  UPDATE `keys`
	//  SET remaining_requests = ?
	//  WHERE id = ?
	UpdateKeyCreditsSet(ctx context.Context, db DBTX, arg UpdateKeyCreditsSetParams) error
	//UpdateKeyHashAndMigration
	//
	//  UPDATE `keys`
	//  SET
	//      hash = ?,
	//      pending_migration_id = ?,
	//      start = ?,
	//      updated_at_m = ?
	//  WHERE id = ?
	UpdateKeyHashAndMigration(ctx context.Context, db DBTX, arg UpdateKeyHashAndMigrationParams) error
	//UpdateKeySpaceKeyEncryption
	//
	//  UPDATE `key_auth` SET store_encrypted_keys = ? WHERE id = ?
	UpdateKeySpaceKeyEncryption(ctx context.Context, db DBTX, arg UpdateKeySpaceKeyEncryptionParams) error
	//UpdateProjectDeployments
	//
	//  UPDATE projects
	//  SET
	//    live_deployment_id = ?,
	//    is_rolled_back = ?,
	//    updated_at = ?
	//  WHERE id = ?
	UpdateProjectDeployments(ctx context.Context, db DBTX, arg UpdateProjectDeploymentsParams) error
	//UpdateProjectDepotID
	//
	//  UPDATE projects
	//  SET
	//      depot_project_id = ?,
	//      updated_at = ?
	//  WHERE id = ?
	UpdateProjectDepotID(ctx context.Context, db DBTX, arg UpdateProjectDepotIDParams) error
	//UpdateRatelimit
	//
	//  UPDATE `ratelimits`
	//  SET
	//      name = ?,
	//      `limit` = ?,
	//      duration = ?,
	//      auto_apply = ?,
	//      updated_at = NOW()
	//  WHERE
	//      id = ?
	UpdateRatelimit(ctx context.Context, db DBTX, arg UpdateRatelimitParams) error
	//UpdateRatelimitOverride
	//
	//  UPDATE `ratelimit_overrides`
	//  SET
	//      `limit` = ?,
	//      duration = ?,
	//      async = ?,
	//      updated_at_m= ?
	//  WHERE id = ?
	UpdateRatelimitOverride(ctx context.Context, db DBTX, arg UpdateRatelimitOverrideParams) (sql.Result, error)
	//UpdateSentinelAvailableReplicasAndHealth
	//
	//  UPDATE sentinels SET
	//  available_replicas = ?,
	//  health = ?,
	//  updated_at = ?
	//  WHERE k8s_name = ?
	UpdateSentinelAvailableReplicasAndHealth(ctx context.Context, db DBTX, arg UpdateSentinelAvailableReplicasAndHealthParams) error
	//UpdateWorkspaceEnabled
	//
	//  UPDATE `workspaces`
	//  SET enabled = ?
	//  WHERE id = ?
	UpdateWorkspaceEnabled(ctx context.Context, db DBTX, arg UpdateWorkspaceEnabledParams) (sql.Result, error)
	//UpdateWorkspacePlan
	//
	//  UPDATE `workspaces`
	//  SET plan = ?
	//  WHERE id = ?
	UpdateWorkspacePlan(ctx context.Context, db DBTX, arg UpdateWorkspacePlanParams) error
	//UpsertCustomDomain
	//
	//  INSERT INTO custom_domains (
	//      id, workspace_id, project_id, environment_id, domain,
	//      challenge_type, verification_status, verification_token, target_cname, created_at
	//  )
	//  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	//  ON DUPLICATE KEY UPDATE
	//      workspace_id = VALUES(workspace_id),
	//      project_id = VALUES(project_id),
	//      environment_id = VALUES(environment_id),
	//      challenge_type = VALUES(challenge_type),
	//      verification_status = VALUES(verification_status),
	//      target_cname = VALUES(target_cname),
	//      updated_at = ?
	UpsertCustomDomain(ctx context.Context, db DBTX, arg UpsertCustomDomainParams) error
	//UpsertEnvironment
	//
	//  INSERT INTO environments (
	//      id,
	//      workspace_id,
	//      project_id,
	//      slug,
	//      sentinel_config,
	//      created_at
	//  ) VALUES (?, ?, ?, ?, ?, ?)
	//  ON DUPLICATE KEY UPDATE slug = VALUES(slug)
	UpsertEnvironment(ctx context.Context, db DBTX, arg UpsertEnvironmentParams) error
	// Inserts a new identity or does nothing if one already exists for this workspace/external_id.
	// Use FindIdentityByExternalID after this to get the actual ID.
	//
	//  INSERT INTO `identities` (
	//      id,
	//      external_id,
	//      workspace_id,
	//      environment,
	//      created_at,
	//      meta
	//  ) VALUES (
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      ?,
	//      CAST(? AS JSON)
	//  )
	//  ON DUPLICATE KEY UPDATE external_id = external_id
	UpsertIdentity(ctx context.Context, db DBTX, arg UpsertIdentityParams) error
	//UpsertInstance
	//
	//  INSERT INTO instances (
	//  	id,
	//  	deployment_id,
	//  	workspace_id,
	//  	project_id,
	//  	region,
	//  	k8s_name,
	//  	address,
	//  	cpu_millicores,
	//  	memory_mib,
	//  	status
	//  )
	//  VALUES (
	//  	?,
	//  	?,
	//  	?,
	//  	?,
	//  	?,
	//  	?,
	//  	?,
	//  	?,
	//  	?,
	//  	?
	//  )
	//  ON DUPLICATE KEY UPDATE
	//  	address = ?,
	//  	cpu_millicores = ?,
	//  	memory_mib = ?,
	//  	status = ?
	UpsertInstance(ctx context.Context, db DBTX, arg UpsertInstanceParams) error
	//UpsertKeySpace
	//
	//  INSERT INTO key_auth (
	//      id,
	//      workspace_id,
	//      created_at_m,
	//      default_prefix,
	//      default_bytes,
	//      store_encrypted_keys
	//  ) VALUES (?, ?, ?, ?, ?, ?)
	//  ON DUPLICATE KEY UPDATE
	//      workspace_id = VALUES(workspace_id),
	//      store_encrypted_keys = VALUES(store_encrypted_keys)
	UpsertKeySpace(ctx context.Context, db DBTX, arg UpsertKeySpaceParams) error
	//UpsertQuota
	//
	//  INSERT INTO quota (
	//      workspace_id,
	//      requests_per_month,
	//      audit_logs_retention_days,
	//      logs_retention_days,
	//      team
	//  ) VALUES (?, ?, ?, ?, ?)
	//  ON DUPLICATE KEY UPDATE
	//      requests_per_month = VALUES(requests_per_month),
	//      audit_logs_retention_days = VALUES(audit_logs_retention_days),
	//      logs_retention_days = VALUES(logs_retention_days)
	UpsertQuota(ctx context.Context, db DBTX, arg UpsertQuotaParams) error
	//UpsertWorkspace
	//
	//  INSERT INTO workspaces (
	//      id,
	//      org_id,
	//      name,
	//      slug,
	//      created_at_m,
	//      tier,
	//      beta_features,
	//      features,
	//      enabled,
	//      delete_protection
	//  ) VALUES (?, ?, ?, ?, ?, ?, ?, '{}', true, false)
	//  ON DUPLICATE KEY UPDATE
	//      beta_features = VALUES(beta_features),
	//      name = VALUES(name)
	UpsertWorkspace(ctx context.Context, db DBTX, arg UpsertWorkspaceParams) error
}
```

### type Queries

```go
type Queries struct{}
```

#### func (Queries) ClearAcmeChallengeTokens

```go
func (q *Queries) ClearAcmeChallengeTokens(ctx context.Context, db DBTX, arg ClearAcmeChallengeTokensParams) error
```

ClearAcmeChallengeTokens

	UPDATE acme_challenges
	SET token = ?, authorization = ?, updated_at = ?
	WHERE domain_id = ?

#### func (Queries) DeleteAcmeChallengeByDomainID

```go
func (q *Queries) DeleteAcmeChallengeByDomainID(ctx context.Context, db DBTX, domainID string) error
```

DeleteAcmeChallengeByDomainID

	DELETE FROM acme_challenges WHERE domain_id = ?

#### func (Queries) DeleteAllKeyPermissionsByKeyID

```go
func (q *Queries) DeleteAllKeyPermissionsByKeyID(ctx context.Context, db DBTX, keyID string) error
```

DeleteAllKeyPermissionsByKeyID

	DELETE FROM keys_permissions
	WHERE key_id = ?

#### func (Queries) DeleteAllKeyRolesByKeyID

```go
func (q *Queries) DeleteAllKeyRolesByKeyID(ctx context.Context, db DBTX, keyID string) error
```

DeleteAllKeyRolesByKeyID

	DELETE FROM keys_roles
	WHERE key_id = ?

#### func (Queries) DeleteCustomDomainByID

```go
func (q *Queries) DeleteCustomDomainByID(ctx context.Context, db DBTX, id string) error
```

DeleteCustomDomainByID

	DELETE FROM custom_domains WHERE id = ?

#### func (Queries) DeleteDeploymentInstances

```go
func (q *Queries) DeleteDeploymentInstances(ctx context.Context, db DBTX, arg DeleteDeploymentInstancesParams) error
```

DeleteDeploymentInstances

	DELETE FROM instances
	WHERE deployment_id = ? AND region = ?

#### func (Queries) DeleteFrontlineRouteByFQDN

```go
func (q *Queries) DeleteFrontlineRouteByFQDN(ctx context.Context, db DBTX, fqdn string) error
```

DeleteFrontlineRouteByFQDN

	DELETE FROM frontline_routes WHERE fully_qualified_domain_name = ?

#### func (Queries) DeleteIdentity

```go
func (q *Queries) DeleteIdentity(ctx context.Context, db DBTX, arg DeleteIdentityParams) error
```

DeleteIdentity

	DELETE FROM identities
	WHERE id = ?
	  AND workspace_id = ?

#### func (Queries) DeleteInstance

```go
func (q *Queries) DeleteInstance(ctx context.Context, db DBTX, arg DeleteInstanceParams) error
```

DeleteInstance

	DELETE FROM instances WHERE k8s_name = ? AND region = ?

#### func (Queries) DeleteKeyByID

```go
func (q *Queries) DeleteKeyByID(ctx context.Context, db DBTX, id string) error
```

DeleteKeyByID

	DELETE k, kp, kr, rl, ek
	FROM `keys` k
	LEFT JOIN keys_permissions kp ON k.id = kp.key_id
	LEFT JOIN keys_roles kr ON k.id = kr.key_id
	LEFT JOIN ratelimits rl ON k.id = rl.key_id
	LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
	WHERE k.id = ?

#### func (Queries) DeleteKeyPermissionByKeyAndPermissionID

```go
func (q *Queries) DeleteKeyPermissionByKeyAndPermissionID(ctx context.Context, db DBTX, arg DeleteKeyPermissionByKeyAndPermissionIDParams) error
```

DeleteKeyPermissionByKeyAndPermissionID

	DELETE FROM keys_permissions
	WHERE key_id = ? AND permission_id = ?

#### func (Queries) DeleteManyKeyPermissionByKeyAndPermissionIDs

```go
func (q *Queries) DeleteManyKeyPermissionByKeyAndPermissionIDs(ctx context.Context, db DBTX, arg DeleteManyKeyPermissionByKeyAndPermissionIDsParams) error
```

DeleteManyKeyPermissionByKeyAndPermissionIDs

	DELETE FROM keys_permissions
	WHERE key_id = ? AND permission_id IN (/*SLICE:ids*/?)

#### func (Queries) DeleteManyKeyPermissionsByPermissionID

```go
func (q *Queries) DeleteManyKeyPermissionsByPermissionID(ctx context.Context, db DBTX, permissionID string) error
```

DeleteManyKeyPermissionsByPermissionID

	DELETE FROM keys_permissions
	WHERE permission_id = ?

#### func (Queries) DeleteManyKeyRolesByKeyAndRoleIDs

```go
func (q *Queries) DeleteManyKeyRolesByKeyAndRoleIDs(ctx context.Context, db DBTX, arg DeleteManyKeyRolesByKeyAndRoleIDsParams) error
```

DeleteManyKeyRolesByKeyAndRoleIDs

	DELETE FROM keys_roles
	WHERE key_id = ? AND role_id IN(/*SLICE:role_ids*/?)

#### func (Queries) DeleteManyKeyRolesByKeyID

```go
func (q *Queries) DeleteManyKeyRolesByKeyID(ctx context.Context, db DBTX, arg DeleteManyKeyRolesByKeyIDParams) error
```

DeleteManyKeyRolesByKeyID

	DELETE FROM keys_roles
	WHERE key_id = ? AND role_id = ?

#### func (Queries) DeleteManyKeyRolesByRoleID

```go
func (q *Queries) DeleteManyKeyRolesByRoleID(ctx context.Context, db DBTX, roleID string) error
```

DeleteManyKeyRolesByRoleID

	DELETE FROM keys_roles
	WHERE role_id = ?

#### func (Queries) DeleteManyRatelimitsByIDs

```go
func (q *Queries) DeleteManyRatelimitsByIDs(ctx context.Context, db DBTX, ids []string) error
```

DeleteManyRatelimitsByIDs

	DELETE FROM ratelimits WHERE id IN (/*SLICE:ids*/?)

#### func (Queries) DeleteManyRatelimitsByIdentityID

```go
func (q *Queries) DeleteManyRatelimitsByIdentityID(ctx context.Context, db DBTX, identityID sql.NullString) error
```

DeleteManyRatelimitsByIdentityID

	DELETE FROM ratelimits WHERE identity_id = ?

#### func (Queries) DeleteManyRolePermissionsByPermissionID

```go
func (q *Queries) DeleteManyRolePermissionsByPermissionID(ctx context.Context, db DBTX, permissionID string) error
```

DeleteManyRolePermissionsByPermissionID

	DELETE FROM roles_permissions
	WHERE permission_id = ?

#### func (Queries) DeleteManyRolePermissionsByRoleID

```go
func (q *Queries) DeleteManyRolePermissionsByRoleID(ctx context.Context, db DBTX, roleID string) error
```

DeleteManyRolePermissionsByRoleID

	DELETE FROM roles_permissions
	WHERE role_id = ?

#### func (Queries) DeleteOldIdentityByExternalID

```go
func (q *Queries) DeleteOldIdentityByExternalID(ctx context.Context, db DBTX, arg DeleteOldIdentityByExternalIDParams) error
```

DeleteOldIdentityByExternalID

	DELETE i, rl
	FROM identities i
	LEFT JOIN ratelimits rl ON rl.identity_id = i.id
	WHERE i.workspace_id = ?
	  AND i.external_id = ?
	  AND i.id != ?
	  AND i.deleted = true

#### func (Queries) DeleteOldIdentityWithRatelimits

```go
func (q *Queries) DeleteOldIdentityWithRatelimits(ctx context.Context, db DBTX, arg DeleteOldIdentityWithRatelimitsParams) error
```

DeleteOldIdentityWithRatelimits

	DELETE i, rl
	FROM identities i
	LEFT JOIN ratelimits rl ON rl.identity_id = i.id
	WHERE i.workspace_id = ?
	  AND (i.id = ? OR i.external_id = ?)
	  AND i.deleted = true

#### func (Queries) DeletePermission

```go
func (q *Queries) DeletePermission(ctx context.Context, db DBTX, permissionID string) error
```

DeletePermission

	DELETE FROM permissions
	WHERE id = ?

#### func (Queries) DeleteRatelimit

```go
func (q *Queries) DeleteRatelimit(ctx context.Context, db DBTX, id string) error
```

DeleteRatelimit

	DELETE FROM `ratelimits` WHERE id = ?

#### func (Queries) DeleteRatelimitNamespace

```go
func (q *Queries) DeleteRatelimitNamespace(ctx context.Context, db DBTX, arg DeleteRatelimitNamespaceParams) (sql.Result, error)
```

DeleteRatelimitNamespace

	UPDATE `ratelimit_namespaces`
	SET deleted_at_m = ?
	WHERE id = ?

#### func (Queries) DeleteRoleByID

```go
func (q *Queries) DeleteRoleByID(ctx context.Context, db DBTX, roleID string) error
```

DeleteRoleByID

	DELETE FROM roles
	WHERE id = ?

#### func (Queries) FindAcmeChallengeByToken

```go
func (q *Queries) FindAcmeChallengeByToken(ctx context.Context, db DBTX, arg FindAcmeChallengeByTokenParams) (AcmeChallenge, error)
```

FindAcmeChallengeByToken

	SELECT pk, domain_id, workspace_id, token, challenge_type, authorization, status, expires_at, created_at, updated_at FROM acme_challenges WHERE workspace_id = ? AND domain_id = ? AND token = ?

#### func (Queries) FindAcmeUserByWorkspaceID

```go
func (q *Queries) FindAcmeUserByWorkspaceID(ctx context.Context, db DBTX, workspaceID string) (AcmeUser, error)
```

FindAcmeUserByWorkspaceID

	SELECT pk, id, workspace_id, encrypted_key, registration_uri, created_at, updated_at FROM acme_users WHERE workspace_id = ? LIMIT 1

#### func (Queries) FindApiByID

```go
func (q *Queries) FindApiByID(ctx context.Context, db DBTX, id string) (Api, error)
```

FindApiByID

	SELECT pk, id, name, workspace_id, ip_whitelist, auth_type, key_auth_id, created_at_m, updated_at_m, deleted_at_m, delete_protection FROM apis WHERE id = ?

#### func (Queries) FindAuditLogTargetByID

```go
func (q *Queries) FindAuditLogTargetByID(ctx context.Context, db DBTX, id string) ([]FindAuditLogTargetByIDRow, error)
```

FindAuditLogTargetByID

	SELECT audit_log_target.pk, audit_log_target.workspace_id, audit_log_target.bucket_id, audit_log_target.bucket, audit_log_target.audit_log_id, audit_log_target.display_name, audit_log_target.type, audit_log_target.id, audit_log_target.name, audit_log_target.meta, audit_log_target.created_at, audit_log_target.updated_at, audit_log.pk, audit_log.id, audit_log.workspace_id, audit_log.bucket, audit_log.bucket_id, audit_log.event, audit_log.time, audit_log.display, audit_log.remote_ip, audit_log.user_agent, audit_log.actor_type, audit_log.actor_id, audit_log.actor_name, audit_log.actor_meta, audit_log.created_at, audit_log.updated_at
	FROM audit_log_target
	JOIN audit_log ON audit_log.id = audit_log_target.audit_log_id
	WHERE audit_log_target.id = ?

#### func (Queries) FindCertificateByHostname

```go
func (q *Queries) FindCertificateByHostname(ctx context.Context, db DBTX, hostname string) (Certificate, error)
```

FindCertificateByHostname

	SELECT pk, id, workspace_id, hostname, certificate, encrypted_private_key, created_at, updated_at FROM certificates WHERE hostname = ?

#### func (Queries) FindCertificatesByHostnames

```go
func (q *Queries) FindCertificatesByHostnames(ctx context.Context, db DBTX, hostnames []string) ([]Certificate, error)
```

FindCertificatesByHostnames

	SELECT pk, id, workspace_id, hostname, certificate, encrypted_private_key, created_at, updated_at FROM certificates WHERE hostname IN (/*SLICE:hostnames*/?)

#### func (Queries) FindCiliumNetworkPoliciesByEnvironmentID

```go
func (q *Queries) FindCiliumNetworkPoliciesByEnvironmentID(ctx context.Context, db DBTX, environmentID string) ([]CiliumNetworkPolicy, error)
```

FindCiliumNetworkPoliciesByEnvironmentID

	SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, region, policy, version, created_at, updated_at FROM cilium_network_policies WHERE environment_id = ?

#### func (Queries) FindCiliumNetworkPolicyByIDAndRegion

```go
func (q *Queries) FindCiliumNetworkPolicyByIDAndRegion(ctx context.Context, db DBTX, arg FindCiliumNetworkPolicyByIDAndRegionParams) (FindCiliumNetworkPolicyByIDAndRegionRow, error)
```

FindCiliumNetworkPolicyByIDAndRegion

	SELECT
	    n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
	    w.k8s_namespace
	FROM `cilium_network_policies` n
	JOIN `workspaces` w ON w.id = n.workspace_id
	WHERE n.region = ? AND n.id = ?
	LIMIT 1

#### func (Queries) FindClickhouseWorkspaceSettingsByWorkspaceID

```go
func (q *Queries) FindClickhouseWorkspaceSettingsByWorkspaceID(ctx context.Context, db DBTX, workspaceID string) (FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error)
```

FindClickhouseWorkspaceSettingsByWorkspaceID

	SELECT
	    c.pk, c.workspace_id, c.username, c.password_encrypted, c.quota_duration_seconds, c.max_queries_per_window, c.max_execution_time_per_window, c.max_query_execution_time, c.max_query_memory_bytes, c.max_query_result_rows, c.created_at, c.updated_at,
	    q.pk, q.workspace_id, q.requests_per_month, q.logs_retention_days, q.audit_logs_retention_days, q.team
	FROM `clickhouse_workspace_settings` c
	JOIN `quota` q ON c.workspace_id = q.workspace_id
	WHERE c.workspace_id = ?

#### func (Queries) FindCustomDomainByDomain

```go
func (q *Queries) FindCustomDomainByDomain(ctx context.Context, db DBTX, domain string) (CustomDomain, error)
```

FindCustomDomainByDomain

	SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at
	FROM custom_domains
	WHERE domain = ?

#### func (Queries) FindCustomDomainByDomainOrWildcard

```go
func (q *Queries) FindCustomDomainByDomainOrWildcard(ctx context.Context, db DBTX, arg FindCustomDomainByDomainOrWildcardParams) (CustomDomain, error)
```

FindCustomDomainByDomainOrWildcard

	SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at FROM custom_domains
	WHERE domain IN (?, ?)
	ORDER BY
	    CASE WHEN domain = ? THEN 0 ELSE 1 END
	LIMIT 1

#### func (Queries) FindCustomDomainById

```go
func (q *Queries) FindCustomDomainById(ctx context.Context, db DBTX, id string) (CustomDomain, error)
```

FindCustomDomainById

	SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at
	FROM custom_domains
	WHERE id = ?

#### func (Queries) FindCustomDomainWithCertByDomain

```go
func (q *Queries) FindCustomDomainWithCertByDomain(ctx context.Context, db DBTX, domain string) (FindCustomDomainWithCertByDomainRow, error)
```

FindCustomDomainWithCertByDomain

	SELECT
	    cd.pk, cd.id, cd.workspace_id, cd.project_id, cd.environment_id, cd.domain, cd.challenge_type, cd.verification_status, cd.verification_token, cd.ownership_verified, cd.cname_verified, cd.target_cname, cd.last_checked_at, cd.check_attempts, cd.verification_error, cd.invocation_id, cd.created_at, cd.updated_at,
	    c.id AS certificate_id
	FROM custom_domains cd
	LEFT JOIN certificates c ON c.hostname = cd.domain
	WHERE cd.domain = ?

#### func (Queries) FindDeploymentById

```go
func (q *Queries) FindDeploymentById(ctx context.Context, db DBTX, id string) (Deployment, error)
```

FindDeploymentById

	SELECT pk, id, k8s_name, workspace_id, project_id, environment_id, image, build_id, git_commit_sha, git_branch, git_commit_message, git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp, sentinel_config, openapi_spec, cpu_millicores, memory_mib, desired_state, encrypted_environment_variables, command, status, created_at, updated_at FROM `deployments` WHERE id = ?

#### func (Queries) FindDeploymentByK8sName

```go
func (q *Queries) FindDeploymentByK8sName(ctx context.Context, db DBTX, k8sName string) (Deployment, error)
```

FindDeploymentByK8sName

	SELECT pk, id, k8s_name, workspace_id, project_id, environment_id, image, build_id, git_commit_sha, git_branch, git_commit_message, git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp, sentinel_config, openapi_spec, cpu_millicores, memory_mib, desired_state, encrypted_environment_variables, command, status, created_at, updated_at FROM `deployments` WHERE k8s_name = ?

#### func (Queries) FindDeploymentRegions

```go
func (q *Queries) FindDeploymentRegions(ctx context.Context, db DBTX, deploymentID string) ([]string, error)
```

Returns all regions where a deployment is configured. Used for fan-out: when a deployment changes, emit state\_change to each region.

	SELECT region
	FROM `deployment_topology`
	WHERE deployment_id = ?

#### func (Queries) FindDeploymentTopologyByIDAndRegion

```go
func (q *Queries) FindDeploymentTopologyByIDAndRegion(ctx context.Context, db DBTX, arg FindDeploymentTopologyByIDAndRegionParams) (FindDeploymentTopologyByIDAndRegionRow, error)
```

FindDeploymentTopologyByIDAndRegion

	SELECT
	    d.id,
	    d.k8s_name,
	    w.k8s_namespace,
	    d.workspace_id,
	    d.project_id,
	    d.environment_id,
	    d.build_id,
	    d.image,
	    dt.region,
	    d.cpu_millicores,
	    d.memory_mib,
	    dt.desired_replicas,
	    d.desired_state,
	    d.encrypted_environment_variables
	FROM `deployment_topology` dt
	INNER JOIN `deployments` d ON dt.deployment_id = d.id
	INNER JOIN `workspaces` w ON d.workspace_id = w.id
	WHERE  dt.region = ?
	    AND dt.deployment_id = ?
	LIMIT 1

#### func (Queries) FindEnvironmentById

```go
func (q *Queries) FindEnvironmentById(ctx context.Context, db DBTX, id string) (FindEnvironmentByIdRow, error)
```

FindEnvironmentById

	SELECT id, workspace_id, project_id, slug, description
	FROM environments
	WHERE id = ?

#### func (Queries) FindEnvironmentByProjectIdAndSlug

```go
func (q *Queries) FindEnvironmentByProjectIdAndSlug(ctx context.Context, db DBTX, arg FindEnvironmentByProjectIdAndSlugParams) (Environment, error)
```

FindEnvironmentByProjectIdAndSlug

	SELECT pk, id, workspace_id, project_id, slug, description, sentinel_config, delete_protection, created_at, updated_at
	FROM environments
	WHERE workspace_id = ?
	  AND project_id = ?
	  AND slug = ?

#### func (Queries) FindEnvironmentVariablesByEnvironmentId

```go
func (q *Queries) FindEnvironmentVariablesByEnvironmentId(ctx context.Context, db DBTX, environmentID string) ([]FindEnvironmentVariablesByEnvironmentIdRow, error)
```

FindEnvironmentVariablesByEnvironmentId

	SELECT `key`, value
	FROM environment_variables
	WHERE environment_id = ?

#### func (Queries) FindFrontlineRouteByFQDN

```go
func (q *Queries) FindFrontlineRouteByFQDN(ctx context.Context, db DBTX, fullyQualifiedDomainName string) (FrontlineRoute, error)
```

FindFrontlineRouteByFQDN

	SELECT pk, id, project_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at, updated_at FROM frontline_routes WHERE fully_qualified_domain_name = ?

#### func (Queries) FindFrontlineRouteForPromotion

```go
func (q *Queries) FindFrontlineRouteForPromotion(ctx context.Context, db DBTX, arg FindFrontlineRouteForPromotionParams) ([]FindFrontlineRouteForPromotionRow, error)
```

FindFrontlineRouteForPromotion

	SELECT
	    id,
	    project_id,
	    environment_id,
	    fully_qualified_domain_name,
	    deployment_id,
	    sticky,
	    created_at,
	    updated_at
	FROM frontline_routes
	WHERE
	  environment_id = ?
	  AND sticky IN (/*SLICE:sticky*/?)
	ORDER BY created_at ASC

#### func (Queries) FindFrontlineRoutesByDeploymentID

```go
func (q *Queries) FindFrontlineRoutesByDeploymentID(ctx context.Context, db DBTX, deploymentID string) ([]FrontlineRoute, error)
```

FindFrontlineRoutesByDeploymentID

	SELECT pk, id, project_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at, updated_at FROM frontline_routes WHERE deployment_id = ?

#### func (Queries) FindFrontlineRoutesForRollback

```go
func (q *Queries) FindFrontlineRoutesForRollback(ctx context.Context, db DBTX, arg FindFrontlineRoutesForRollbackParams) ([]FindFrontlineRoutesForRollbackRow, error)
```

FindFrontlineRoutesForRollback

	SELECT
	    id,
	    project_id,
	    environment_id,
	    fully_qualified_domain_name,
	    deployment_id,
	    sticky,
	    created_at,
	    updated_at
	FROM frontline_routes
	WHERE
	  environment_id = ?
	  AND sticky IN (/*SLICE:sticky*/?)
	ORDER BY created_at ASC

#### func (Queries) FindGithubRepoConnection

```go
func (q *Queries) FindGithubRepoConnection(ctx context.Context, db DBTX, arg FindGithubRepoConnectionParams) (GithubRepoConnection, error)
```

FindGithubRepoConnection

	SELECT
	    pk,
	    project_id,
	    installation_id,
	    repository_id,
	    repository_full_name,
	    created_at,
	    updated_at
	FROM github_repo_connections
	WHERE installation_id = ?
	  AND repository_id = ?

#### func (Queries) FindIdentities

```go
func (q *Queries) FindIdentities(ctx context.Context, db DBTX, arg FindIdentitiesParams) ([]Identity, error)
```

FindIdentities

	SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
	FROM identities
	WHERE workspace_id = ?
	 AND deleted = ?
	 AND (external_id IN(/*SLICE:identities*/?) OR id IN (/*SLICE:identities*/?))

#### func (Queries) FindIdentitiesByExternalId

```go
func (q *Queries) FindIdentitiesByExternalId(ctx context.Context, db DBTX, arg FindIdentitiesByExternalIdParams) ([]Identity, error)
```

FindIdentitiesByExternalId

	SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
	FROM identities
	WHERE workspace_id = ? AND external_id IN (/*SLICE:externalIds*/?) AND deleted = ?

#### func (Queries) FindIdentity

```go
func (q *Queries) FindIdentity(ctx context.Context, db DBTX, arg FindIdentityParams) (FindIdentityRow, error)
```

FindIdentity

	SELECT
	    i.pk, i.id, i.external_id, i.workspace_id, i.environment, i.meta, i.deleted, i.created_at, i.updated_at,
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', rl.id,
	                'name', rl.name,
	                'key_id', rl.key_id,
	                'identity_id', rl.identity_id,
	                'limit', rl.`limit`,
	                'duration', rl.duration,
	                'auto_apply', rl.auto_apply = 1
	            )
	        )
	        FROM ratelimits rl WHERE rl.identity_id = i.id),
	        JSON_ARRAY()
	    ) as ratelimits
	FROM identities i
	JOIN (
	    SELECT id1.id FROM identities id1
	    WHERE id1.id = ?
	      AND id1.workspace_id = ?
	      AND id1.deleted = ?
	    UNION ALL
	    SELECT id2.id FROM identities id2
	    WHERE id2.workspace_id = ?
	      AND id2.external_id = ?
	      AND id2.deleted = ?
	) AS identity_lookup ON i.id = identity_lookup.id
	LIMIT 1

#### func (Queries) FindIdentityByExternalID

```go
func (q *Queries) FindIdentityByExternalID(ctx context.Context, db DBTX, arg FindIdentityByExternalIDParams) (Identity, error)
```

FindIdentityByExternalID

	SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
	FROM identities
	WHERE workspace_id = ?
	  AND external_id = ?
	  AND deleted = ?

#### func (Queries) FindIdentityByID

```go
func (q *Queries) FindIdentityByID(ctx context.Context, db DBTX, arg FindIdentityByIDParams) (Identity, error)
```

FindIdentityByID

	SELECT pk, id, external_id, workspace_id, environment, meta, deleted, created_at, updated_at
	FROM identities
	WHERE workspace_id = ?
	  AND id = ?
	  AND deleted = ?

#### func (Queries) FindInstanceByPodName

```go
func (q *Queries) FindInstanceByPodName(ctx context.Context, db DBTX, arg FindInstanceByPodNameParams) (Instance, error)
```

FindInstanceByPodName

	SELECT
	 pk, id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status
	FROM instances
	  WHERE k8s_name = ? AND region = ?

#### func (Queries) FindInstancesByDeploymentId

```go
func (q *Queries) FindInstancesByDeploymentId(ctx context.Context, db DBTX, deploymentid string) ([]Instance, error)
```

FindInstancesByDeploymentId

	SELECT
	 pk, id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status
	FROM instances
	WHERE deployment_id = ?

#### func (Queries) FindInstancesByDeploymentIdAndRegion

```go
func (q *Queries) FindInstancesByDeploymentIdAndRegion(ctx context.Context, db DBTX, arg FindInstancesByDeploymentIdAndRegionParams) ([]Instance, error)
```

FindInstancesByDeploymentIdAndRegion

	SELECT
	 pk, id, deployment_id, workspace_id, project_id, region, k8s_name, address, cpu_millicores, memory_mib, status
	FROM instances
	WHERE deployment_id = ? AND region = ?

#### func (Queries) FindKeyAuthsByIds

```go
func (q *Queries) FindKeyAuthsByIds(ctx context.Context, db DBTX, arg FindKeyAuthsByIdsParams) ([]FindKeyAuthsByIdsRow, error)
```

FindKeyAuthsByIds

	SELECT ka.id as key_auth_id, a.id as api_id
	FROM apis a
	JOIN key_auth as ka ON ka.id = a.key_auth_id
	WHERE a.workspace_id = ?
	    AND a.id IN (/*SLICE:api_ids*/?)
	    AND ka.deleted_at_m IS NULL
	    AND a.deleted_at_m IS NULL

#### func (Queries) FindKeyAuthsByKeyAuthIds

```go
func (q *Queries) FindKeyAuthsByKeyAuthIds(ctx context.Context, db DBTX, arg FindKeyAuthsByKeyAuthIdsParams) ([]FindKeyAuthsByKeyAuthIdsRow, error)
```

FindKeyAuthsByKeyAuthIds

	SELECT ka.id as key_auth_id, a.id as api_id
	FROM key_auth as ka
	JOIN apis a ON a.key_auth_id = ka.id
	WHERE a.workspace_id = ?
	    AND ka.id IN (/*SLICE:key_auth_ids*/?)
	    AND ka.deleted_at_m IS NULL
	    AND a.deleted_at_m IS NULL

#### func (Queries) FindKeyByID

```go
func (q *Queries) FindKeyByID(ctx context.Context, db DBTX, id string) (Key, error)
```

FindKeyByID

	SELECT pk, id, key_auth_id, hash, start, workspace_id, for_workspace_id, name, owner_id, identity_id, meta, expires, created_at_m, updated_at_m, deleted_at_m, refill_day, refill_amount, last_refill_at, enabled, remaining_requests, ratelimit_async, ratelimit_limit, ratelimit_duration, environment, pending_migration_id FROM `keys` k
	WHERE k.id = ?

#### func (Queries) FindKeyCredits

```go
func (q *Queries) FindKeyCredits(ctx context.Context, db DBTX, id string) (sql.NullInt32, error)
```

FindKeyCredits

	SELECT remaining_requests FROM `keys` k WHERE k.id = ?

#### func (Queries) FindKeyEncryptionByKeyID

```go
func (q *Queries) FindKeyEncryptionByKeyID(ctx context.Context, db DBTX, keyID string) (EncryptedKey, error)
```

FindKeyEncryptionByKeyID

	SELECT pk, workspace_id, key_id, created_at, updated_at, encrypted, encryption_key_id FROM encrypted_keys WHERE key_id = ?

#### func (Queries) FindKeyForVerification

```go
func (q *Queries) FindKeyForVerification(ctx context.Context, db DBTX, hash string) (FindKeyForVerificationRow, error)
```

FindKeyForVerification

	select k.id,
	       k.key_auth_id,
	       k.workspace_id,
	       k.for_workspace_id,
	       k.name,
	       k.meta,
	       k.expires,
	       k.deleted_at_m,
	       k.refill_day,
	       k.refill_amount,
	       k.last_refill_at,
	       k.enabled,
	       k.remaining_requests,
	       k.pending_migration_id,
	       a.ip_whitelist,
	       a.workspace_id  as api_workspace_id,
	       a.id            as api_id,
	       a.deleted_at_m  as api_deleted_at_m,

	       COALESCE(
	               (SELECT JSON_ARRAYAGG(name)
	                FROM (SELECT name
	                      FROM keys_roles kr
	                               JOIN roles r ON r.id = kr.role_id
	                      WHERE kr.key_id = k.id) as roles),
	               JSON_ARRAY()
	       )               as roles,

	       COALESCE(
	               (SELECT JSON_ARRAYAGG(slug)
	                FROM (SELECT slug
	                      FROM keys_permissions kp
	                               JOIN permissions p ON kp.permission_id = p.id
	                      WHERE kp.key_id = k.id

	                      UNION ALL

	                      SELECT slug
	                      FROM keys_roles kr
	                               JOIN roles_permissions rp ON kr.role_id = rp.role_id
	                               JOIN permissions p ON rp.permission_id = p.id
	                      WHERE kr.key_id = k.id) as combined_perms),
	               JSON_ARRAY()
	       )               as permissions,

	       coalesce(
	               (select json_arrayagg(
	                    json_object(
	                       'id', rl.id,
	                       'name', rl.name,
	                       'key_id', rl.key_id,
	                       'identity_id', rl.identity_id,
	                       'limit', rl.limit,
	                       'duration', rl.duration,
	                       'auto_apply', rl.auto_apply
	                    )
	                )
	                from `ratelimits` rl
	                where rl.key_id = k.id
	                   OR rl.identity_id = i.id),
	               json_array()
	       ) as ratelimits,

	       i.id as identity_id,
	       i.external_id,
	       i.meta          as identity_meta,
	       ka.deleted_at_m as key_auth_deleted_at_m,
	       ws.enabled      as workspace_enabled,
	       fws.enabled     as for_workspace_enabled
	from `keys` k
	         JOIN apis a USING (key_auth_id)
	         JOIN key_auth ka ON ka.id = k.key_auth_id
	         JOIN workspaces ws ON ws.id = k.workspace_id
	         LEFT JOIN workspaces fws ON fws.id = k.for_workspace_id
	         LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = 0
	where k.hash = ?
	  and k.deleted_at_m is null

#### func (Queries) FindKeyMigrationByID

```go
func (q *Queries) FindKeyMigrationByID(ctx context.Context, db DBTX, arg FindKeyMigrationByIDParams) (FindKeyMigrationByIDRow, error)
```

FindKeyMigrationByID

	SELECT
	    id,
	    workspace_id,
	    algorithm
	FROM key_migrations
	WHERE id = ?
	and workspace_id = ?

#### func (Queries) FindKeyRoleByKeyAndRoleID

```go
func (q *Queries) FindKeyRoleByKeyAndRoleID(ctx context.Context, db DBTX, arg FindKeyRoleByKeyAndRoleIDParams) ([]KeysRole, error)
```

FindKeyRoleByKeyAndRoleID

	SELECT pk, key_id, role_id, workspace_id, created_at_m, updated_at_m
	FROM keys_roles
	WHERE key_id = ?
	  AND role_id = ?

#### func (Queries) FindKeySpaceByID

```go
func (q *Queries) FindKeySpaceByID(ctx context.Context, db DBTX, id string) (KeyAuth, error)
```

FindKeySpaceByID

	SELECT pk, id, workspace_id, created_at_m, updated_at_m, deleted_at_m, store_encrypted_keys, default_prefix, default_bytes, size_approx, size_last_updated_at FROM `key_auth` WHERE id = ?

#### func (Queries) FindKeysByHash

```go
func (q *Queries) FindKeysByHash(ctx context.Context, db DBTX, hashes []string) ([]FindKeysByHashRow, error)
```

FindKeysByHash

	SELECT id, hash FROM `keys` WHERE hash IN (/*SLICE:hashes*/?)

#### func (Queries) FindLiveApiByID

```go
func (q *Queries) FindLiveApiByID(ctx context.Context, db DBTX, id string) (FindLiveApiByIDRow, error)
```

FindLiveApiByID

	SELECT apis.pk, apis.id, apis.name, apis.workspace_id, apis.ip_whitelist, apis.auth_type, apis.key_auth_id, apis.created_at_m, apis.updated_at_m, apis.deleted_at_m, apis.delete_protection, ka.pk, ka.id, ka.workspace_id, ka.created_at_m, ka.updated_at_m, ka.deleted_at_m, ka.store_encrypted_keys, ka.default_prefix, ka.default_bytes, ka.size_approx, ka.size_last_updated_at
	FROM apis
	JOIN key_auth as ka ON ka.id = apis.key_auth_id
	WHERE apis.id = ?
	    AND ka.deleted_at_m IS NULL
	    AND apis.deleted_at_m IS NULL
	LIMIT 1

#### func (Queries) FindLiveKeyByHash

```go
func (q *Queries) FindLiveKeyByHash(ctx context.Context, db DBTX, hash string) (FindLiveKeyByHashRow, error)
```

FindLiveKeyByHash

	SELECT
	    k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
	    a.pk, a.id, a.name, a.workspace_id, a.ip_whitelist, a.auth_type, a.key_auth_id, a.created_at_m, a.updated_at_m, a.deleted_at_m, a.delete_protection,
	    ka.pk, ka.id, ka.workspace_id, ka.created_at_m, ka.updated_at_m, ka.deleted_at_m, ka.store_encrypted_keys, ka.default_prefix, ka.default_bytes, ka.size_approx, ka.size_last_updated_at,
	    ws.pk, ws.id, ws.org_id, ws.name, ws.slug, ws.k8s_namespace, ws.partition_id, ws.plan, ws.tier, ws.stripe_customer_id, ws.stripe_subscription_id, ws.beta_features, ws.features, ws.subscriptions, ws.enabled, ws.delete_protection, ws.created_at_m, ws.updated_at_m, ws.deleted_at_m,
	    i.id as identity_table_id,
	    i.external_id as identity_external_id,
	    i.meta as identity_meta,
	    ek.encrypted as encrypted_key,
	    ek.encryption_key_id as encryption_key_id,

	    -- Roles with both IDs and names
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', r.id,
	                'name', r.name,
	                'description', r.description
	            )
	        )
	        FROM keys_roles kr
	        JOIN roles r ON r.id = kr.role_id
	        WHERE kr.key_id = k.id),
	        JSON_ARRAY()
	    ) as roles,

	    -- Direct permissions attached to the key
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', p.id,
	                'name', p.name,
	                'slug', p.slug,
	                'description', p.description
	            )
	        )
	        FROM keys_permissions kp
	        JOIN permissions p ON kp.permission_id = p.id
	        WHERE kp.key_id = k.id),
	        JSON_ARRAY()
	    ) as permissions,

	    -- Permissions from roles
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', p.id,
	                'name', p.name,
	                'slug', p.slug,
	                'description', p.description
	            )
	        )
	        FROM keys_roles kr
	        JOIN roles_permissions rp ON kr.role_id = rp.role_id
	        JOIN permissions p ON rp.permission_id = p.id
	        WHERE kr.key_id = k.id),
	        JSON_ARRAY()
	    ) as role_permissions,

	    -- Rate limits
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', rl.id,
	                'name', rl.name,
	                'key_id', rl.key_id,
	                'identity_id', rl.identity_id,
	                'limit', rl.`limit`,
	                'duration', rl.duration,
	                'auto_apply', rl.auto_apply = 1
	            )
	        )
	        FROM ratelimits rl
	        WHERE rl.key_id = k.id OR rl.identity_id = i.id),
	        JSON_ARRAY()
	    ) as ratelimits

	FROM `keys` k
	JOIN apis a ON a.key_auth_id = k.key_auth_id
	JOIN key_auth ka ON ka.id = k.key_auth_id
	JOIN workspaces ws ON ws.id = k.workspace_id
	LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
	LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
	WHERE k.hash = ?
	    AND k.deleted_at_m IS NULL
	    AND a.deleted_at_m IS NULL
	    AND ka.deleted_at_m IS NULL
	    AND ws.deleted_at_m IS NULL

#### func (Queries) FindLiveKeyByID

```go
func (q *Queries) FindLiveKeyByID(ctx context.Context, db DBTX, id string) (FindLiveKeyByIDRow, error)
```

FindLiveKeyByID

	SELECT
	    k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
	    a.pk, a.id, a.name, a.workspace_id, a.ip_whitelist, a.auth_type, a.key_auth_id, a.created_at_m, a.updated_at_m, a.deleted_at_m, a.delete_protection,
	    ka.pk, ka.id, ka.workspace_id, ka.created_at_m, ka.updated_at_m, ka.deleted_at_m, ka.store_encrypted_keys, ka.default_prefix, ka.default_bytes, ka.size_approx, ka.size_last_updated_at,
	    ws.pk, ws.id, ws.org_id, ws.name, ws.slug, ws.k8s_namespace, ws.partition_id, ws.plan, ws.tier, ws.stripe_customer_id, ws.stripe_subscription_id, ws.beta_features, ws.features, ws.subscriptions, ws.enabled, ws.delete_protection, ws.created_at_m, ws.updated_at_m, ws.deleted_at_m,
	    i.id as identity_table_id,
	    i.external_id as identity_external_id,
	    i.meta as identity_meta,
	    ek.encrypted as encrypted_key,
	    ek.encryption_key_id as encryption_key_id,

	    -- Roles with both IDs and names
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', r.id,
	                'name', r.name,
	                'description', r.description
	            )
	        )
	        FROM keys_roles kr
	        JOIN roles r ON r.id = kr.role_id
	        WHERE kr.key_id = k.id),
	        JSON_ARRAY()
	    ) as roles,

	    -- Direct permissions attached to the key
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', p.id,
	                'name', p.name,
	                'slug', p.slug,
	                'description', p.description
	            )
	        )
	        FROM keys_permissions kp
	        JOIN permissions p ON kp.permission_id = p.id
	        WHERE kp.key_id = k.id),
	        JSON_ARRAY()
	    ) as permissions,

	    -- Permissions from roles
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', p.id,
	                'name', p.name,
	                'slug', p.slug,
	                'description', p.description
	            )
	        )
	        FROM keys_roles kr
	        JOIN roles_permissions rp ON kr.role_id = rp.role_id
	        JOIN permissions p ON rp.permission_id = p.id
	        WHERE kr.key_id = k.id),
	        JSON_ARRAY()
	    ) as role_permissions,

	    -- Rate limits
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', rl.id,
	                'name', rl.name,
	                'key_id', rl.key_id,
	                'identity_id', rl.identity_id,
	                'limit', rl.`limit`,
	                'duration', rl.duration,
	                'auto_apply', rl.auto_apply = 1
	            )
	        )
	        FROM ratelimits rl
	        WHERE rl.key_id = k.id
	            OR rl.identity_id = i.id),
	        JSON_ARRAY()
	    ) as ratelimits

	FROM `keys` k
	JOIN apis a ON a.key_auth_id = k.key_auth_id
	JOIN key_auth ka ON ka.id = k.key_auth_id
	JOIN workspaces ws ON ws.id = k.workspace_id
	LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
	LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
	WHERE k.id = ?
	    AND k.deleted_at_m IS NULL
	    AND a.deleted_at_m IS NULL
	    AND ka.deleted_at_m IS NULL
	    AND ws.deleted_at_m IS NULL

#### func (Queries) FindManyRatelimitNamespaces

```go
func (q *Queries) FindManyRatelimitNamespaces(ctx context.Context, db DBTX, arg FindManyRatelimitNamespacesParams) ([]FindManyRatelimitNamespacesRow, error)
```

FindManyRatelimitNamespaces

	SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m,
	       coalesce(
	               (select json_arrayagg(
	                               json_object(
	                                       'id', ro.id,
	                                       'identifier', ro.identifier,
	                                       'limit', ro.limit,
	                                       'duration', ro.duration
	                               )
	                       )
	                from ratelimit_overrides ro where ro.namespace_id = ns.id AND ro.deleted_at_m IS NULL),
	               json_array()
	       ) as overrides
	FROM `ratelimit_namespaces` ns
	WHERE ns.workspace_id = ?
	  AND (ns.id IN (/*SLICE:namespaces*/?) OR ns.name IN (/*SLICE:namespaces*/?))

#### func (Queries) FindManyRolesByIdOrNameWithPerms

```go
func (q *Queries) FindManyRolesByIdOrNameWithPerms(ctx context.Context, db DBTX, arg FindManyRolesByIdOrNameWithPermsParams) ([]FindManyRolesByIdOrNameWithPermsRow, error)
```

FindManyRolesByIdOrNameWithPerms

	SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            json_object(
	                'id', permission.id,
	                'name', permission.name,
	                'slug', permission.slug,
	                'description', permission.description
	           )
	        )
	         FROM (SELECT name, id, slug, description
	               FROM roles_permissions rp
	                        JOIN permissions p ON p.id = rp.permission_id
	               WHERE rp.role_id = r.id) as permission),
	        JSON_ARRAY()
	) as permissions
	FROM roles r
	WHERE r.workspace_id = ? AND (
	    r.id IN (/*SLICE:search*/?)
	    OR r.name IN (/*SLICE:search*/?)
	)

#### func (Queries) FindManyRolesByNamesWithPerms

```go
func (q *Queries) FindManyRolesByNamesWithPerms(ctx context.Context, db DBTX, arg FindManyRolesByNamesWithPermsParams) ([]FindManyRolesByNamesWithPermsRow, error)
```

FindManyRolesByNamesWithPerms

	SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            json_object(
	                'id', permission.id,
	                'name', permission.name,
	                'slug', permission.slug,
	                'description', permission.description
	           )
	        )
	         FROM (SELECT name, id, slug, description
	               FROM roles_permissions rp
	                        JOIN permissions p ON p.id = rp.permission_id
	               WHERE rp.role_id = r.id) as permission),
	        JSON_ARRAY()
	) as permissions
	FROM roles r
	WHERE r.workspace_id = ? AND r.name IN (/*SLICE:names*/?)

#### func (Queries) FindPermissionByID

```go
func (q *Queries) FindPermissionByID(ctx context.Context, db DBTX, permissionID string) (Permission, error)
```

Finds a permission record by its ID Returns: The permission record if found

	SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
	FROM permissions
	WHERE id = ?
	LIMIT 1

#### func (Queries) FindPermissionByIdOrSlug

```go
func (q *Queries) FindPermissionByIdOrSlug(ctx context.Context, db DBTX, arg FindPermissionByIdOrSlugParams) (Permission, error)
```

FindPermissionByIdOrSlug

	SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
	FROM permissions
	WHERE workspace_id = ? AND (id = ? OR slug = ?)

#### func (Queries) FindPermissionByNameAndWorkspaceID

```go
func (q *Queries) FindPermissionByNameAndWorkspaceID(ctx context.Context, db DBTX, arg FindPermissionByNameAndWorkspaceIDParams) (Permission, error)
```

FindPermissionByNameAndWorkspaceID

	SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
	FROM permissions
	WHERE name = ?
	AND workspace_id = ?
	LIMIT 1

#### func (Queries) FindPermissionBySlugAndWorkspaceID

```go
func (q *Queries) FindPermissionBySlugAndWorkspaceID(ctx context.Context, db DBTX, arg FindPermissionBySlugAndWorkspaceIDParams) (Permission, error)
```

FindPermissionBySlugAndWorkspaceID

	SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m
	FROM permissions
	WHERE slug = ?
	AND workspace_id = ?
	LIMIT 1

#### func (Queries) FindPermissionsBySlugs

```go
func (q *Queries) FindPermissionsBySlugs(ctx context.Context, db DBTX, arg FindPermissionsBySlugsParams) ([]Permission, error)
```

FindPermissionsBySlugs

	SELECT pk, id, workspace_id, name, slug, description, created_at_m, updated_at_m FROM permissions WHERE workspace_id = ? AND slug IN (/*SLICE:slugs*/?)

#### func (Queries) FindProjectById

```go
func (q *Queries) FindProjectById(ctx context.Context, db DBTX, id string) (FindProjectByIdRow, error)
```

FindProjectById

	SELECT
	    id,
	    workspace_id,
	    name,
	    slug,
	    git_repository_url,
	    default_branch,
	    delete_protection,
	    live_deployment_id,
	    is_rolled_back,
	    created_at,
	    updated_at,
	    depot_project_id,
	    command
	FROM projects
	WHERE id = ?

#### func (Queries) FindProjectByWorkspaceSlug

```go
func (q *Queries) FindProjectByWorkspaceSlug(ctx context.Context, db DBTX, arg FindProjectByWorkspaceSlugParams) (FindProjectByWorkspaceSlugRow, error)
```

FindProjectByWorkspaceSlug

	SELECT
	    id,
	    workspace_id,
	    name,
	    slug,
	    git_repository_url,
	    default_branch,
	    delete_protection,
	    created_at,
	    updated_at
	FROM projects
	WHERE workspace_id = ? AND slug = ?
	LIMIT 1

#### func (Queries) FindQuotaByWorkspaceID

```go
func (q *Queries) FindQuotaByWorkspaceID(ctx context.Context, db DBTX, workspaceID string) (Quotum, error)
```

FindQuotaByWorkspaceID

	SELECT pk, workspace_id, requests_per_month, logs_retention_days, audit_logs_retention_days, team
	FROM `quota`
	WHERE workspace_id = ?

#### func (Queries) FindRatelimitNamespace

```go
func (q *Queries) FindRatelimitNamespace(ctx context.Context, db DBTX, arg FindRatelimitNamespaceParams) (FindRatelimitNamespaceRow, error)
```

FindRatelimitNamespace

	SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m,
	       coalesce(
	               (select json_arrayagg(
	                               json_object(
	                                       'id', ro.id,
	                                       'identifier', ro.identifier,
	                                       'limit', ro.limit,
	                                       'duration', ro.duration
	                               )
	                       )
	                from ratelimit_overrides ro where ro.namespace_id = ns.id AND ro.deleted_at_m IS NULL),
	               json_array()
	       ) as overrides
	FROM `ratelimit_namespaces` ns
	WHERE ns.workspace_id = ?
	AND (ns.id = ? OR ns.name = ?)

#### func (Queries) FindRatelimitNamespaceByID

```go
func (q *Queries) FindRatelimitNamespaceByID(ctx context.Context, db DBTX, id string) (RatelimitNamespace, error)
```

FindRatelimitNamespaceByID

	SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m FROM `ratelimit_namespaces`
	WHERE id = ?

#### func (Queries) FindRatelimitNamespaceByName

```go
func (q *Queries) FindRatelimitNamespaceByName(ctx context.Context, db DBTX, arg FindRatelimitNamespaceByNameParams) (RatelimitNamespace, error)
```

FindRatelimitNamespaceByName

	SELECT pk, id, workspace_id, name, created_at_m, updated_at_m, deleted_at_m FROM `ratelimit_namespaces`
	WHERE name = ?
	AND workspace_id = ?

#### func (Queries) FindRatelimitOverrideByID

```go
func (q *Queries) FindRatelimitOverrideByID(ctx context.Context, db DBTX, arg FindRatelimitOverrideByIDParams) (RatelimitOverride, error)
```

FindRatelimitOverrideByID

	SELECT pk, id, workspace_id, namespace_id, identifier, `limit`, duration, async, sharding, created_at_m, updated_at_m, deleted_at_m FROM ratelimit_overrides
	WHERE
	    workspace_id = ?
	    AND id = ?

#### func (Queries) FindRatelimitOverrideByIdentifier

```go
func (q *Queries) FindRatelimitOverrideByIdentifier(ctx context.Context, db DBTX, arg FindRatelimitOverrideByIdentifierParams) (RatelimitOverride, error)
```

FindRatelimitOverrideByIdentifier

	SELECT pk, id, workspace_id, namespace_id, identifier, `limit`, duration, async, sharding, created_at_m, updated_at_m, deleted_at_m FROM ratelimit_overrides
	WHERE
	    workspace_id = ?
	    AND namespace_id = ?
	    AND identifier = ?

#### func (Queries) FindRoleByID

```go
func (q *Queries) FindRoleByID(ctx context.Context, db DBTX, roleID string) (Role, error)
```

Finds a role record by its ID Returns: The role record if found

	SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m
	FROM roles
	WHERE id = ?
	LIMIT 1

#### func (Queries) FindRoleByIdOrNameWithPerms

```go
func (q *Queries) FindRoleByIdOrNameWithPerms(ctx context.Context, db DBTX, arg FindRoleByIdOrNameWithPermsParams) (FindRoleByIdOrNameWithPermsRow, error)
```

FindRoleByIdOrNameWithPerms

	SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m, COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            json_object(
	                'id', permission.id,
	                'name', permission.name,
	                'slug', permission.slug,
	                'description', permission.description
	           )
	        )
	         FROM (SELECT name, id, slug, description
	               FROM roles_permissions rp
	                        JOIN permissions p ON p.id = rp.permission_id
	               WHERE rp.role_id = r.id) as permission),
	        JSON_ARRAY()
	) as permissions
	FROM roles r
	WHERE r.workspace_id = ? AND (
	    r.id = ?
	    OR r.name = ?
	)

#### func (Queries) FindRoleByNameAndWorkspaceID

```go
func (q *Queries) FindRoleByNameAndWorkspaceID(ctx context.Context, db DBTX, arg FindRoleByNameAndWorkspaceIDParams) (Role, error)
```

Finds a role record by its name within a specific workspace Returns: The role record if found

	SELECT pk, id, workspace_id, name, description, created_at_m, updated_at_m
	FROM roles
	WHERE name = ?
	AND workspace_id = ?
	LIMIT 1

#### func (Queries) FindRolePermissionByRoleAndPermissionID

```go
func (q *Queries) FindRolePermissionByRoleAndPermissionID(ctx context.Context, db DBTX, arg FindRolePermissionByRoleAndPermissionIDParams) ([]RolesPermission, error)
```

FindRolePermissionByRoleAndPermissionID

	SELECT pk, role_id, permission_id, workspace_id, created_at_m, updated_at_m
	FROM roles_permissions
	WHERE role_id = ?
	  AND permission_id = ?

#### func (Queries) FindRolesByNames

```go
func (q *Queries) FindRolesByNames(ctx context.Context, db DBTX, arg FindRolesByNamesParams) ([]FindRolesByNamesRow, error)
```

FindRolesByNames

	SELECT id, name FROM roles WHERE workspace_id = ? AND name IN (/*SLICE:names*/?)

#### func (Queries) FindSentinelByID

```go
func (q *Queries) FindSentinelByID(ctx context.Context, db DBTX, id string) (Sentinel, error)
```

FindSentinelByID

	SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at FROM sentinels s
	WHERE id = ? LIMIT 1

#### func (Queries) FindSentinelsByEnvironmentID

```go
func (q *Queries) FindSentinelsByEnvironmentID(ctx context.Context, db DBTX, environmentID string) ([]Sentinel, error)
```

FindSentinelsByEnvironmentID

	SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at FROM sentinels WHERE environment_id = ?

#### func (Queries) FindWorkspaceByID

```go
func (q *Queries) FindWorkspaceByID(ctx context.Context, db DBTX, id string) (Workspace, error)
```

FindWorkspaceByID

	SELECT pk, id, org_id, name, slug, k8s_namespace, partition_id, plan, tier, stripe_customer_id, stripe_subscription_id, beta_features, features, subscriptions, enabled, delete_protection, created_at_m, updated_at_m, deleted_at_m FROM `workspaces`
	WHERE id = ?

#### func (Queries) GetKeyAuthByID

```go
func (q *Queries) GetKeyAuthByID(ctx context.Context, db DBTX, id string) (GetKeyAuthByIDRow, error)
```

GetKeyAuthByID

	SELECT
	    id,
	    workspace_id,
	    created_at_m,
	    default_prefix,
	    default_bytes,
	    store_encrypted_keys
	FROM key_auth
	WHERE id = ?
	  AND deleted_at_m IS NULL

#### func (Queries) GetWorkspacesForQuotaCheckByIDs

```go
func (q *Queries) GetWorkspacesForQuotaCheckByIDs(ctx context.Context, db DBTX, workspaceIds []string) ([]GetWorkspacesForQuotaCheckByIDsRow, error)
```

GetWorkspacesForQuotaCheckByIDs

	SELECT
	   w.id,
	   w.org_id,
	   w.name,
	   w.stripe_customer_id,
	   w.tier,
	   w.enabled,
	   q.requests_per_month
	FROM `workspaces` w
	LEFT JOIN quota q ON w.id = q.workspace_id
	WHERE w.id IN (/*SLICE:workspace_ids*/?)

#### func (Queries) HardDeleteWorkspace

```go
func (q *Queries) HardDeleteWorkspace(ctx context.Context, db DBTX, id string) (sql.Result, error)
```

HardDeleteWorkspace

	DELETE FROM `workspaces`
	WHERE id = ?
	AND delete_protection = false

#### func (Queries) InsertAcmeChallenge

```go
func (q *Queries) InsertAcmeChallenge(ctx context.Context, db DBTX, arg InsertAcmeChallengeParams) error
```

InsertAcmeChallenge

	INSERT INTO acme_challenges (
	    workspace_id,
	    domain_id,
	    token,
	    authorization,
	    status,
	    challenge_type,
	    created_at,
	    updated_at,
	    expires_at
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertAcmeUser

```go
func (q *Queries) InsertAcmeUser(ctx context.Context, db DBTX, arg InsertAcmeUserParams) error
```

InsertAcmeUser

	INSERT INTO acme_users (id, workspace_id, encrypted_key, created_at)
	VALUES (?,?,?,?)

#### func (Queries) InsertApi

```go
func (q *Queries) InsertApi(ctx context.Context, db DBTX, arg InsertApiParams) error
```

InsertApi

	INSERT INTO apis (
	    id,
	    name,
	    workspace_id,
	    auth_type,
	    ip_whitelist,
	    key_auth_id,
	    created_at_m,
	    deleted_at_m
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    NULL
	)

#### func (Queries) InsertAuditLog

```go
func (q *Queries) InsertAuditLog(ctx context.Context, db DBTX, arg InsertAuditLogParams) error
```

InsertAuditLog

	INSERT INTO `audit_log` (
	    id,
	    workspace_id,
	    bucket_id,
	    bucket,
	    event,
	    time,
	    display,
	    remote_ip,
	    user_agent,
	    actor_type,
	    actor_id,
	    actor_name,
	    actor_meta,
	    created_at
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    CAST(? AS JSON),
	    ?
	)

#### func (Queries) InsertAuditLogTarget

```go
func (q *Queries) InsertAuditLogTarget(ctx context.Context, db DBTX, arg InsertAuditLogTargetParams) error
```

InsertAuditLogTarget

	INSERT INTO `audit_log_target` (
	    workspace_id,
	    bucket_id,
	    bucket,
	    audit_log_id,
	    display_name,
	    type,
	    id,
	    name,
	    meta,
	    created_at
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    CAST(? AS JSON),
	    ?
	)

#### func (Queries) InsertCertificate

```go
func (q *Queries) InsertCertificate(ctx context.Context, db DBTX, arg InsertCertificateParams) error
```

InsertCertificate

	INSERT INTO certificates (id, workspace_id, hostname, certificate, encrypted_private_key, created_at)
	VALUES (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE
	workspace_id = VALUES(workspace_id),
	hostname = VALUES(hostname),
	certificate = VALUES(certificate),
	encrypted_private_key = VALUES(encrypted_private_key),
	updated_at = ?

#### func (Queries) InsertCiliumNetworkPolicy

```go
func (q *Queries) InsertCiliumNetworkPolicy(ctx context.Context, db DBTX, arg InsertCiliumNetworkPolicyParams) error
```

InsertCiliumNetworkPolicy

	INSERT INTO cilium_network_policies (
	    id,
	    workspace_id,
	    project_id,
	    environment_id,
	    k8s_name,
	    region,
	    policy,
	    version,
	    created_at
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertClickhouseWorkspaceSettings

```go
func (q *Queries) InsertClickhouseWorkspaceSettings(ctx context.Context, db DBTX, arg InsertClickhouseWorkspaceSettingsParams) error
```

InsertClickhouseWorkspaceSettings

	INSERT INTO `clickhouse_workspace_settings` (
	    workspace_id,
	    username,
	    password_encrypted,
	    quota_duration_seconds,
	    max_queries_per_window,
	    max_execution_time_per_window,
	    max_query_execution_time,
	    max_query_memory_bytes,
	    max_query_result_rows,
	    created_at,
	    updated_at
	)
	VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertCustomDomain

```go
func (q *Queries) InsertCustomDomain(ctx context.Context, db DBTX, arg InsertCustomDomainParams) error
```

InsertCustomDomain

	INSERT INTO custom_domains (
	    id, workspace_id, project_id, environment_id, domain,
	    challenge_type, verification_status, verification_token, target_cname, invocation_id, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)

#### func (Queries) InsertDeployment

```go
func (q *Queries) InsertDeployment(ctx context.Context, db DBTX, arg InsertDeploymentParams) error
```

InsertDeployment

	INSERT INTO `deployments` (
	    id,
	    k8s_name,
	    workspace_id,
	    project_id,
	    environment_id,
	    git_commit_sha,
	    git_branch,
	    sentinel_config,
	    git_commit_message,
	    git_commit_author_handle,
	    git_commit_author_avatar_url,
	    git_commit_timestamp,
	    openapi_spec,
	    encrypted_environment_variables,
	    command,
	    status,
	    cpu_millicores,
	    memory_mib,
	    created_at,
	    updated_at
	)
	VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertDeploymentTopology

```go
func (q *Queries) InsertDeploymentTopology(ctx context.Context, db DBTX, arg InsertDeploymentTopologyParams) error
```

InsertDeploymentTopology

	INSERT INTO `deployment_topology` (
	    workspace_id,
	    deployment_id,
	    region,
	    desired_replicas,
	    desired_status,
	    version,
	    created_at
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertEnvironment

```go
func (q *Queries) InsertEnvironment(ctx context.Context, db DBTX, arg InsertEnvironmentParams) error
```

InsertEnvironment

	INSERT INTO environments (
	    id,
	    workspace_id,
	    project_id,
	    slug,
	    description,
	    created_at,
	    updated_at,
	    sentinel_config
	) VALUES (
	    ?, ?, ?, ?, ?, ?, ?, ?
	)

#### func (Queries) InsertFrontlineRoute

```go
func (q *Queries) InsertFrontlineRoute(ctx context.Context, db DBTX, arg InsertFrontlineRouteParams) error
```

InsertFrontlineRoute

	INSERT INTO frontline_routes (
	    id,
	    project_id,
	    deployment_id,
	    environment_id,
	    fully_qualified_domain_name,
	    sticky,
	    created_at,
	    updated_at
	)
	VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertGithubRepoConnection

```go
func (q *Queries) InsertGithubRepoConnection(ctx context.Context, db DBTX, arg InsertGithubRepoConnectionParams) error
```

InsertGithubRepoConnection

	INSERT INTO github_repo_connections (
	    project_id,
	    installation_id,
	    repository_id,
	    repository_full_name,
	    created_at,
	    updated_at
	)
	VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertIdentity

```go
func (q *Queries) InsertIdentity(ctx context.Context, db DBTX, arg InsertIdentityParams) error
```

InsertIdentity

	INSERT INTO `identities` (
	    id,
	    external_id,
	    workspace_id,
	    environment,
	    created_at,
	    meta
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    CAST(? AS JSON)
	)

#### func (Queries) InsertIdentityRatelimit

```go
func (q *Queries) InsertIdentityRatelimit(ctx context.Context, db DBTX, arg InsertIdentityRatelimitParams) error
```

InsertIdentityRatelimit

	INSERT INTO `ratelimits` (
	    id,
	    workspace_id,
	    identity_id,
	    name,
	    `limit`,
	    duration,
	    created_at,
	    auto_apply
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	) ON DUPLICATE KEY UPDATE
	    name = VALUES(name),
	    `limit` = VALUES(`limit`),
	    duration = VALUES(duration),
	    auto_apply = VALUES(auto_apply),
	    updated_at = VALUES(created_at)

#### func (Queries) InsertKey

```go
func (q *Queries) InsertKey(ctx context.Context, db DBTX, arg InsertKeyParams) error
```

InsertKey

	INSERT INTO `keys` (
	    id,
	    key_auth_id,
	    hash,
	    start,
	    workspace_id,
	    for_workspace_id,
	    name,
	    owner_id,
	    identity_id,
	    meta,
	    expires,
	    created_at_m,
	    enabled,
	    remaining_requests,
	    refill_day,
	    refill_amount,
	    pending_migration_id
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    null,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertKeyAuth

```go
func (q *Queries) InsertKeyAuth(ctx context.Context, db DBTX, arg InsertKeyAuthParams) error
```

InsertKeyAuth

	INSERT INTO key_auth (
	    id,
	    workspace_id,
	    created_at_m,
	    default_prefix,
	    default_bytes,
	    store_encrypted_keys
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    false
	)

#### func (Queries) InsertKeyEncryption

```go
func (q *Queries) InsertKeyEncryption(ctx context.Context, db DBTX, arg InsertKeyEncryptionParams) error
```

InsertKeyEncryption

	INSERT INTO encrypted_keys
	(workspace_id, key_id, encrypted, encryption_key_id, created_at)
	VALUES (?, ?, ?, ?, ?)

#### func (Queries) InsertKeyMigration

```go
func (q *Queries) InsertKeyMigration(ctx context.Context, db DBTX, arg InsertKeyMigrationParams) error
```

InsertKeyMigration

	INSERT INTO key_migrations (
	    id,
	    workspace_id,
	    algorithm
	) VALUES (
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertKeyPermission

```go
func (q *Queries) InsertKeyPermission(ctx context.Context, db DBTX, arg InsertKeyPermissionParams) error
```

InsertKeyPermission

	INSERT INTO `keys_permissions` (
	    key_id,
	    permission_id,
	    workspace_id,
	    created_at_m
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?
	) ON DUPLICATE KEY UPDATE updated_at_m = ?

#### func (Queries) InsertKeyRatelimit

```go
func (q *Queries) InsertKeyRatelimit(ctx context.Context, db DBTX, arg InsertKeyRatelimitParams) error
```

InsertKeyRatelimit

	INSERT INTO `ratelimits` (
	    id,
	    workspace_id,
	    key_id,
	    name,
	    `limit`,
	    duration,
	    auto_apply,
	    created_at
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	) ON DUPLICATE KEY UPDATE
	`limit` = VALUES(`limit`),
	duration = VALUES(duration),
	auto_apply = VALUES(auto_apply),
	updated_at = ?

#### func (Queries) InsertKeyRole

```go
func (q *Queries) InsertKeyRole(ctx context.Context, db DBTX, arg InsertKeyRoleParams) error
```

InsertKeyRole

	INSERT INTO keys_roles (
	  key_id,
	  role_id,
	  workspace_id,
	  created_at_m
	)
	VALUES (
	  ?,
	  ?,
	  ?,
	  ?
	)

#### func (Queries) InsertKeySpace

```go
func (q *Queries) InsertKeySpace(ctx context.Context, db DBTX, arg InsertKeySpaceParams) error
```

InsertKeySpace

	INSERT INTO `key_auth` (
	    id,
	    workspace_id,
	    created_at_m,
	    store_encrypted_keys,
	    default_prefix,
	    default_bytes,
	    size_approx,
	    size_last_updated_at
	) VALUES (
	    ?,
	    ?,
	      ?,
	    ?,
	    ?,
	    ?,
	    0,
	    0
	)

#### func (Queries) InsertPermission

```go
func (q *Queries) InsertPermission(ctx context.Context, db DBTX, arg InsertPermissionParams) error
```

InsertPermission

	INSERT INTO permissions (
	  id,
	  workspace_id,
	  name,
	  slug,
	  description,
	  created_at_m
	)
	VALUES (
	  ?,
	  ?,
	  ?,
	  ?,
	  ?,
	  ?
	)

#### func (Queries) InsertProject

```go
func (q *Queries) InsertProject(ctx context.Context, db DBTX, arg InsertProjectParams) error
```

InsertProject

	INSERT INTO projects (
	    id,
	    workspace_id,
	    name,
	    slug,
	    git_repository_url,
	    default_branch,
	    delete_protection,
	    created_at,
	    updated_at
	) VALUES (
	    ?, ?, ?, ?, ?, ?, ?, ?, ?
	)

#### func (Queries) InsertRatelimitNamespace

```go
func (q *Queries) InsertRatelimitNamespace(ctx context.Context, db DBTX, arg InsertRatelimitNamespaceParams) error
```

InsertRatelimitNamespace

	INSERT INTO
	    `ratelimit_namespaces` (
	        id,
	        workspace_id,
	        name,
	        created_at_m,
	        updated_at_m,
	        deleted_at_m
	        )
	VALUES
	    (
	        ?,
	        ?,
	        ?,
	         ?,
	        NULL,
	        NULL
	    )

#### func (Queries) InsertRatelimitOverride

```go
func (q *Queries) InsertRatelimitOverride(ctx context.Context, db DBTX, arg InsertRatelimitOverrideParams) error
```

InsertRatelimitOverride

	INSERT INTO ratelimit_overrides (
	    id,
	    workspace_id,
	    namespace_id,
	    identifier,
	    `limit`,
	    duration,
	    async,
	    created_at_m
	)
	VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    false,
	    ?
	)
	ON DUPLICATE KEY UPDATE
	    `limit` = VALUES(`limit`),
	    duration = VALUES(duration),
	    async = VALUES(async),
	    updated_at_m = ?

#### func (Queries) InsertRole

```go
func (q *Queries) InsertRole(ctx context.Context, db DBTX, arg InsertRoleParams) error
```

InsertRole

	INSERT INTO roles (
	  id,
	  workspace_id,
	  name,
	  description,
	  created_at_m
	)
	VALUES (
	  ?,
	  ?,
	  ?,
	  ?,
	  ?
	)

#### func (Queries) InsertRolePermission

```go
func (q *Queries) InsertRolePermission(ctx context.Context, db DBTX, arg InsertRolePermissionParams) error
```

InsertRolePermission

	INSERT INTO roles_permissions (
	  role_id,
	  permission_id,
	  workspace_id,
	  created_at_m
	)
	VALUES (
	  ?,
	  ?,
	  ?,
	  ?
	)

#### func (Queries) InsertSentinel

```go
func (q *Queries) InsertSentinel(ctx context.Context, db DBTX, arg InsertSentinelParams) error
```

InsertSentinel

	INSERT INTO sentinels (
	    id,
	    workspace_id,
	    environment_id,
	    project_id,
	    k8s_address,
	    k8s_name,
	    region,
	    image,
	    health,
	    desired_replicas,
	    available_replicas,
	    cpu_millicores,
	    memory_mib,
	    version,
	    created_at
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    ?
	)

#### func (Queries) InsertWorkspace

```go
func (q *Queries) InsertWorkspace(ctx context.Context, db DBTX, arg InsertWorkspaceParams) error
```

InsertWorkspace

	INSERT INTO `workspaces` (
	    id,
	    org_id,
	    name,
	    slug,
	    created_at_m,
	    tier,
	    beta_features,
	    features,
	    enabled,
	    delete_protection
	)
	VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    'Free',
	    '{}',
	    '{}',
	    true,
	    true
	)

#### func (Queries) ListCiliumNetworkPoliciesByRegion

```go
func (q *Queries) ListCiliumNetworkPoliciesByRegion(ctx context.Context, db DBTX, arg ListCiliumNetworkPoliciesByRegionParams) ([]ListCiliumNetworkPoliciesByRegionRow, error)
```

ListCiliumNetworkPoliciesByRegion returns cilium network policies for a region with version > after\_version. Used by WatchCiliumNetworkPolicies to stream policy state changes to krane agents.

	SELECT
	    n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
	    w.k8s_namespace
	FROM `cilium_network_policies` n
	JOIN `workspaces` w ON w.id = n.workspace_id
	WHERE n.region = ? AND n.version > ?
	ORDER BY n.version ASC
	LIMIT ?

#### func (Queries) ListCustomDomainsByProjectID

```go
func (q *Queries) ListCustomDomainsByProjectID(ctx context.Context, db DBTX, projectID string) ([]CustomDomain, error)
```

ListCustomDomainsByProjectID

	SELECT pk, id, workspace_id, project_id, environment_id, domain, challenge_type, verification_status, verification_token, ownership_verified, cname_verified, target_cname, last_checked_at, check_attempts, verification_error, invocation_id, created_at, updated_at
	FROM custom_domains
	WHERE project_id = ?
	ORDER BY created_at DESC

#### func (Queries) ListDeploymentTopologyByRegion

```go
func (q *Queries) ListDeploymentTopologyByRegion(ctx context.Context, db DBTX, arg ListDeploymentTopologyByRegionParams) ([]ListDeploymentTopologyByRegionRow, error)
```

ListDeploymentTopologyByRegion returns deployment topologies for a region with version > after\_version. Used by WatchDeployments to stream deployment state changes to krane agents.

	SELECT
	    dt.pk, dt.workspace_id, dt.deployment_id, dt.region, dt.desired_replicas, dt.version, dt.desired_status, dt.created_at, dt.updated_at,
	    d.pk, d.id, d.k8s_name, d.workspace_id, d.project_id, d.environment_id, d.image, d.build_id, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.sentinel_config, d.openapi_spec, d.cpu_millicores, d.memory_mib, d.desired_state, d.encrypted_environment_variables, d.command, d.status, d.created_at, d.updated_at,
	    w.k8s_namespace
	FROM `deployment_topology` dt
	INNER JOIN `deployments` d ON dt.deployment_id = d.id
	INNER JOIN `workspaces` w ON d.workspace_id = w.id
	WHERE dt.region = ? AND dt.version > ?
	ORDER BY dt.version ASC
	LIMIT ?

#### func (Queries) ListDesiredDeploymentTopology

```go
func (q *Queries) ListDesiredDeploymentTopology(ctx context.Context, db DBTX, arg ListDesiredDeploymentTopologyParams) ([]ListDesiredDeploymentTopologyRow, error)
```

ListDesiredDeploymentTopology returns all deployment topologies matching the desired state for a region. Used during bootstrap to stream all running deployments to krane.

	SELECT
	    dt.pk, dt.workspace_id, dt.deployment_id, dt.region, dt.desired_replicas, dt.version, dt.desired_status, dt.created_at, dt.updated_at,
	    d.pk, d.id, d.k8s_name, d.workspace_id, d.project_id, d.environment_id, d.image, d.build_id, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.sentinel_config, d.openapi_spec, d.cpu_millicores, d.memory_mib, d.desired_state, d.encrypted_environment_variables, d.command, d.status, d.created_at, d.updated_at,
	    w.k8s_namespace
	FROM `deployment_topology` dt
	INNER JOIN `deployments` d ON dt.deployment_id = d.id
	INNER JOIN `workspaces` w ON d.workspace_id = w.id
	WHERE (? = '' OR dt.region = ?)
	    AND d.desired_state = ?
	    AND dt.deployment_id > ?
	ORDER BY dt.deployment_id ASC
	LIMIT ?

#### func (Queries) ListDesiredNetworkPolicies

```go
func (q *Queries) ListDesiredNetworkPolicies(ctx context.Context, db DBTX, arg ListDesiredNetworkPoliciesParams) ([]ListDesiredNetworkPoliciesRow, error)
```

ListDesiredNetworkPolicies

	SELECT
	    n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
	    w.k8s_namespace
	FROM `cilium_network_policies` n
	INNER JOIN `workspaces` w ON n.workspace_id = w.id
	WHERE (? = '' OR n.region = ?) AND n.id > ?
	ORDER BY n.id ASC
	LIMIT ?

#### func (Queries) ListDesiredSentinels

```go
func (q *Queries) ListDesiredSentinels(ctx context.Context, db DBTX, arg ListDesiredSentinelsParams) ([]Sentinel, error)
```

ListDesiredSentinels returns all sentinels matching the desired state for a region. Used during bootstrap to stream all running sentinels to krane.

	SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at
	FROM `sentinels`
	WHERE (? = '' OR region = ?)
	    AND desired_state = ?
	    AND id > ?
	ORDER BY id ASC
	LIMIT ?

#### func (Queries) ListDirectPermissionsByKeyID

```go
func (q *Queries) ListDirectPermissionsByKeyID(ctx context.Context, db DBTX, keyID string) ([]Permission, error)
```

ListDirectPermissionsByKeyID

	SELECT p.pk, p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
	FROM keys_permissions kp
	JOIN permissions p ON kp.permission_id = p.id
	WHERE kp.key_id = ?
	ORDER BY p.slug

#### func (Queries) ListExecutableChallenges

```go
func (q *Queries) ListExecutableChallenges(ctx context.Context, db DBTX, verificationTypes []AcmeChallengesChallengeType) ([]ListExecutableChallengesRow, error)
```

ListExecutableChallenges

	SELECT dc.workspace_id, dc.challenge_type, d.domain FROM acme_challenges dc
	JOIN custom_domains d ON dc.domain_id = d.id
	WHERE (dc.status = 'waiting' OR (dc.status = 'verified' AND dc.expires_at <= UNIX_TIMESTAMP(DATE_ADD(NOW(), INTERVAL 30 DAY)) * 1000))
	AND dc.challenge_type IN (/*SLICE:verification_types*/?)
	ORDER BY d.created_at ASC

#### func (Queries) ListIdentities

```go
func (q *Queries) ListIdentities(ctx context.Context, db DBTX, arg ListIdentitiesParams) ([]ListIdentitiesRow, error)
```

ListIdentities

	SELECT
	    i.id,
	    i.external_id,
	    i.workspace_id,
	    i.environment,
	    i.meta,
	    i.deleted,
	    i.created_at,
	    i.updated_at,
	    COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            JSON_OBJECT(
	                'id', r.id,
	                'name', r.name,
	                'limit', r.`limit`,
	                'duration', r.duration,
	                'auto_apply', r.auto_apply = 1
	            )
	        )
	        FROM ratelimits r
	        WHERE r.identity_id = i.id),
	        JSON_ARRAY()
	    ) as ratelimits
	FROM identities i
	WHERE i.workspace_id = ?
	AND i.deleted = ?
	AND i.id >= ?
	ORDER BY i.id ASC
	LIMIT ?

#### func (Queries) ListIdentityRatelimits

```go
func (q *Queries) ListIdentityRatelimits(ctx context.Context, db DBTX, identityID sql.NullString) ([]Ratelimit, error)
```

ListIdentityRatelimits

	SELECT pk, id, name, workspace_id, created_at, updated_at, key_id, identity_id, `limit`, duration, auto_apply
	FROM ratelimits
	WHERE identity_id = ?
	ORDER BY id ASC

#### func (Queries) ListIdentityRatelimitsByID

```go
func (q *Queries) ListIdentityRatelimitsByID(ctx context.Context, db DBTX, identityID sql.NullString) ([]Ratelimit, error)
```

ListIdentityRatelimitsByID

	SELECT pk, id, name, workspace_id, created_at, updated_at, key_id, identity_id, `limit`, duration, auto_apply FROM ratelimits WHERE identity_id = ?

#### func (Queries) ListIdentityRatelimitsByIDs

```go
func (q *Queries) ListIdentityRatelimitsByIDs(ctx context.Context, db DBTX, ids []sql.NullString) ([]Ratelimit, error)
```

ListIdentityRatelimitsByIDs

	SELECT pk, id, name, workspace_id, created_at, updated_at, key_id, identity_id, `limit`, duration, auto_apply FROM ratelimits WHERE identity_id IN (/*SLICE:ids*/?)

#### func (Queries) ListKeysByKeySpaceID

```go
func (q *Queries) ListKeysByKeySpaceID(ctx context.Context, db DBTX, arg ListKeysByKeySpaceIDParams) ([]ListKeysByKeySpaceIDRow, error)
```

ListKeysByKeySpaceID

	SELECT
	  k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
	  i.id as identity_id,
	  i.external_id as external_id,
	  i.meta as identity_meta,
	  ek.encrypted as encrypted_key,
	  ek.encryption_key_id as encryption_key_id

	FROM `keys` k
	LEFT JOIN `identities` i ON k.identity_id = i.id
	LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
	WHERE k.key_auth_id = ?
	AND k.id >= ?
	AND (? IS NULL OR k.identity_id = ?)
	AND k.deleted_at_m IS NULL
	ORDER BY k.id ASC
	LIMIT ?

#### func (Queries) ListLiveKeysByKeySpaceID

```go
func (q *Queries) ListLiveKeysByKeySpaceID(ctx context.Context, db DBTX, arg ListLiveKeysByKeySpaceIDParams) ([]ListLiveKeysByKeySpaceIDRow, error)
```

ListLiveKeysByKeySpaceID

	SELECT k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id, k.name, k.owner_id, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled, k.remaining_requests, k.ratelimit_async, k.ratelimit_limit, k.ratelimit_duration, k.environment, k.pending_migration_id,
	       i.id                 as identity_table_id,
	       i.external_id        as identity_external_id,
	       i.meta               as identity_meta,
	       ek.encrypted         as encrypted_key,
	       ek.encryption_key_id as encryption_key_id,
	       -- Roles with both IDs and names (sorted by name)
	       COALESCE(
	               (SELECT JSON_ARRAYAGG(
	                               JSON_OBJECT(
	                                       'id', r.id,
	                                       'name', r.name,
	                                       'description', r.description
	                               )
	                       )
	                FROM keys_roles kr
	                         JOIN roles r ON r.id = kr.role_id
	                WHERE kr.key_id = k.id
	                ORDER BY r.name),
	               JSON_ARRAY()
	       )                    as roles,
	       -- Direct permissions attached to the key (sorted by slug)
	       COALESCE(
	               (SELECT JSON_ARRAYAGG(
	                               JSON_OBJECT(
	                                       'id', p.id,
	                                       'name', p.name,
	                                       'slug', p.slug,
	                                       'description', p.description
	                               )
	                       )
	                FROM keys_permissions kp
	                         JOIN permissions p ON kp.permission_id = p.id
	                WHERE kp.key_id = k.id
	                ORDER BY p.slug),
	               JSON_ARRAY()
	       )                    as permissions,
	       -- Permissions from roles (sorted by slug)
	       COALESCE(
	               (SELECT JSON_ARRAYAGG(
	                               JSON_OBJECT(
	                                       'id', p.id,
	                                       'name', p.name,
	                                       'slug', p.slug,
	                                       'description', p.description
	                               )
	                       )
	                FROM keys_roles kr
	                         JOIN roles_permissions rp ON kr.role_id = rp.role_id
	                         JOIN permissions p ON rp.permission_id = p.id
	                WHERE kr.key_id = k.id
	                ORDER BY p.slug),
	               JSON_ARRAY()
	       )                    as role_permissions,
	       -- Rate limits
	       COALESCE(
	               (SELECT JSON_ARRAYAGG(
	                               JSON_OBJECT(
	                                       'id', id,
	                                       'name', name,
	                                       'key_id', key_id,
	                                       'identity_id', identity_id,
	                                       'limit', `limit`,
	                                       'duration', duration,
	                                       'auto_apply', auto_apply = 1
	                               )
	                       )
	                FROM (
	                    SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.`limit`, rl.duration, rl.auto_apply
	                    FROM ratelimits rl
	                    WHERE rl.key_id = k.id
	                    UNION ALL
	                    SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.`limit`, rl.duration, rl.auto_apply
	                    FROM ratelimits rl
	                    WHERE rl.identity_id = i.id
	                ) AS combined_rl),
	               JSON_ARRAY()
	       )                    AS ratelimits
	FROM `keys` k
	         STRAIGHT_JOIN key_auth ka ON ka.id = k.key_auth_id
	         LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
	         LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
	WHERE k.key_auth_id = ?
	  AND k.id >= ?
	  AND (? IS NULL OR k.identity_id = ?)
	  AND k.deleted_at_m IS NULL
	  AND ka.deleted_at_m IS NULL
	ORDER BY k.id ASC
	LIMIT ?

#### func (Queries) ListNetworkPolicyByRegion

```go
func (q *Queries) ListNetworkPolicyByRegion(ctx context.Context, db DBTX, arg ListNetworkPolicyByRegionParams) ([]ListNetworkPolicyByRegionRow, error)
```

ListNetworkPolicyByRegion

	SELECT
	    n.pk, n.id, n.workspace_id, n.project_id, n.environment_id, n.k8s_name, n.region, n.policy, n.version, n.created_at, n.updated_at,
	    w.k8s_namespace
	FROM `cilium_network_policies` n
	INNER JOIN `workspaces` w ON n.workspace_id = w.id
	WHERE n.region = ? AND n.version > ?
	ORDER BY n.version ASC
	LIMIT ?

#### func (Queries) ListPermissions

```go
func (q *Queries) ListPermissions(ctx context.Context, db DBTX, arg ListPermissionsParams) ([]Permission, error)
```

ListPermissions

	SELECT p.pk, p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
	FROM permissions p
	WHERE p.workspace_id = ?
	  AND p.id >= ?
	ORDER BY p.id
	LIMIT ?

#### func (Queries) ListPermissionsByKeyID

```go
func (q *Queries) ListPermissionsByKeyID(ctx context.Context, db DBTX, arg ListPermissionsByKeyIDParams) ([]string, error)
```

ListPermissionsByKeyID

	WITH direct_permissions AS (
	    SELECT p.slug as permission_slug
	    FROM keys_permissions kp
	    JOIN permissions p ON kp.permission_id = p.id
	    WHERE kp.key_id = ?
	),
	role_permissions AS (
	    SELECT p.slug as permission_slug
	    FROM keys_roles kr
	    JOIN roles_permissions rp ON kr.role_id = rp.role_id
	    JOIN permissions p ON rp.permission_id = p.id
	    WHERE kr.key_id = ?
	)
	SELECT DISTINCT permission_slug
	FROM (
	    SELECT permission_slug FROM direct_permissions
	    UNION ALL
	    SELECT permission_slug FROM role_permissions
	) all_permissions

#### func (Queries) ListPermissionsByRoleID

```go
func (q *Queries) ListPermissionsByRoleID(ctx context.Context, db DBTX, roleID string) ([]Permission, error)
```

ListPermissionsByRoleID

	SELECT p.pk, p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
	FROM permissions p
	JOIN roles_permissions rp ON p.id = rp.permission_id
	WHERE rp.role_id = ?
	ORDER BY p.slug

#### func (Queries) ListRatelimitOverridesByNamespaceID

```go
func (q *Queries) ListRatelimitOverridesByNamespaceID(ctx context.Context, db DBTX, arg ListRatelimitOverridesByNamespaceIDParams) ([]RatelimitOverride, error)
```

ListRatelimitOverridesByNamespaceID

	SELECT pk, id, workspace_id, namespace_id, identifier, `limit`, duration, async, sharding, created_at_m, updated_at_m, deleted_at_m FROM ratelimit_overrides
	WHERE
	workspace_id = ?
	AND namespace_id = ?
	AND deleted_at_m IS NULL
	AND id >= ?
	ORDER BY id ASC
	LIMIT ?

#### func (Queries) ListRatelimitsByKeyID

```go
func (q *Queries) ListRatelimitsByKeyID(ctx context.Context, db DBTX, keyID sql.NullString) ([]ListRatelimitsByKeyIDRow, error)
```

ListRatelimitsByKeyID

	SELECT
	  id,
	  name,
	  `limit`,
	  duration,
	  auto_apply
	FROM ratelimits
	WHERE key_id = ?

#### func (Queries) ListRatelimitsByKeyIDs

```go
func (q *Queries) ListRatelimitsByKeyIDs(ctx context.Context, db DBTX, keyIds []sql.NullString) ([]ListRatelimitsByKeyIDsRow, error)
```

ListRatelimitsByKeyIDs

	SELECT
	  id,
	  key_id,
	  name,
	  `limit`,
	  duration,
	  auto_apply
	FROM ratelimits
	WHERE key_id IN (/*SLICE:key_ids*/?)
	ORDER BY key_id, id

#### func (Queries) ListRoles

```go
func (q *Queries) ListRoles(ctx context.Context, db DBTX, arg ListRolesParams) ([]ListRolesRow, error)
```

ListRoles

	SELECT r.pk, r.id, r.workspace_id, r.name, r.description, r.created_at_m, r.updated_at_m, COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            json_object(
	                'id', permission.id,
	                'name', permission.name,
	                'slug', permission.slug,
	                'description', permission.description
	           )
	        )
	         FROM (SELECT name, id, slug, description
	               FROM roles_permissions rp
	                        JOIN permissions p ON p.id = rp.permission_id
	               WHERE rp.role_id = r.id) as permission),
	        JSON_ARRAY()
	) as permissions
	FROM roles r
	WHERE r.workspace_id = ?
	AND r.id >= ?
	ORDER BY r.id
	LIMIT ?

#### func (Queries) ListRolesByKeyID

```go
func (q *Queries) ListRolesByKeyID(ctx context.Context, db DBTX, keyID string) ([]ListRolesByKeyIDRow, error)
```

ListRolesByKeyID

	SELECT r.pk, r.id, r.workspace_id, r.name, r.description, r.created_at_m, r.updated_at_m, COALESCE(
	        (SELECT JSON_ARRAYAGG(
	            json_object(
	                'id', permission.id,
	                'name', permission.name,
	                'slug', permission.slug,
	                'description', permission.description
	           )
	        )
	         FROM (SELECT name, id, slug, description
	               FROM roles_permissions rp
	                        JOIN permissions p ON p.id = rp.permission_id
	               WHERE rp.role_id = r.id) as permission),
	        JSON_ARRAY()
	) as permissions
	FROM keys_roles kr
	JOIN roles r ON kr.role_id = r.id
	WHERE kr.key_id = ?
	ORDER BY r.name

#### func (Queries) ListSentinelsByRegion

```go
func (q *Queries) ListSentinelsByRegion(ctx context.Context, db DBTX, arg ListSentinelsByRegionParams) ([]Sentinel, error)
```

ListSentinelsByRegion returns sentinels for a region with version > after\_version. Used by WatchSentinels to stream sentinel state changes to krane agents.

	SELECT pk, id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, version, created_at, updated_at FROM `sentinels`
	WHERE region = ? AND version > ?
	ORDER BY version ASC
	LIMIT ?

#### func (Queries) ListWorkspaces

```go
func (q *Queries) ListWorkspaces(ctx context.Context, db DBTX, cursor string) ([]ListWorkspacesRow, error)
```

ListWorkspaces

	SELECT
	   w.pk, w.id, w.org_id, w.name, w.slug, w.k8s_namespace, w.partition_id, w.plan, w.tier, w.stripe_customer_id, w.stripe_subscription_id, w.beta_features, w.features, w.subscriptions, w.enabled, w.delete_protection, w.created_at_m, w.updated_at_m, w.deleted_at_m,
	   q.pk, q.workspace_id, q.requests_per_month, q.logs_retention_days, q.audit_logs_retention_days, q.team
	FROM `workspaces` w
	LEFT JOIN quota q ON w.id = q.workspace_id
	WHERE w.id > ?
	ORDER BY w.id ASC
	LIMIT 100

#### func (Queries) ListWorkspacesForQuotaCheck

```go
func (q *Queries) ListWorkspacesForQuotaCheck(ctx context.Context, db DBTX, cursor string) ([]ListWorkspacesForQuotaCheckRow, error)
```

ListWorkspacesForQuotaCheck

	SELECT
	   w.id,
	   w.org_id,
	   w.name,
	   w.stripe_customer_id,
	   w.tier,
	   w.enabled,
	   q.requests_per_month
	FROM `workspaces` w
	LEFT JOIN quota q ON w.id = q.workspace_id
	WHERE w.id > ?
	ORDER BY w.id ASC
	LIMIT 100

#### func (Queries) LockIdentityForUpdate

```go
func (q *Queries) LockIdentityForUpdate(ctx context.Context, db DBTX, id string) (string, error)
```

Acquires an exclusive lock on the identity row to prevent concurrent modifications. This should be called at the start of a transaction before modifying identity-related data.

	SELECT id FROM identities
	WHERE id = ?
	FOR UPDATE

#### func (Queries) LockKeyForUpdate

```go
func (q *Queries) LockKeyForUpdate(ctx context.Context, db DBTX, id string) (string, error)
```

Acquires an exclusive lock on the key row to prevent concurrent modifications. This is used to prevent deadlocks when updating key ratelimits concurrently.

	SELECT id FROM `keys`
	WHERE id = ?
	FOR UPDATE

#### func (Queries) ReassignFrontlineRoute

```go
func (q *Queries) ReassignFrontlineRoute(ctx context.Context, db DBTX, arg ReassignFrontlineRouteParams) error
```

ReassignFrontlineRoute

	UPDATE frontline_routes
	SET
	  deployment_id = ?,
	  updated_at = ?
	WHERE id = ?

#### func (Queries) ResetCustomDomainVerification

```go
func (q *Queries) ResetCustomDomainVerification(ctx context.Context, db DBTX, arg ResetCustomDomainVerificationParams) error
```

ResetCustomDomainVerification

	UPDATE custom_domains
	SET verification_status = ?,
	    check_attempts = ?,
	    verification_error = NULL,
	    last_checked_at = NULL,
	    invocation_id = ?,
	    updated_at = ?
	WHERE domain = ?

#### func (Queries) SetWorkspaceK8sNamespace

```go
func (q *Queries) SetWorkspaceK8sNamespace(ctx context.Context, db DBTX, arg SetWorkspaceK8sNamespaceParams) error
```

SetWorkspaceK8sNamespace

	UPDATE `workspaces`
	SET k8s_namespace = ?
	WHERE id = ? AND k8s_namespace IS NULL

#### func (Queries) SoftDeleteApi

```go
func (q *Queries) SoftDeleteApi(ctx context.Context, db DBTX, arg SoftDeleteApiParams) error
```

SoftDeleteApi

	UPDATE apis
	SET deleted_at_m = ?
	WHERE id = ?

#### func (Queries) SoftDeleteIdentity

```go
func (q *Queries) SoftDeleteIdentity(ctx context.Context, db DBTX, arg SoftDeleteIdentityParams) error
```

SoftDeleteIdentity

	UPDATE identities
	SET deleted = 1
	WHERE id = ?
	  AND workspace_id = ?

#### func (Queries) SoftDeleteKeyByID

```go
func (q *Queries) SoftDeleteKeyByID(ctx context.Context, db DBTX, arg SoftDeleteKeyByIDParams) error
```

SoftDeleteKeyByID

	UPDATE `keys` SET deleted_at_m = ? WHERE id = ?

#### func (Queries) SoftDeleteManyKeysByKeySpaceID

```go
func (q *Queries) SoftDeleteManyKeysByKeySpaceID(ctx context.Context, db DBTX, arg SoftDeleteManyKeysByKeySpaceIDParams) error
```

SoftDeleteManyKeysByKeySpaceID

	UPDATE `keys`
	SET deleted_at_m = ?
	WHERE key_auth_id = ?
	AND deleted_at_m IS NULL

#### func (Queries) SoftDeleteRatelimitNamespace

```go
func (q *Queries) SoftDeleteRatelimitNamespace(ctx context.Context, db DBTX, arg SoftDeleteRatelimitNamespaceParams) error
```

SoftDeleteRatelimitNamespace

	UPDATE `ratelimit_namespaces`
	SET
	    deleted_at_m =  ?
	WHERE id = ?

#### func (Queries) SoftDeleteRatelimitOverride

```go
func (q *Queries) SoftDeleteRatelimitOverride(ctx context.Context, db DBTX, arg SoftDeleteRatelimitOverrideParams) error
```

SoftDeleteRatelimitOverride

	UPDATE `ratelimit_overrides`
	SET
	    deleted_at_m =  ?
	WHERE id = ?

#### func (Queries) SoftDeleteWorkspace

```go
func (q *Queries) SoftDeleteWorkspace(ctx context.Context, db DBTX, arg SoftDeleteWorkspaceParams) (sql.Result, error)
```

SoftDeleteWorkspace

	UPDATE `workspaces`
	SET deleted_at_m = ?
	WHERE id = ?
	AND delete_protection = false

#### func (Queries) UpdateAcmeChallengePending

```go
func (q *Queries) UpdateAcmeChallengePending(ctx context.Context, db DBTX, arg UpdateAcmeChallengePendingParams) error
```

UpdateAcmeChallengePending

	UPDATE acme_challenges
	SET status = ?, token = ?, authorization = ?, updated_at = ?
	WHERE domain_id = ?

#### func (Queries) UpdateAcmeChallengeStatus

```go
func (q *Queries) UpdateAcmeChallengeStatus(ctx context.Context, db DBTX, arg UpdateAcmeChallengeStatusParams) error
```

UpdateAcmeChallengeStatus

	UPDATE acme_challenges
	SET status = ?, updated_at = ?
	WHERE domain_id = ?

#### func (Queries) UpdateAcmeChallengeTryClaiming

```go
func (q *Queries) UpdateAcmeChallengeTryClaiming(ctx context.Context, db DBTX, arg UpdateAcmeChallengeTryClaimingParams) error
```

UpdateAcmeChallengeTryClaiming

	UPDATE acme_challenges
	SET status = ?, updated_at = ?
	WHERE domain_id = ? AND status = 'waiting'

#### func (Queries) UpdateAcmeChallengeVerifiedWithExpiry

```go
func (q *Queries) UpdateAcmeChallengeVerifiedWithExpiry(ctx context.Context, db DBTX, arg UpdateAcmeChallengeVerifiedWithExpiryParams) error
```

UpdateAcmeChallengeVerifiedWithExpiry

	UPDATE acme_challenges
	SET status = ?, expires_at = ?, updated_at = ?
	WHERE domain_id = ?

#### func (Queries) UpdateAcmeUserRegistrationURI

```go
func (q *Queries) UpdateAcmeUserRegistrationURI(ctx context.Context, db DBTX, arg UpdateAcmeUserRegistrationURIParams) error
```

UpdateAcmeUserRegistrationURI

	UPDATE acme_users SET registration_uri = ? WHERE id = ?

#### func (Queries) UpdateApiDeleteProtection

```go
func (q *Queries) UpdateApiDeleteProtection(ctx context.Context, db DBTX, arg UpdateApiDeleteProtectionParams) error
```

UpdateApiDeleteProtection

	UPDATE apis
	SET delete_protection = ?
	WHERE id = ?

#### func (Queries) UpdateClickhouseWorkspaceSettingsLimits

```go
func (q *Queries) UpdateClickhouseWorkspaceSettingsLimits(ctx context.Context, db DBTX, arg UpdateClickhouseWorkspaceSettingsLimitsParams) error
```

UpdateClickhouseWorkspaceSettingsLimits

	UPDATE `clickhouse_workspace_settings`
	SET
	    quota_duration_seconds = ?,
	    max_queries_per_window = ?,
	    max_execution_time_per_window = ?,
	    max_query_execution_time = ?,
	    max_query_memory_bytes = ?,
	    max_query_result_rows = ?,
	    updated_at = ?
	WHERE workspace_id = ?

#### func (Queries) UpdateCustomDomainCheckAttempt

```go
func (q *Queries) UpdateCustomDomainCheckAttempt(ctx context.Context, db DBTX, arg UpdateCustomDomainCheckAttemptParams) error
```

UpdateCustomDomainCheckAttempt

	UPDATE custom_domains
	SET check_attempts = ?,
	    last_checked_at = ?,
	    updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateCustomDomainFailed

```go
func (q *Queries) UpdateCustomDomainFailed(ctx context.Context, db DBTX, arg UpdateCustomDomainFailedParams) error
```

UpdateCustomDomainFailed

	UPDATE custom_domains
	SET verification_status = ?,
	    verification_error = ?,
	    updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateCustomDomainInvocationID

```go
func (q *Queries) UpdateCustomDomainInvocationID(ctx context.Context, db DBTX, arg UpdateCustomDomainInvocationIDParams) error
```

UpdateCustomDomainInvocationID

	UPDATE custom_domains
	SET invocation_id = ?,
	    updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateCustomDomainOwnership

```go
func (q *Queries) UpdateCustomDomainOwnership(ctx context.Context, db DBTX, arg UpdateCustomDomainOwnershipParams) error
```

UpdateCustomDomainOwnership

	UPDATE custom_domains
	SET ownership_verified = ?, cname_verified = ?, updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateCustomDomainVerificationStatus

```go
func (q *Queries) UpdateCustomDomainVerificationStatus(ctx context.Context, db DBTX, arg UpdateCustomDomainVerificationStatusParams) error
```

UpdateCustomDomainVerificationStatus

	UPDATE custom_domains
	SET verification_status = ?,
	    updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateDeploymentBuildID

```go
func (q *Queries) UpdateDeploymentBuildID(ctx context.Context, db DBTX, arg UpdateDeploymentBuildIDParams) error
```

UpdateDeploymentBuildID

	UPDATE deployments
	SET build_id = ?, updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateDeploymentImage

```go
func (q *Queries) UpdateDeploymentImage(ctx context.Context, db DBTX, arg UpdateDeploymentImageParams) error
```

UpdateDeploymentImage

	UPDATE deployments
	SET image = ?, updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateDeploymentOpenapiSpec

```go
func (q *Queries) UpdateDeploymentOpenapiSpec(ctx context.Context, db DBTX, arg UpdateDeploymentOpenapiSpecParams) error
```

UpdateDeploymentOpenapiSpec

	UPDATE deployments
	SET openapi_spec = ?, updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateDeploymentStatus

```go
func (q *Queries) UpdateDeploymentStatus(ctx context.Context, db DBTX, arg UpdateDeploymentStatusParams) error
```

UpdateDeploymentStatus

	UPDATE deployments
	SET status = ?, updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateFrontlineRouteDeploymentId

```go
func (q *Queries) UpdateFrontlineRouteDeploymentId(ctx context.Context, db DBTX, arg UpdateFrontlineRouteDeploymentIdParams) error
```

UpdateFrontlineRouteDeploymentId

	UPDATE frontline_routes
	SET deployment_id = ?
	WHERE id = ?

#### func (Queries) UpdateIdentity

```go
func (q *Queries) UpdateIdentity(ctx context.Context, db DBTX, arg UpdateIdentityParams) error
```

UpdateIdentity

	UPDATE `identities`
	SET
	    meta = CAST(? AS JSON),
	    updated_at = NOW()
	WHERE
	    id = ?

#### func (Queries) UpdateKey

```go
func (q *Queries) UpdateKey(ctx context.Context, db DBTX, arg UpdateKeyParams) error
```

UpdateKey

	UPDATE `keys` k SET
	    name = CASE
	        WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	        ELSE k.name
	    END,
	    identity_id = CASE
	        WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	        ELSE k.identity_id
	    END,
	    enabled = CASE
	        WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	        ELSE k.enabled
	    END,
	    meta = CASE
	        WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	        ELSE k.meta
	    END,
	    expires = CASE
	        WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	        ELSE k.expires
	    END,
	    remaining_requests = CASE
	        WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	        ELSE k.remaining_requests
	    END,
	    refill_amount = CASE
	        WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	        ELSE k.refill_amount
	    END,
	    refill_day = CASE
	        WHEN CAST(? AS UNSIGNED) = 1 THEN ?
	        ELSE k.refill_day
	    END,
	    updated_at_m = ?
	WHERE id = ?

#### func (Queries) UpdateKeyCreditsDecrement

```go
func (q *Queries) UpdateKeyCreditsDecrement(ctx context.Context, db DBTX, arg UpdateKeyCreditsDecrementParams) error
```

UpdateKeyCreditsDecrement

	UPDATE `keys`
	SET remaining_requests = CASE
	    WHEN remaining_requests >= ? THEN remaining_requests - ?
	    ELSE 0
	END
	WHERE id = ?

#### func (Queries) UpdateKeyCreditsIncrement

```go
func (q *Queries) UpdateKeyCreditsIncrement(ctx context.Context, db DBTX, arg UpdateKeyCreditsIncrementParams) error
```

UpdateKeyCreditsIncrement

	UPDATE `keys`
	SET remaining_requests = remaining_requests + ?
	WHERE id = ?

#### func (Queries) UpdateKeyCreditsRefill

```go
func (q *Queries) UpdateKeyCreditsRefill(ctx context.Context, db DBTX, arg UpdateKeyCreditsRefillParams) error
```

UpdateKeyCreditsRefill

	UPDATE `keys` SET refill_amount = ?, refill_day = ? WHERE id = ?

#### func (Queries) UpdateKeyCreditsSet

```go
func (q *Queries) UpdateKeyCreditsSet(ctx context.Context, db DBTX, arg UpdateKeyCreditsSetParams) error
```

UpdateKeyCreditsSet

	UPDATE `keys`
	SET remaining_requests = ?
	WHERE id = ?

#### func (Queries) UpdateKeyHashAndMigration

```go
func (q *Queries) UpdateKeyHashAndMigration(ctx context.Context, db DBTX, arg UpdateKeyHashAndMigrationParams) error
```

UpdateKeyHashAndMigration

	UPDATE `keys`
	SET
	    hash = ?,
	    pending_migration_id = ?,
	    start = ?,
	    updated_at_m = ?
	WHERE id = ?

#### func (Queries) UpdateKeySpaceKeyEncryption

```go
func (q *Queries) UpdateKeySpaceKeyEncryption(ctx context.Context, db DBTX, arg UpdateKeySpaceKeyEncryptionParams) error
```

UpdateKeySpaceKeyEncryption

	UPDATE `key_auth` SET store_encrypted_keys = ? WHERE id = ?

#### func (Queries) UpdateProjectDeployments

```go
func (q *Queries) UpdateProjectDeployments(ctx context.Context, db DBTX, arg UpdateProjectDeploymentsParams) error
```

UpdateProjectDeployments

	UPDATE projects
	SET
	  live_deployment_id = ?,
	  is_rolled_back = ?,
	  updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateProjectDepotID

```go
func (q *Queries) UpdateProjectDepotID(ctx context.Context, db DBTX, arg UpdateProjectDepotIDParams) error
```

UpdateProjectDepotID

	UPDATE projects
	SET
	    depot_project_id = ?,
	    updated_at = ?
	WHERE id = ?

#### func (Queries) UpdateRatelimit

```go
func (q *Queries) UpdateRatelimit(ctx context.Context, db DBTX, arg UpdateRatelimitParams) error
```

UpdateRatelimit

	UPDATE `ratelimits`
	SET
	    name = ?,
	    `limit` = ?,
	    duration = ?,
	    auto_apply = ?,
	    updated_at = NOW()
	WHERE
	    id = ?

#### func (Queries) UpdateRatelimitOverride

```go
func (q *Queries) UpdateRatelimitOverride(ctx context.Context, db DBTX, arg UpdateRatelimitOverrideParams) (sql.Result, error)
```

UpdateRatelimitOverride

	UPDATE `ratelimit_overrides`
	SET
	    `limit` = ?,
	    duration = ?,
	    async = ?,
	    updated_at_m= ?
	WHERE id = ?

#### func (Queries) UpdateSentinelAvailableReplicasAndHealth

```go
func (q *Queries) UpdateSentinelAvailableReplicasAndHealth(ctx context.Context, db DBTX, arg UpdateSentinelAvailableReplicasAndHealthParams) error
```

UpdateSentinelAvailableReplicasAndHealth

	UPDATE sentinels SET
	available_replicas = ?,
	health = ?,
	updated_at = ?
	WHERE k8s_name = ?

#### func (Queries) UpdateWorkspaceEnabled

```go
func (q *Queries) UpdateWorkspaceEnabled(ctx context.Context, db DBTX, arg UpdateWorkspaceEnabledParams) (sql.Result, error)
```

UpdateWorkspaceEnabled

	UPDATE `workspaces`
	SET enabled = ?
	WHERE id = ?

#### func (Queries) UpdateWorkspacePlan

```go
func (q *Queries) UpdateWorkspacePlan(ctx context.Context, db DBTX, arg UpdateWorkspacePlanParams) error
```

UpdateWorkspacePlan

	UPDATE `workspaces`
	SET plan = ?
	WHERE id = ?

#### func (Queries) UpsertCustomDomain

```go
func (q *Queries) UpsertCustomDomain(ctx context.Context, db DBTX, arg UpsertCustomDomainParams) error
```

UpsertCustomDomain

	INSERT INTO custom_domains (
	    id, workspace_id, project_id, environment_id, domain,
	    challenge_type, verification_status, verification_token, target_cname, created_at
	)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
	    workspace_id = VALUES(workspace_id),
	    project_id = VALUES(project_id),
	    environment_id = VALUES(environment_id),
	    challenge_type = VALUES(challenge_type),
	    verification_status = VALUES(verification_status),
	    target_cname = VALUES(target_cname),
	    updated_at = ?

#### func (Queries) UpsertEnvironment

```go
func (q *Queries) UpsertEnvironment(ctx context.Context, db DBTX, arg UpsertEnvironmentParams) error
```

UpsertEnvironment

	INSERT INTO environments (
	    id,
	    workspace_id,
	    project_id,
	    slug,
	    sentinel_config,
	    created_at
	) VALUES (?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE slug = VALUES(slug)

#### func (Queries) UpsertIdentity

```go
func (q *Queries) UpsertIdentity(ctx context.Context, db DBTX, arg UpsertIdentityParams) error
```

Inserts a new identity or does nothing if one already exists for this workspace/external\_id. Use FindIdentityByExternalID after this to get the actual ID.

	INSERT INTO `identities` (
	    id,
	    external_id,
	    workspace_id,
	    environment,
	    created_at,
	    meta
	) VALUES (
	    ?,
	    ?,
	    ?,
	    ?,
	    ?,
	    CAST(? AS JSON)
	)
	ON DUPLICATE KEY UPDATE external_id = external_id

#### func (Queries) UpsertInstance

```go
func (q *Queries) UpsertInstance(ctx context.Context, db DBTX, arg UpsertInstanceParams) error
```

UpsertInstance

	INSERT INTO instances (
		id,
		deployment_id,
		workspace_id,
		project_id,
		region,
		k8s_name,
		address,
		cpu_millicores,
		memory_mib,
		status
	)
	VALUES (
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?
	)
	ON DUPLICATE KEY UPDATE
		address = ?,
		cpu_millicores = ?,
		memory_mib = ?,
		status = ?

#### func (Queries) UpsertKeySpace

```go
func (q *Queries) UpsertKeySpace(ctx context.Context, db DBTX, arg UpsertKeySpaceParams) error
```

UpsertKeySpace

	INSERT INTO key_auth (
	    id,
	    workspace_id,
	    created_at_m,
	    default_prefix,
	    default_bytes,
	    store_encrypted_keys
	) VALUES (?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
	    workspace_id = VALUES(workspace_id),
	    store_encrypted_keys = VALUES(store_encrypted_keys)

#### func (Queries) UpsertQuota

```go
func (q *Queries) UpsertQuota(ctx context.Context, db DBTX, arg UpsertQuotaParams) error
```

UpsertQuota

	INSERT INTO quota (
	    workspace_id,
	    requests_per_month,
	    audit_logs_retention_days,
	    logs_retention_days,
	    team
	) VALUES (?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
	    requests_per_month = VALUES(requests_per_month),
	    audit_logs_retention_days = VALUES(audit_logs_retention_days),
	    logs_retention_days = VALUES(logs_retention_days)

#### func (Queries) UpsertWorkspace

```go
func (q *Queries) UpsertWorkspace(ctx context.Context, db DBTX, arg UpsertWorkspaceParams) error
```

UpsertWorkspace

	INSERT INTO workspaces (
	    id,
	    org_id,
	    name,
	    slug,
	    created_at_m,
	    tier,
	    beta_features,
	    features,
	    enabled,
	    delete_protection
	) VALUES (?, ?, ?, ?, ?, ?, ?, '{}', true, false)
	ON DUPLICATE KEY UPDATE
	    beta_features = VALUES(beta_features),
	    name = VALUES(name)

### type Quotum

```go
type Quotum struct {
	Pk                     uint64 `db:"pk"`
	WorkspaceID            string `db:"workspace_id"`
	RequestsPerMonth       int64  `db:"requests_per_month"`
	LogsRetentionDays      int32  `db:"logs_retention_days"`
	AuditLogsRetentionDays int32  `db:"audit_logs_retention_days"`
	Team                   bool   `db:"team"`
}
```

### type Ratelimit

```go
type Ratelimit struct {
	Pk          uint64         `db:"pk"`
	ID          string         `db:"id"`
	Name        string         `db:"name"`
	WorkspaceID string         `db:"workspace_id"`
	CreatedAt   int64          `db:"created_at"`
	UpdatedAt   sql.NullInt64  `db:"updated_at"`
	KeyID       sql.NullString `db:"key_id"`
	IdentityID  sql.NullString `db:"identity_id"`
	Limit       int32          `db:"limit"`
	Duration    int64          `db:"duration"`
	AutoApply   bool           `db:"auto_apply"`
}
```

### type RatelimitInfo

```go
type RatelimitInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	KeyID      dbtype.NullString `json:"key_id"`
	IdentityID dbtype.NullString `json:"identity_id"`
	Limit      int32             `json:"limit"`
	Duration   int64             `json:"duration"`
	AutoApply  bool              `json:"auto_apply"`
}
```

### type RatelimitNamespace

```go
type RatelimitNamespace struct {
	Pk          uint64        `db:"pk"`
	ID          string        `db:"id"`
	WorkspaceID string        `db:"workspace_id"`
	Name        string        `db:"name"`
	CreatedAtM  int64         `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64 `db:"updated_at_m"`
	DeletedAtM  sql.NullInt64 `db:"deleted_at_m"`
}
```

### type RatelimitOverride

```go
type RatelimitOverride struct {
	Pk          uint64                         `db:"pk"`
	ID          string                         `db:"id"`
	WorkspaceID string                         `db:"workspace_id"`
	NamespaceID string                         `db:"namespace_id"`
	Identifier  string                         `db:"identifier"`
	Limit       int32                          `db:"limit"`
	Duration    int32                          `db:"duration"`
	Async       sql.NullBool                   `db:"async"`
	Sharding    NullRatelimitOverridesSharding `db:"sharding"`
	CreatedAtM  int64                          `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64                  `db:"updated_at_m"`
	DeletedAtM  sql.NullInt64                  `db:"deleted_at_m"`
}
```

### type RatelimitOverridesSharding

```go
type RatelimitOverridesSharding string
```

```go
const (
	RatelimitOverridesShardingEdge RatelimitOverridesSharding = "edge"
)
```

#### func (RatelimitOverridesSharding) Scan

```go
func (e *RatelimitOverridesSharding) Scan(src interface{}) error
```

### type ReassignFrontlineRouteParams

```go
type ReassignFrontlineRouteParams struct {
	DeploymentID string        `db:"deployment_id"`
	UpdatedAt    sql.NullInt64 `db:"updated_at"`
	ID           string        `db:"id"`
}
```

### type Replica

```go
type Replica struct {
	mode      string
	db        *sql.DB // Underlying database connection
	debugLogs bool
}
```

Replica wraps a standard SQL database connection and implements the gen.DBTX interface to enable interaction with the generated database code.

#### func (Replica) Begin

```go
func (r *Replica) Begin(ctx context.Context) (DBTx, error)
```

Begin starts a transaction and returns it. This method provides a way to use the Replica in transaction-based operations.

#### func (Replica) ExecContext

```go
func (r *Replica) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
```

ExecContext executes a SQL statement and returns a result summary. It's used for INSERT, UPDATE, DELETE statements that don't return rows.

#### func (Replica) PrepareContext

```go
func (r *Replica) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
```

PrepareContext prepares a SQL statement for later execution.

#### func (Replica) QueryContext

```go
func (r *Replica) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
```

QueryContext executes a SQL query that returns rows.

#### func (Replica) QueryRowContext

```go
func (r *Replica) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
```

QueryRowContext executes a SQL query that returns a single row.

### type ResetCustomDomainVerificationParams

```go
type ResetCustomDomainVerificationParams struct {
	VerificationStatus CustomDomainsVerificationStatus `db:"verification_status"`
	CheckAttempts      int32                           `db:"check_attempts"`
	InvocationID       sql.NullString                  `db:"invocation_id"`
	UpdatedAt          sql.NullInt64                   `db:"updated_at"`
	Domain             string                          `db:"domain"`
}
```

### type Role

```go
type Role struct {
	Pk          uint64         `db:"pk"`
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAtM  int64          `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64  `db:"updated_at_m"`
}
```

### type RoleInfo

```go
type RoleInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description dbtype.NullString `json:"description"`
}
```

RoleInfo  types mirror the database models and support JSON serialization and deserialization. They are used to unmarshal aggregated results (e.g., JSON arrays) returned by database queries.

### type RolesPermission

```go
type RolesPermission struct {
	Pk           uint64        `db:"pk"`
	RoleID       string        `db:"role_id"`
	PermissionID string        `db:"permission_id"`
	WorkspaceID  string        `db:"workspace_id"`
	CreatedAtM   int64         `db:"created_at_m"`
	UpdatedAtM   sql.NullInt64 `db:"updated_at_m"`
}
```

### type Sentinel

```go
type Sentinel struct {
	Pk                uint64                `db:"pk"`
	ID                string                `db:"id"`
	WorkspaceID       string                `db:"workspace_id"`
	ProjectID         string                `db:"project_id"`
	EnvironmentID     string                `db:"environment_id"`
	K8sName           string                `db:"k8s_name"`
	K8sAddress        string                `db:"k8s_address"`
	Region            string                `db:"region"`
	Image             string                `db:"image"`
	DesiredState      SentinelsDesiredState `db:"desired_state"`
	Health            SentinelsHealth       `db:"health"`
	DesiredReplicas   int32                 `db:"desired_replicas"`
	AvailableReplicas int32                 `db:"available_replicas"`
	CpuMillicores     int32                 `db:"cpu_millicores"`
	MemoryMib         int32                 `db:"memory_mib"`
	Version           uint64                `db:"version"`
	CreatedAt         int64                 `db:"created_at"`
	UpdatedAt         sql.NullInt64         `db:"updated_at"`
}
```

### type SentinelsDesiredState

```go
type SentinelsDesiredState string
```

```go
const (
	SentinelsDesiredStateRunning  SentinelsDesiredState = "running"
	SentinelsDesiredStateStandby  SentinelsDesiredState = "standby"
	SentinelsDesiredStateArchived SentinelsDesiredState = "archived"
)
```

#### func (SentinelsDesiredState) Scan

```go
func (e *SentinelsDesiredState) Scan(src interface{}) error
```

### type SentinelsHealth

```go
type SentinelsHealth string
```

```go
const (
	SentinelsHealthUnknown   SentinelsHealth = "unknown"
	SentinelsHealthPaused    SentinelsHealth = "paused"
	SentinelsHealthHealthy   SentinelsHealth = "healthy"
	SentinelsHealthUnhealthy SentinelsHealth = "unhealthy"
)
```

#### func (SentinelsHealth) Scan

```go
func (e *SentinelsHealth) Scan(src interface{}) error
```

### type SetWorkspaceK8sNamespaceParams

```go
type SetWorkspaceK8sNamespaceParams struct {
	K8sNamespace sql.NullString `db:"k8s_namespace"`
	ID           string         `db:"id"`
}
```

### type SoftDeleteApiParams

```go
type SoftDeleteApiParams struct {
	Now   sql.NullInt64 `db:"now"`
	ApiID string        `db:"api_id"`
}
```

### type SoftDeleteIdentityParams

```go
type SoftDeleteIdentityParams struct {
	IdentityID  string `db:"identity_id"`
	WorkspaceID string `db:"workspace_id"`
}
```

### type SoftDeleteKeyByIDParams

```go
type SoftDeleteKeyByIDParams struct {
	Now sql.NullInt64 `db:"now"`
	ID  string        `db:"id"`
}
```

### type SoftDeleteManyKeysByKeySpaceIDParams

```go
type SoftDeleteManyKeysByKeySpaceIDParams struct {
	Now        sql.NullInt64 `db:"now"`
	KeySpaceID string        `db:"key_space_id"`
}
```

### type SoftDeleteRatelimitNamespaceParams

```go
type SoftDeleteRatelimitNamespaceParams struct {
	Now sql.NullInt64 `db:"now"`
	ID  string        `db:"id"`
}
```

### type SoftDeleteRatelimitOverrideParams

```go
type SoftDeleteRatelimitOverrideParams struct {
	Now sql.NullInt64 `db:"now"`
	ID  string        `db:"id"`
}
```

### type SoftDeleteWorkspaceParams

```go
type SoftDeleteWorkspaceParams struct {
	Now sql.NullInt64 `db:"now"`
	ID  string        `db:"id"`
}
```

### type TracedTx

```go
type TracedTx struct {
	tx   *sql.Tx
	mode string
	ctx  context.Context // Store the context for commit/rollback tracing
}
```

TracedTx wraps a sql.Tx to add tracing to all database operations within a transaction

#### func (TracedTx) Commit

```go
func (t *TracedTx) Commit() error
```

Commit commits the transaction with tracing

#### func (TracedTx) ExecContext

```go
func (t *TracedTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
```

ExecContext executes a SQL statement within the transaction with tracing

#### func (TracedTx) PrepareContext

```go
func (t *TracedTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
```

PrepareContext prepares a SQL statement within the transaction with tracing

#### func (TracedTx) QueryContext

```go
func (t *TracedTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
```

QueryContext executes a SQL query within the transaction with tracing

#### func (TracedTx) QueryRowContext

```go
func (t *TracedTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
```

QueryRowContext executes a SQL query that returns a single row within the transaction with tracing

#### func (TracedTx) Rollback

```go
func (t *TracedTx) Rollback() error
```

Rollback rolls back the transaction with tracing

### type UpdateAcmeChallengePendingParams

```go
type UpdateAcmeChallengePendingParams struct {
	Status        AcmeChallengesStatus `db:"status"`
	Token         string               `db:"token"`
	Authorization string               `db:"authorization"`
	UpdatedAt     sql.NullInt64        `db:"updated_at"`
	DomainID      string               `db:"domain_id"`
}
```

### type UpdateAcmeChallengeStatusParams

```go
type UpdateAcmeChallengeStatusParams struct {
	Status    AcmeChallengesStatus `db:"status"`
	UpdatedAt sql.NullInt64        `db:"updated_at"`
	DomainID  string               `db:"domain_id"`
}
```

### type UpdateAcmeChallengeTryClaimingParams

```go
type UpdateAcmeChallengeTryClaimingParams struct {
	Status    AcmeChallengesStatus `db:"status"`
	UpdatedAt sql.NullInt64        `db:"updated_at"`
	DomainID  string               `db:"domain_id"`
}
```

### type UpdateAcmeChallengeVerifiedWithExpiryParams

```go
type UpdateAcmeChallengeVerifiedWithExpiryParams struct {
	Status    AcmeChallengesStatus `db:"status"`
	ExpiresAt int64                `db:"expires_at"`
	UpdatedAt sql.NullInt64        `db:"updated_at"`
	DomainID  string               `db:"domain_id"`
}
```

### type UpdateAcmeUserRegistrationURIParams

```go
type UpdateAcmeUserRegistrationURIParams struct {
	RegistrationUri sql.NullString `db:"registration_uri"`
	ID              string         `db:"id"`
}
```

### type UpdateApiDeleteProtectionParams

```go
type UpdateApiDeleteProtectionParams struct {
	DeleteProtection sql.NullBool `db:"delete_protection"`
	ApiID            string       `db:"api_id"`
}
```

### type UpdateClickhouseWorkspaceSettingsLimitsParams

```go
type UpdateClickhouseWorkspaceSettingsLimitsParams struct {
	QuotaDurationSeconds      int32         `db:"quota_duration_seconds"`
	MaxQueriesPerWindow       int32         `db:"max_queries_per_window"`
	MaxExecutionTimePerWindow int32         `db:"max_execution_time_per_window"`
	MaxQueryExecutionTime     int32         `db:"max_query_execution_time"`
	MaxQueryMemoryBytes       int64         `db:"max_query_memory_bytes"`
	MaxQueryResultRows        int32         `db:"max_query_result_rows"`
	UpdatedAt                 sql.NullInt64 `db:"updated_at"`
	WorkspaceID               string        `db:"workspace_id"`
}
```

### type UpdateCustomDomainCheckAttemptParams

```go
type UpdateCustomDomainCheckAttemptParams struct {
	CheckAttempts int32         `db:"check_attempts"`
	LastCheckedAt sql.NullInt64 `db:"last_checked_at"`
	UpdatedAt     sql.NullInt64 `db:"updated_at"`
	ID            string        `db:"id"`
}
```

### type UpdateCustomDomainFailedParams

```go
type UpdateCustomDomainFailedParams struct {
	VerificationStatus CustomDomainsVerificationStatus `db:"verification_status"`
	VerificationError  sql.NullString                  `db:"verification_error"`
	UpdatedAt          sql.NullInt64                   `db:"updated_at"`
	ID                 string                          `db:"id"`
}
```

### type UpdateCustomDomainInvocationIDParams

```go
type UpdateCustomDomainInvocationIDParams struct {
	InvocationID sql.NullString `db:"invocation_id"`
	UpdatedAt    sql.NullInt64  `db:"updated_at"`
	ID           string         `db:"id"`
}
```

### type UpdateCustomDomainOwnershipParams

```go
type UpdateCustomDomainOwnershipParams struct {
	OwnershipVerified bool          `db:"ownership_verified"`
	CnameVerified     bool          `db:"cname_verified"`
	UpdatedAt         sql.NullInt64 `db:"updated_at"`
	ID                string        `db:"id"`
}
```

### type UpdateCustomDomainVerificationStatusParams

```go
type UpdateCustomDomainVerificationStatusParams struct {
	VerificationStatus CustomDomainsVerificationStatus `db:"verification_status"`
	UpdatedAt          sql.NullInt64                   `db:"updated_at"`
	ID                 string                          `db:"id"`
}
```

### type UpdateDeploymentBuildIDParams

```go
type UpdateDeploymentBuildIDParams struct {
	BuildID   sql.NullString `db:"build_id"`
	UpdatedAt sql.NullInt64  `db:"updated_at"`
	ID        string         `db:"id"`
}
```

### type UpdateDeploymentImageParams

```go
type UpdateDeploymentImageParams struct {
	Image     sql.NullString `db:"image"`
	UpdatedAt sql.NullInt64  `db:"updated_at"`
	ID        string         `db:"id"`
}
```

### type UpdateDeploymentOpenapiSpecParams

```go
type UpdateDeploymentOpenapiSpecParams struct {
	OpenapiSpec sql.NullString `db:"openapi_spec"`
	UpdatedAt   sql.NullInt64  `db:"updated_at"`
	ID          string         `db:"id"`
}
```

### type UpdateDeploymentStatusParams

```go
type UpdateDeploymentStatusParams struct {
	Status    DeploymentsStatus `db:"status"`
	UpdatedAt sql.NullInt64     `db:"updated_at"`
	ID        string            `db:"id"`
}
```

### type UpdateFrontlineRouteDeploymentIdParams

```go
type UpdateFrontlineRouteDeploymentIdParams struct {
	Deploymentid string `db:"deploymentid"`
	ID           string `db:"id"`
}
```

### type UpdateIdentityParams

```go
type UpdateIdentityParams struct {
	Meta json.RawMessage `db:"meta"`
	ID   string          `db:"id"`
}
```

### type UpdateKeyCreditsDecrementParams

```go
type UpdateKeyCreditsDecrementParams struct {
	Credits sql.NullInt32 `db:"credits"`
	ID      string        `db:"id"`
}
```

### type UpdateKeyCreditsIncrementParams

```go
type UpdateKeyCreditsIncrementParams struct {
	Credits sql.NullInt32 `db:"credits"`
	ID      string        `db:"id"`
}
```

### type UpdateKeyCreditsRefillParams

```go
type UpdateKeyCreditsRefillParams struct {
	RefillAmount sql.NullInt32 `db:"refill_amount"`
	RefillDay    sql.NullInt16 `db:"refill_day"`
	ID           string        `db:"id"`
}
```

### type UpdateKeyCreditsSetParams

```go
type UpdateKeyCreditsSetParams struct {
	Credits sql.NullInt32 `db:"credits"`
	ID      string        `db:"id"`
}
```

### type UpdateKeyHashAndMigrationParams

```go
type UpdateKeyHashAndMigrationParams struct {
	Hash               string         `db:"hash"`
	PendingMigrationID sql.NullString `db:"pending_migration_id"`
	Start              string         `db:"start"`
	UpdatedAtM         sql.NullInt64  `db:"updated_at_m"`
	ID                 string         `db:"id"`
}
```

### type UpdateKeyParams

```go
type UpdateKeyParams struct {
	NameSpecified              int64          `db:"name_specified"`
	Name                       sql.NullString `db:"name"`
	IdentityIDSpecified        int64          `db:"identity_id_specified"`
	IdentityID                 sql.NullString `db:"identity_id"`
	EnabledSpecified           int64          `db:"enabled_specified"`
	Enabled                    sql.NullBool   `db:"enabled"`
	MetaSpecified              int64          `db:"meta_specified"`
	Meta                       sql.NullString `db:"meta"`
	ExpiresSpecified           int64          `db:"expires_specified"`
	Expires                    sql.NullTime   `db:"expires"`
	RemainingRequestsSpecified int64          `db:"remaining_requests_specified"`
	RemainingRequests          sql.NullInt32  `db:"remaining_requests"`
	RefillAmountSpecified      int64          `db:"refill_amount_specified"`
	RefillAmount               sql.NullInt32  `db:"refill_amount"`
	RefillDaySpecified         int64          `db:"refill_day_specified"`
	RefillDay                  sql.NullInt16  `db:"refill_day"`
	Now                        sql.NullInt64  `db:"now"`
	ID                         string         `db:"id"`
}
```

### type UpdateKeySpaceKeyEncryptionParams

```go
type UpdateKeySpaceKeyEncryptionParams struct {
	StoreEncryptedKeys bool   `db:"store_encrypted_keys"`
	ID                 string `db:"id"`
}
```

### type UpdateProjectDeploymentsParams

```go
type UpdateProjectDeploymentsParams struct {
	LiveDeploymentID sql.NullString `db:"live_deployment_id"`
	IsRolledBack     bool           `db:"is_rolled_back"`
	UpdatedAt        sql.NullInt64  `db:"updated_at"`
	ID               string         `db:"id"`
}
```

### type UpdateProjectDepotIDParams

```go
type UpdateProjectDepotIDParams struct {
	DepotProjectID sql.NullString `db:"depot_project_id"`
	UpdatedAt      sql.NullInt64  `db:"updated_at"`
	ID             string         `db:"id"`
}
```

### type UpdateRatelimitOverrideParams

```go
type UpdateRatelimitOverrideParams struct {
	Windowlimit int32         `db:"windowlimit"`
	Duration    int32         `db:"duration"`
	Async       sql.NullBool  `db:"async"`
	Now         sql.NullInt64 `db:"now"`
	ID          string        `db:"id"`
}
```

### type UpdateRatelimitParams

```go
type UpdateRatelimitParams struct {
	Name      string `db:"name"`
	Limit     int32  `db:"limit"`
	Duration  int64  `db:"duration"`
	AutoApply bool   `db:"auto_apply"`
	ID        string `db:"id"`
}
```

### type UpdateSentinelAvailableReplicasAndHealthParams

```go
type UpdateSentinelAvailableReplicasAndHealthParams struct {
	AvailableReplicas int32           `db:"available_replicas"`
	Health            SentinelsHealth `db:"health"`
	UpdatedAt         sql.NullInt64   `db:"updated_at"`
	K8sName           string          `db:"k8s_name"`
}
```

### type UpdateWorkspaceEnabledParams

```go
type UpdateWorkspaceEnabledParams struct {
	Enabled bool   `db:"enabled"`
	ID      string `db:"id"`
}
```

### type UpdateWorkspacePlanParams

```go
type UpdateWorkspacePlanParams struct {
	Plan NullWorkspacesPlan `db:"plan"`
	ID   string             `db:"id"`
}
```

### type UpsertCustomDomainParams

```go
type UpsertCustomDomainParams struct {
	ID                 string                          `db:"id"`
	WorkspaceID        string                          `db:"workspace_id"`
	ProjectID          string                          `db:"project_id"`
	EnvironmentID      string                          `db:"environment_id"`
	Domain             string                          `db:"domain"`
	ChallengeType      CustomDomainsChallengeType      `db:"challenge_type"`
	VerificationStatus CustomDomainsVerificationStatus `db:"verification_status"`
	VerificationToken  string                          `db:"verification_token"`
	TargetCname        string                          `db:"target_cname"`
	CreatedAt          int64                           `db:"created_at"`
	UpdatedAt          sql.NullInt64                   `db:"updated_at"`
}
```

### type UpsertEnvironmentParams

```go
type UpsertEnvironmentParams struct {
	ID             string `db:"id"`
	WorkspaceID    string `db:"workspace_id"`
	ProjectID      string `db:"project_id"`
	Slug           string `db:"slug"`
	SentinelConfig []byte `db:"sentinel_config"`
	CreatedAt      int64  `db:"created_at"`
}
```

### type UpsertIdentityParams

```go
type UpsertIdentityParams struct {
	ID          string          `db:"id"`
	ExternalID  string          `db:"external_id"`
	WorkspaceID string          `db:"workspace_id"`
	Environment string          `db:"environment"`
	CreatedAt   int64           `db:"created_at"`
	Meta        json.RawMessage `db:"meta"`
}
```

### type UpsertInstanceParams

```go
type UpsertInstanceParams struct {
	ID            string          `db:"id"`
	DeploymentID  string          `db:"deployment_id"`
	WorkspaceID   string          `db:"workspace_id"`
	ProjectID     string          `db:"project_id"`
	Region        string          `db:"region"`
	K8sName       string          `db:"k8s_name"`
	Address       string          `db:"address"`
	CpuMillicores int32           `db:"cpu_millicores"`
	MemoryMib     int32           `db:"memory_mib"`
	Status        InstancesStatus `db:"status"`
}
```

### type UpsertKeySpaceParams

```go
type UpsertKeySpaceParams struct {
	ID                 string         `db:"id"`
	WorkspaceID        string         `db:"workspace_id"`
	CreatedAtM         int64          `db:"created_at_m"`
	DefaultPrefix      sql.NullString `db:"default_prefix"`
	DefaultBytes       sql.NullInt32  `db:"default_bytes"`
	StoreEncryptedKeys bool           `db:"store_encrypted_keys"`
}
```

### type UpsertQuotaParams

```go
type UpsertQuotaParams struct {
	WorkspaceID            string `db:"workspace_id"`
	RequestsPerMonth       int64  `db:"requests_per_month"`
	AuditLogsRetentionDays int32  `db:"audit_logs_retention_days"`
	LogsRetentionDays      int32  `db:"logs_retention_days"`
	Team                   bool   `db:"team"`
}
```

### type UpsertWorkspaceParams

```go
type UpsertWorkspaceParams struct {
	ID           string          `db:"id"`
	OrgID        string          `db:"org_id"`
	Name         string          `db:"name"`
	Slug         string          `db:"slug"`
	CreatedAtM   int64           `db:"created_at_m"`
	Tier         sql.NullString  `db:"tier"`
	BetaFeatures json.RawMessage `db:"beta_features"`
}
```

### type VercelBinding

```go
type VercelBinding struct {
	Pk            uint64                     `db:"pk"`
	ID            string                     `db:"id"`
	IntegrationID string                     `db:"integration_id"`
	WorkspaceID   string                     `db:"workspace_id"`
	ProjectID     string                     `db:"project_id"`
	Environment   VercelBindingsEnvironment  `db:"environment"`
	ResourceID    string                     `db:"resource_id"`
	ResourceType  VercelBindingsResourceType `db:"resource_type"`
	VercelEnvID   string                     `db:"vercel_env_id"`
	LastEditedBy  string                     `db:"last_edited_by"`
	CreatedAtM    int64                      `db:"created_at_m"`
	UpdatedAtM    sql.NullInt64              `db:"updated_at_m"`
	DeletedAtM    sql.NullInt64              `db:"deleted_at_m"`
}
```

### type VercelBindingsEnvironment

```go
type VercelBindingsEnvironment string
```

```go
const (
	VercelBindingsEnvironmentDevelopment VercelBindingsEnvironment = "development"
	VercelBindingsEnvironmentPreview     VercelBindingsEnvironment = "preview"
	VercelBindingsEnvironmentProduction  VercelBindingsEnvironment = "production"
)
```

#### func (VercelBindingsEnvironment) Scan

```go
func (e *VercelBindingsEnvironment) Scan(src interface{}) error
```

### type VercelBindingsResourceType

```go
type VercelBindingsResourceType string
```

```go
const (
	VercelBindingsResourceTypeRootKey VercelBindingsResourceType = "rootKey"
	VercelBindingsResourceTypeApiId   VercelBindingsResourceType = "apiId"
)
```

#### func (VercelBindingsResourceType) Scan

```go
func (e *VercelBindingsResourceType) Scan(src interface{}) error
```

### type VercelIntegration

```go
type VercelIntegration struct {
	Pk          uint64         `db:"pk"`
	ID          string         `db:"id"`
	WorkspaceID string         `db:"workspace_id"`
	TeamID      sql.NullString `db:"team_id"`
	AccessToken string         `db:"access_token"`
	CreatedAtM  int64          `db:"created_at_m"`
	UpdatedAtM  sql.NullInt64  `db:"updated_at_m"`
	DeletedAtM  sql.NullInt64  `db:"deleted_at_m"`
}
```

### type Workspace

```go
type Workspace struct {
	Pk                   uint64             `db:"pk"`
	ID                   string             `db:"id"`
	OrgID                string             `db:"org_id"`
	Name                 string             `db:"name"`
	Slug                 string             `db:"slug"`
	K8sNamespace         sql.NullString     `db:"k8s_namespace"`
	PartitionID          sql.NullString     `db:"partition_id"`
	Plan                 NullWorkspacesPlan `db:"plan"`
	Tier                 sql.NullString     `db:"tier"`
	StripeCustomerID     sql.NullString     `db:"stripe_customer_id"`
	StripeSubscriptionID sql.NullString     `db:"stripe_subscription_id"`
	BetaFeatures         json.RawMessage    `db:"beta_features"`
	Features             json.RawMessage    `db:"features"`
	Subscriptions        []byte             `db:"subscriptions"`
	Enabled              bool               `db:"enabled"`
	DeleteProtection     sql.NullBool       `db:"delete_protection"`
	CreatedAtM           int64              `db:"created_at_m"`
	UpdatedAtM           sql.NullInt64      `db:"updated_at_m"`
	DeletedAtM           sql.NullInt64      `db:"deleted_at_m"`
}
```

### type WorkspacesPlan

```go
type WorkspacesPlan string
```

```go
const (
	WorkspacesPlanFree       WorkspacesPlan = "free"
	WorkspacesPlanPro        WorkspacesPlan = "pro"
	WorkspacesPlanEnterprise WorkspacesPlan = "enterprise"
)
```

#### func (WorkspacesPlan) Scan

```go
func (e *WorkspacesPlan) Scan(src interface{}) error
```

