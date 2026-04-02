import { sql } from "drizzle-orm";
import {
  bigint,
  boolean,
  datetime,
  index,
  int,
  json,
  longblob,
  mysqlEnum,
  mysqlTable,
  text,
  tinyint,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";

export const encryptedKeys = mysqlTable(
  "encrypted_keys",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    keyId: varchar("key_id", { length: 256 }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).default(0).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
    encrypted: varchar({ length: 1024 }).notNull(),
    encryptionKeyId: varchar("encryption_key_id", { length: 256 }).notNull(),
  },
  (table) => [uniqueIndex("key_id_idx").on(table.keyId)],
);

export const keyAuth = mysqlTable(
  "key_auth",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
    deletedAtM: bigint("deleted_at_m", { mode: "number" }),
    storeEncryptedKeys: boolean("store_encrypted_keys").default(false).notNull(),
    defaultPrefix: varchar("default_prefix", { length: 8 }),
    defaultBytes: int("default_bytes").default(16),
    sizeApprox: int("size_approx").default(0).notNull(),
    sizeLastUpdatedAt: bigint("size_last_updated_at", { mode: "number" }).default(0).notNull(),
  },
  (table) => [uniqueIndex("key_auth_id_unique").on(table.id)],
);

export const keyMigrations = mysqlTable(
  "key_migrations",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    algorithm: mysqlEnum(["sha256", "github.com/seamapi/prefixed-api-key"]).notNull(),
  },
  (table) => [
    uniqueIndex("key_migrations_id_unique").on(table.id),
    uniqueIndex("unique_id_per_workspace_id").on(table.id, table.workspaceId),
  ],
);

export const keysPermissions = mysqlTable(
  "keys_permissions",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    keyId: varchar("key_id", { length: 256 }).notNull(),
    permissionId: varchar("permission_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("keys_permissions_key_id_permission_id_workspace_id").on(
      table.keyId,
      table.permissionId,
      table.workspaceId,
    ),
    uniqueIndex("key_id_permission_id_idx").on(table.keyId, table.permissionId),
  ],
);

export const keysRoles = mysqlTable(
  "keys_roles",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    keyId: varchar("key_id", { length: 256 }).notNull(),
    roleId: varchar("role_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("keys_roles_role_id_key_id_workspace_id").on(
      table.roleId,
      table.keyId,
      table.workspaceId,
    ),
    uniqueIndex("unique_key_id_role_id").on(table.keyId, table.roleId),
  ],
);

export const permissions = mysqlTable(
  "permissions",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    name: varchar({ length: 512 }).notNull(),
    slug: varchar({ length: 128 }).notNull(),
    description: varchar({ length: 512 }),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("permissions_id_unique").on(table.id),
    uniqueIndex("unique_slug_per_workspace_idx").on(table.workspaceId, table.slug),
  ],
);

export const rolesPermissions = mysqlTable(
  "roles_permissions",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    roleId: varchar("role_id", { length: 256 }).notNull(),
    permissionId: varchar("permission_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("roles_permissions_role_id_permission_id_workspace_id").on(
      table.roleId,
      table.permissionId,
      table.workspaceId,
    ),
    uniqueIndex("unique_tuple_permission_id_role_id").on(table.permissionId, table.roleId),
  ],
);

export const vercelBindings = mysqlTable(
  "vercel_bindings",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    integrationId: varchar("integration_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    environment: mysqlEnum(["development", "preview", "production"]).notNull(),
    resourceId: varchar("resource_id", { length: 256 }).notNull(),
    resourceType: mysqlEnum("resource_type", ["rootKey", "apiId"]).notNull(),
    vercelEnvId: varchar("vercel_env_id", { length: 256 }).notNull(),
    lastEditedBy: varchar("last_edited_by", { length: 256 }).notNull(),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
    deletedAtM: bigint("deleted_at_m", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("vercel_bindings_id_unique").on(table.id),
    uniqueIndex("project_environment_resource_type_idx").on(
      table.projectId,
      table.environment,
      table.resourceType,
    ),
  ],
);

export const appBuildSettings = mysqlTable(
  "app_build_settings",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    dockerfile: varchar({ length: 500 }).default("Dockerfile").notNull(),
    dockerContext: varchar("docker_context", { length: 500 }).default(".").notNull(),
    watchPaths: json("watch_paths").default(sql`_latin1'[]'`).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [uniqueIndex("app_build_settings_app_env_idx").on(table.appId, table.environmentId)],
);

export const appEnvironmentVariables = mysqlTable(
  "app_environment_variables",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 128 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    key: varchar({ length: 256 }).notNull(),
    value: varchar({ length: 4096 }).notNull(),
    type: mysqlEnum(["recoverable", "writeonly"]).notNull(),
    description: varchar({ length: 255 }),
    deleteProtection: boolean("delete_protection").default(false),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("app_environment_variables_id_unique").on(table.id),
    uniqueIndex("app_env_id_key").on(table.appId, table.environmentId, table.key),
  ],
);

