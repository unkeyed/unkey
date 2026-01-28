// Package clickhouseuser implements ClickHouse user provisioning workflows.
//
// This package manages the complete lifecycle of ClickHouse users for workspace
// analytics access. It creates users with appropriate permissions, quotas, and
// row-level security (RLS) policies that restrict data access to the owning
// workspace with time-based retention filters.
//
// # Why Restate
//
// ClickHouse user provisioning involves multiple systems: MySQL for credential
// storage, Vault for password encryption, and ClickHouse for user creation.
// Any of these operations can fail independently, leaving the system in an
// inconsistent state. Restate's durable execution ensures that either all steps
// complete successfully or the workflow can be safely retried.
//
// The virtual object model keys workflows by workspace_id, preventing concurrent
// provisioning attempts for the same workspace which could result in password
// mismatches between MySQL and ClickHouse.
//
// # Key Types
//
// [Service] is the main entry point implementing hydrav1.ClickhouseUserServiceServer.
// Configure it via [Config] and create instances with [New]. The service exposes
// [Service.ConfigureUser] for creating or updating ClickHouse users.
//
// # User Configuration
//
// For each workspace, the service creates:
//   - A ClickHouse user with SHA256 password authentication
//   - SELECT permissions on analytics tables (key verifications)
//   - Row-level security policies restricting access to workspace data
//   - Time-based retention filters based on the workspace's quota settings
//   - Query quotas (queries per window, execution time limits)
//   - Settings profile (per-query limits for execution time, memory, result rows)
//
// # Security
//
// Passwords are generated using crypto/rand and encrypted via the Vault API
// before storage in MySQL. The encryption uses the workspace_id as the keyring
// identifier, ensuring passwords can only be decrypted for their owning workspace.
//
// Row-level security policies use ClickHouse's native RLS feature to enforce
// workspace isolation at the database level, preventing any possibility of
// cross-workspace data access even with valid credentials.
//
// # Admin User
//
// This service requires a dedicated ClickHouse admin user with minimal permissions:
//   - CREATE/ALTER/DROP USER, QUOTA, ROW POLICY, SETTINGS PROFILE
//   - GRANT OPTION on analytics tables for granting SELECT to workspace users
//
// This follows the principle of least privilege - the admin user cannot read
// any analytics data itself, only manage user access controls.
//
// # Idempotency
//
// [Service.ConfigureUser] is idempotent. Calling it multiple times for the same
// workspace will preserve the existing password while updating quotas and
// reapplying permissions. This allows safe retries and quota updates without
// breaking existing integrations using the workspace's ClickHouse credentials.
package clickhouseuser