export const appRuntimeSettings = mysqlTable(
  "app_runtime_settings",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    port: int().default(8080).notNull(),
    cpuMillicores: int("cpu_millicores").default(250).notNull(),
    memoryMib: int("memory_mib").default(256).notNull(),
    command: json().default(sql`_latin1'[]'`).notNull(),
    healthcheck: json(),
    shutdownSignal: mysqlEnum("shutdown_signal", ["SIGTERM", "SIGINT", "SIGQUIT", "SIGKILL"])
      .default("SIGTERM")
      .notNull(),
    sentinelConfig: longblob("sentinel_config").notNull(),
    openapiSpecPath: varchar("openapi_spec_path", { length: 512 }),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [uniqueIndex("app_runtime_settings_app_env_idx").on(table.appId, table.environmentId)],
);

export const certificates = mysqlTable(
  "certificates",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 64 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    hostname: varchar({ length: 255 }).notNull(),
    certificate: text().notNull(),
    encryptedPrivateKey: text("encrypted_private_key").notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("certificates_id_unique").on(table.id),
    uniqueIndex("unique_hostname").on(table.hostname),
  ],
);

export const clickhouseWorkspaceSettings = mysqlTable(
  "clickhouse_workspace_settings",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    username: varchar({ length: 256 }).notNull(),
    passwordEncrypted: text("password_encrypted").notNull(),
    quotaDurationSeconds: int("quota_duration_seconds").default(3600).notNull(),
    maxQueriesPerWindow: int("max_queries_per_window").default(1000).notNull(),
    maxExecutionTimePerWindow: int("max_execution_time_per_window").default(1800).notNull(),
    maxQueryExecutionTime: int("max_query_execution_time").default(30).notNull(),
    maxQueryMemoryBytes: bigint("max_query_memory_bytes", { mode: "number" })
      .default(1000000000)
      .notNull(),
    maxQueryResultRows: int("max_query_result_rows").default(10000).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).default(0).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("clickhouse_workspace_settings_workspace_id_unique").on(table.workspaceId),
    uniqueIndex("clickhouse_workspace_settings_username_unique").on(table.username),
  ],
);

export const githubAppInstallations = mysqlTable(
  "github_app_installations",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    installationId: bigint("installation_id", { mode: "number" }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("workspace_installation_idx").on(table.workspaceId, table.installationId),
  ],
);

export const identities = mysqlTable(
  "identities",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    externalId: varchar("external_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    environment: varchar({ length: 256 }).default("default").notNull(),
    meta: json(),
    deleted: boolean().default(false).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("identities_id_unique").on(table.id),
    uniqueIndex("workspace_id_external_id_deleted_idx").on(
      table.workspaceId,
      table.externalId,
      table.deleted,
    ),
  ],
);

export const openapiSpecs = mysqlTable(
  "openapi_specs",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 128 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 128 }),
    portalConfigId: varchar("portal_config_id", { length: 256 }),
    content: longblob().notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("openapi_specs_id_unique").on(table.id),
    uniqueIndex("workspace_deployment_idx").on(table.workspaceId, table.deploymentId),
    uniqueIndex("workspace_portal_config_idx").on(table.workspaceId, table.portalConfigId),
  ],
);

export const projects = mysqlTable(
  "projects",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 64 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    name: varchar({ length: 256 }).notNull(),
    slug: varchar({ length: 256 }).notNull(),
    depotProjectId: varchar("depot_project_id", { length: 255 }),
    deleteProtection: boolean("delete_protection").default(false),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("projects_id_unique").on(table.id),
    uniqueIndex("workspace_slug_idx").on(table.workspaceId, table.slug),
  ],
);

export const quota = mysqlTable(
  "quota",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    requestsPerMonth: bigint("requests_per_month", { mode: "number" }).default(0).notNull(),
    logsRetentionDays: int("logs_retention_days").default(0).notNull(),
    auditLogsRetentionDays: int("audit_logs_retention_days").default(0).notNull(),
    team: boolean().default(false).notNull(),
    ratelimitApiLimit: int("ratelimit_api_limit", { unsigned: true }),
    ratelimitApiDuration: int("ratelimit_api_duration", { unsigned: true }),
    allocatedCpuMillicoresTotal: int("allocated_cpu_millicores_total", { unsigned: true })
      .default(10000)
      .notNull(),
    allocatedMemoryMibTotal: int("allocated_memory_mib_total", { unsigned: true })
      .default(20480)
      .notNull(),
  },
  (table) => [uniqueIndex("quota_workspace_id_unique").on(table.workspaceId)],
);

export const ratelimitNamespaces = mysqlTable(
  "ratelimit_namespaces",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    name: varchar({ length: 512 }).notNull(),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
    deletedAtM: bigint("deleted_at_m", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("ratelimit_namespaces_id_unique").on(table.id),
    uniqueIndex("unique_name_per_workspace_idx").on(table.workspaceId, table.name),
  ],
);

export const ratelimitOverrides = mysqlTable(
  "ratelimit_overrides",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    namespaceId: varchar("namespace_id", { length: 256 }).notNull(),
    identifier: varchar({ length: 512 }).notNull(),
    limit: int().notNull(),
    duration: int().notNull(),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
    deletedAtM: bigint("deleted_at_m", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("ratelimit_overrides_id_unique").on(table.id),
    uniqueIndex("unique_identifier_per_namespace_idx").on(table.namespaceId, table.identifier),
  ],
);

export const ratelimits = mysqlTable(
  "ratelimits",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    name: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
    keyId: varchar("key_id", { length: 256 }),
    identityId: varchar("identity_id", { length: 256 }),
    limit: int().notNull(),
    duration: bigint({ mode: "number" }).notNull(),
    autoApply: boolean("auto_apply").default(false).notNull(),
  },
  (table) => [
    uniqueIndex("ratelimits_id_unique").on(table.id),
    uniqueIndex("unique_name_per_key_idx").on(table.keyId, table.name),
    uniqueIndex("unique_name_per_identity_idx").on(table.identityId, table.name),
  ],
);

export const vercelIntegrations = mysqlTable(
  "vercel_integrations",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    teamId: varchar("team_id", { length: 256 }),
    accessToken: varchar("access_token", { length: 256 }).notNull(),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
    deletedAtM: bigint("deleted_at_m", { mode: "number" }),
  },
  (table) => [uniqueIndex("vercel_integrations_id_unique").on(table.id)],
);

export const workspaces = mysqlTable(
  "workspaces",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    orgId: varchar("org_id", { length: 256 }).notNull(),
    name: varchar({ length: 256 }).notNull(),
    slug: varchar({ length: 64 }).notNull(),
    k8sNamespace: varchar("k8s_namespace", { length: 256 }),
    tier: varchar({ length: 256 }).default("Free"),
    stripeCustomerId: varchar("stripe_customer_id", { length: 256 }),
    stripeSubscriptionId: varchar("stripe_subscription_id", { length: 256 }),
    betaFeatures: json("beta_features").notNull(),
    subscriptions: json(),
    enabled: boolean().default(true).notNull(),
    deleteProtection: boolean("delete_protection").default(false),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
    deletedAtM: bigint("deleted_at_m", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("workspaces_id_unique").on(table.id),
    uniqueIndex("workspaces_org_id_unique").on(table.orgId),
    uniqueIndex("workspaces_slug_unique").on(table.slug),
    uniqueIndex("workspaces_k8s_namespace_unique").on(table.k8sNamespace),
  ],
);

export const acmeChallenges = mysqlTable(
  "acme_challenges",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    domainId: varchar("domain_id", { length: 255 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    token: varchar({ length: 255 }).notNull(),
    challengeType: mysqlEnum("challenge_type", ["HTTP-01", "DNS-01"]).notNull(),
    authorization: varchar({ length: 255 }).notNull(),
    status: mysqlEnum(["waiting", "pending", "verified", "failed"]).notNull(),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("acme_challenges_domain_id_unique").on(table.domainId),
    index("workspace_idx").on(table.workspaceId),
    index("status_idx").on(table.status),
  ],
);

export const acmeUsers = mysqlTable(
  "acme_users",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 128 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    encryptedKey: text("encrypted_key").notNull(),
    registrationUri: text("registration_uri"),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("acme_users_id_unique").on(table.id),
    index("domain_idx").on(table.workspaceId),
  ],
);

export const apis = mysqlTable(
  "apis",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    name: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    ipWhitelist: varchar("ip_whitelist", { length: 512 }),
    authType: mysqlEnum("auth_type", ["key", "jwt"]),
    keyAuthId: varchar("key_auth_id", { length: 256 }),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
    deletedAtM: bigint("deleted_at_m", { mode: "number" }),
    deleteProtection: boolean("delete_protection").default(false),
  },
  (table) => [
    uniqueIndex("apis_id_unique").on(table.id),
    uniqueIndex("apis_key_auth_id_unique").on(table.keyAuthId),
    index("workspace_id_idx").on(table.workspaceId),
  ],
);

export const appRegionalSettings = mysqlTable(
  "app_regional_settings",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    regionId: varchar("region_id", { length: 64 }).notNull(),
    replicas: int().default(1).notNull(),
    horizontalAutoscalingPolicyId: varchar("horizontal_autoscaling_policy_id", { length: 64 }),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("unique_app_env_region").on(table.appId, table.environmentId, table.regionId),
    index("workspace_idx").on(table.workspaceId),
  ],
);

export const apps = mysqlTable(
  "apps",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 64 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 64 }).notNull(),
    name: varchar({ length: 256 }).notNull(),
    slug: varchar({ length: 256 }).notNull(),
    defaultBranch: varchar("default_branch", { length: 256 }).default("main").notNull(),
    currentDeploymentId: varchar("current_deployment_id", { length: 256 }),
    isRolledBack: boolean("is_rolled_back").default(false).notNull(),
    deleteProtection: boolean("delete_protection").default(false),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("apps_id_unique").on(table.id),
    uniqueIndex("apps_project_slug_idx").on(table.projectId, table.slug),
    index("apps_workspace_idx").on(table.workspaceId),
  ],
);

export const auditLog = mysqlTable(
  "audit_log",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    bucket: varchar({ length: 256 }).default("unkey_mutations").notNull(),
    bucketId: varchar("bucket_id", { length: 256 }).notNull(),
    event: varchar({ length: 256 }).notNull(),
    time: bigint({ mode: "number" }).notNull(),
    display: varchar({ length: 256 }).notNull(),
    remoteIp: varchar("remote_ip", { length: 256 }),
    userAgent: varchar("user_agent", { length: 256 }),
    actorType: varchar("actor_type", { length: 256 }).notNull(),
    actorId: varchar("actor_id", { length: 256 }).notNull(),
    actorName: varchar("actor_name", { length: 256 }),
    actorMeta: json("actor_meta"),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("audit_log_id_unique").on(table.id),
    index("workspace_id_idx").on(table.workspaceId),
    index("bucket_id_idx").on(table.bucketId),
    index("bucket_idx").on(table.bucket),
    index("event_idx").on(table.event),
    index("actor_id_idx").on(table.actorId),
    index("time_idx").on(table.time),
  ],
);

export const auditLogTarget = mysqlTable(
  "audit_log_target",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    bucketId: varchar("bucket_id", { length: 256 }).notNull(),
    bucket: varchar({ length: 256 }).default("unkey_mutations").notNull(),
    auditLogId: varchar("audit_log_id", { length: 256 }).notNull(),
    displayName: varchar("display_name", { length: 256 }).notNull(),
    type: varchar({ length: 256 }).notNull(),
    id: varchar({ length: 256 }).notNull(),
    name: varchar({ length: 256 }),
    meta: json(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("unique_id_per_log").on(table.auditLogId, table.id),
    index("bucket").on(table.bucket),
    index("id_idx").on(table.id),
  ],
);

export const ciliumNetworkPolicies = mysqlTable(
  "cilium_network_policies",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 64 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 128 }).notNull(),
    k8sName: varchar("k8s_name", { length: 64 }).notNull(),
    k8sNamespace: varchar("k8s_namespace", { length: 255 }).notNull(),
    regionId: varchar("region_id", { length: 64 }).notNull(),
    policy: json().notNull(),
    version: bigint({ unsigned: true, mode: "number" }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("cilium_network_policies_id_unique").on(table.id),
    uniqueIndex("unique_version_per_region").on(table.regionId, table.version),
    index("idx_deployment_region").on(table.deploymentId, table.regionId),
  ],
);

export const clusters = mysqlTable(
  "clusters",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 64 }).notNull(),
    regionId: varchar("region_id", { length: 64 }).notNull(),
    lastHeartbeatAt: bigint("last_heartbeat_at", { unsigned: true, mode: "number" }).notNull(),
  },
  (table) => [
    uniqueIndex("clusters_id_unique").on(table.id),
    uniqueIndex("clusters_region_id_unique").on(table.regionId),
  ],
);

export const customDomains = mysqlTable(
  "custom_domains",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 128 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 256 }).notNull(),
    domain: varchar({ length: 256 }).notNull(),
    challengeType: mysqlEnum("challenge_type", ["HTTP-01", "DNS-01"]).notNull(),
    verificationStatus: mysqlEnum("verification_status", [
      "pending",
      "verifying",
      "verified",
      "failed",
    ])
      .default("pending")
      .notNull(),
    verificationToken: varchar("verification_token", { length: 64 }).notNull(),
    ownershipVerified: boolean("ownership_verified").default(false).notNull(),
    cnameVerified: boolean("cname_verified").default(false).notNull(),
    targetCname: varchar("target_cname", { length: 256 }).notNull(),
    lastCheckedAt: bigint("last_checked_at", { mode: "number" }),
    checkAttempts: int("check_attempts").default(0).notNull(),
    verificationError: varchar("verification_error", { length: 512 }),
    invocationId: varchar("invocation_id", { length: 256 }),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("custom_domains_id_unique").on(table.id),
    uniqueIndex("custom_domains_target_cname_unique").on(table.targetCname),
    uniqueIndex("unique_domain_workspace_idx").on(table.workspaceId, table.domain),
    index("project_idx").on(table.projectId),
    index("verification_status_idx").on(table.verificationStatus),
  ],
);

export const deploymentSteps = mysqlTable(
  "deployment_steps",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 128 }).notNull(),
    projectId: varchar("project_id", { length: 128 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 128 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    step: mysqlEnum(["queued", "starting", "building", "deploying", "network", "finalizing"])
      .default("queued")
      .notNull(),
    startedAt: bigint("started_at", { unsigned: true, mode: "number" }).notNull(),
    endedAt: bigint("ended_at", { unsigned: true, mode: "number" }),
    error: varchar({ length: 512 }),
  },
  (table) => [
    uniqueIndex("unique_step_per_deployment").on(table.deploymentId, table.step),
    index("workspace_idx").on(table.workspaceId),
  ],
);

export const deploymentTopology = mysqlTable(
  "deployment_topology",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 64 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 64 }).notNull(),
    regionId: varchar("region_id", { length: 64 }).notNull(),
    autoscalingReplicasMin: int("autoscaling_replicas_min", { unsigned: true })
      .default(1)
      .notNull(),
    autoscalingReplicasMax: int("autoscaling_replicas_max", { unsigned: true })
      .default(1)
      .notNull(),
    autoscalingThresholdCpu: tinyint("autoscaling_threshold_cpu", { unsigned: true }),
    autoscalingThresholdMemory: tinyint("autoscaling_threshold_memory", { unsigned: true }),
    version: bigint({ unsigned: true, mode: "number" }).notNull(),
    desiredStatus: mysqlEnum("desired_status", ["stopped", "running"]).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("unique_region_per_deployment").on(table.deploymentId, table.regionId),
    uniqueIndex("unique_version_per_region").on(table.regionId, table.version),
    index("workspace_idx").on(table.workspaceId),
    index("status_idx").on(table.desiredStatus),
  ],
);

export const deployments = mysqlTable(
  "deployments",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 128 }).notNull(),
    k8sName: varchar("k8s_name", { length: 255 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    image: varchar({ length: 256 }),
    buildId: varchar("build_id", { length: 128 }),
    gitCommitSha: varchar("git_commit_sha", { length: 40 }),
    gitBranch: varchar("git_branch", { length: 256 }),
    gitCommitMessage: text("git_commit_message"),
    gitCommitAuthorHandle: varchar("git_commit_author_handle", { length: 256 }),
    gitCommitAuthorAvatarUrl: varchar("git_commit_author_avatar_url", { length: 512 }),
    gitCommitTimestamp: bigint("git_commit_timestamp", { mode: "number" }),
    sentinelConfig: longblob("sentinel_config").notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    desiredState: mysqlEnum("desired_state", ["running", "standby", "archived"])
      .default("running")
      .notNull(),
    encryptedEnvironmentVariables: longblob("encrypted_environment_variables").notNull(),
    command: json().default(sql`_latin1'[]'`).notNull(),
    port: int().default(8080).notNull(),
    shutdownSignal: mysqlEnum("shutdown_signal", ["SIGTERM", "SIGINT", "SIGQUIT", "SIGKILL"])
      .default("SIGTERM")
      .notNull(),
    healthcheck: json(),
    prNumber: bigint("pr_number", { mode: "number" }),
    forkRepositoryFullName: varchar("fork_repository_full_name", { length: 256 }),
    githubDeploymentId: bigint("github_deployment_id", { mode: "number" }),
    status: mysqlEnum([
      "pending",
      "starting",
      "building",
      "deploying",
      "network",
      "finalizing",
      "ready",
      "failed",
      "skipped",
      "awaiting_approval",
      "stopped",
    ])
      .default("pending")
      .notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("deployments_id_unique").on(table.id),
    uniqueIndex("deployments_k8s_name_unique").on(table.k8sName),
    uniqueIndex("deployments_build_id_unique").on(table.buildId),
    index("workspace_idx").on(table.workspaceId),
    index("project_idx").on(table.projectId),
    index("status_idx").on(table.status),
  ],
);

export const environments = mysqlTable(
  "environments",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 128 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    slug: varchar({ length: 256 }).notNull(),
    description: varchar({ length: 255 }).default("").notNull(),
    deleteProtection: boolean("delete_protection").default(false),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("environments_id_unique").on(table.id),
    uniqueIndex("environments_app_slug_idx").on(table.appId, table.slug),
    index("environments_project_idx").on(table.projectId),
  ],
);

export const frontlineRoutes = mysqlTable(
  "frontline_routes",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 128 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    fullyQualifiedDomainName: varchar("fully_qualified_domain_name", { length: 256 }).notNull(),
    sticky: mysqlEnum(["none", "branch", "environment", "live", "deployment"])
      .default("none")
      .notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("frontline_routes_id_unique").on(table.id),
    uniqueIndex("frontline_routes_fully_qualified_domain_name_unique").on(
      table.fullyQualifiedDomainName,
    ),
    index("environment_id_idx").on(table.environmentId),
    index("deployment_id_idx").on(table.deploymentId),
    index("fqdn_environment_deployment_idx").on(
      table.fullyQualifiedDomainName,
      table.environmentId,
      table.deploymentId,
    ),
  ],
);

export const githubRepoConnections = mysqlTable(
  "github_repo_connections",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    projectId: varchar("project_id", { length: 64 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    installationId: bigint("installation_id", { mode: "number" }).notNull(),
    repositoryId: bigint("repository_id", { mode: "number" }).notNull(),
    repositoryFullName: varchar("repository_full_name", { length: 500 }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("github_repo_connections_app_id_unique").on(table.appId),
    index("installation_id_idx").on(table.installationId),
  ],
);

export const horizontalAutoscalingPolicies = mysqlTable(
  "horizontal_autoscaling_policies",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 64 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    replicasMin: int("replicas_min").notNull(),
    replicasMax: int("replicas_max").notNull(),
    memoryThreshold: tinyint("memory_threshold"),
    cpuThreshold: tinyint("cpu_threshold"),
    rpsThreshold: tinyint("rps_threshold"),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("horizontal_autoscaling_policies_id_unique").on(table.id),
    index("workspace_idx").on(table.workspaceId),
  ],
);

export const instances = mysqlTable(
  "instances",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 64 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    regionId: varchar("region_id", { length: 64 }).notNull(),
    k8sName: varchar("k8s_name", { length: 255 }).notNull(),
    address: varchar({ length: 255 }).notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    status: mysqlEnum(["inactive", "pending", "running", "failed"]).notNull(),
  },
  (table) => [
    uniqueIndex("instances_id_unique").on(table.id),
    uniqueIndex("unique_address_per_region").on(table.address, table.regionId),
    uniqueIndex("unique_k8s_name_per_region").on(table.k8sName, table.regionId),
    index("idx_deployment_id").on(table.deploymentId),
    index("idx_region").on(table.regionId),
  ],
);

export const keys = mysqlTable(
  "keys",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    keyAuthId: varchar("key_auth_id", { length: 256 }).notNull(),
    hash: varchar({ length: 256 }).notNull(),
    start: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    forWorkspaceId: varchar("for_workspace_id", { length: 256 }),
    name: varchar({ length: 256 }),
    ownerId: varchar("owner_id", { length: 256 }),
    identityId: varchar("identity_id", { length: 256 }),
    meta: text(),
    expires: datetime({ fsp: 3 }),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
    deletedAtM: bigint("deleted_at_m", { mode: "number" }),
    refillDay: tinyint("refill_day"),
    refillAmount: int("refill_amount"),
    lastRefillAt: datetime("last_refill_at", { fsp: 3 }),
    enabled: boolean().default(true).notNull(),
    remainingRequests: int("remaining_requests"),
    environment: varchar({ length: 256 }),
    lastUsedAt: bigint("last_used_at", { unsigned: true, mode: "number" }).default(0).notNull(),
    pendingMigrationId: varchar("pending_migration_id", { length: 256 }),
  },
  (table) => [
    uniqueIndex("keys_id_unique").on(table.id),
    uniqueIndex("hash_idx").on(table.hash),
    index("key_auth_id_deleted_at_idx").on(table.keyAuthId, table.deletedAtM, table.id),
    index("idx_keys_on_for_workspace_id").on(table.forWorkspaceId),
    index("pending_migration_id_idx").on(table.pendingMigrationId),
    index("idx_keys_on_workspace_id").on(table.workspaceId),
    index("owner_id_idx").on(table.ownerId),
    index("identity_id_idx").on(table.identityId, table.keyAuthId, table.id),
    index("idx_keys_refill").on(table.refillAmount, table.deletedAtM),
  ],
);

export const regions = mysqlTable(
  "regions",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 64 }).notNull(),
    name: varchar({ length: 64 }).notNull(),
    platform: varchar({ length: 64 }).notNull(),
    canSchedule: boolean("can_schedule").default(true).notNull(),
  },
  (table) => [
    uniqueIndex("regions_id_unique").on(table.id),
    uniqueIndex("unique_region_per_platform").on(table.name, table.platform),
  ],
);

export const roles = mysqlTable(
  "roles",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    name: varchar({ length: 512 }).notNull(),
    description: varchar({ length: 512 }),
    createdAtM: bigint("created_at_m", { mode: "number" }).default(0).notNull(),
    updatedAtM: bigint("updated_at_m", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("roles_id_unique").on(table.id),
    uniqueIndex("unique_name_per_workspace_idx").on(table.name, table.workspaceId),
    index("workspace_id_idx").on(table.workspaceId),
  ],
);

export const sentinels = mysqlTable(
  "sentinels",
  {
    pk: bigint({ unsigned: true, mode: "number" }).autoincrement().primaryKey(),
    id: varchar({ length: 64 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    k8sName: varchar("k8s_name", { length: 64 }).notNull(),
    k8sAddress: varchar("k8s_address", { length: 255 }).notNull(),
    regionId: varchar("region_id", { length: 255 }).notNull(),
    image: varchar({ length: 255 }).notNull(),
    desiredState: mysqlEnum("desired_state", ["running", "standby", "archived"])
      .default("running")
      .notNull(),
    health: mysqlEnum(["unknown", "paused", "healthy", "unhealthy"]).default("unknown").notNull(),
    desiredReplicas: int("desired_replicas").notNull(),
    availableReplicas: int("available_replicas").notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    version: bigint({ unsigned: true, mode: "number" }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("sentinels_id_unique").on(table.id),
    uniqueIndex("sentinels_k8s_name_unique").on(table.k8sName),
    uniqueIndex("sentinels_k8s_address_unique").on(table.k8sAddress),
    uniqueIndex("one_env_per_region").on(table.environmentId, table.regionId),
    uniqueIndex("unique_version_per_region").on(table.regionId, table.version),
    index("idx_environment_health_region_routing").on(
      table.environmentId,
      table.regionId,
      table.health,
    ),
  ],
);
