export const ACTIONS = [
  "read_project",
  "write_project",
  "delete_project",
  "read_app",
  "write_app",
  "delete_app",
  "read_environment",
  "write_environment",
  "delete_environment",
  "read_deployment",
  "write_deployment",
  "delete_deployment",
  "read_deployment_logs",
  "read_domain",
  "write_domain",
  "delete_domain",
  "read_environment_variable",
  "write_environment_variable",
  "delete_environment_variable",
  "read_gateway_logs",
  "read_gateway_policy",
  "write_gateway_policy",
  "delete_gateway_policy",
  "read_identity",
  "write_identity",
  "delete_identity",
  "read_keyspace",
  "write_keyspace",
  "delete_keyspace",
  "read_keyspace_logs",
  "read_key",
  "write_key",
  "delete_key",
  "decrypt_key",
  "verify_key",
  "read_ratelimit_namespace",
  "write_ratelimit_namespace",
  "delete_ratelimit_namespace",
  "limit_ratelimit_namespace",
  "read_ratelimit_logs",
  "read_ratelimit_override",
  "write_ratelimit_override",
  "delete_ratelimit_override",
  "read_role",
  "write_role",
  "delete_role",
  "read_permission",
  "write_permission",
  "delete_permission",
  "read_github_app",
  "write_github_app",
  "delete_github_app",
] as const;

export type Action = (typeof ACTIONS)[number];

export const ACTION_LABELS = {
  read_project: "Read",
  write_project: "Write",
  delete_project: "Delete",
  read_app: "Read",
  write_app: "Write",
  delete_app: "Delete",
  read_environment: "Read",
  write_environment: "Write",
  delete_environment: "Delete",
  read_deployment: "Read",
  write_deployment: "Write",
  delete_deployment: "Delete",
  read_deployment_logs: "Read",
  read_domain: "Read",
  write_domain: "Write",
  delete_domain: "Delete",
  read_environment_variable: "Read",
  write_environment_variable: "Write",
  delete_environment_variable: "Delete",
  read_gateway_logs: "Read",
  read_gateway_policy: "Read",
  write_gateway_policy: "Write",
  delete_gateway_policy: "Delete",
  read_identity: "Read",
  write_identity: "Write",
  delete_identity: "Delete",
  read_keyspace: "Read",
  write_keyspace: "Write",
  delete_keyspace: "Delete",
  read_keyspace_logs: "Read",
  read_key: "Read",
  write_key: "Write",
  delete_key: "Delete",
  decrypt_key: "Decrypt",
  verify_key: "Verify",
  read_ratelimit_namespace: "Read",
  write_ratelimit_namespace: "Write",
  delete_ratelimit_namespace: "Delete",
  limit_ratelimit_namespace: "Limit",
  read_ratelimit_logs: "Read",
  read_ratelimit_override: "Read",
  write_ratelimit_override: "Write",
  delete_ratelimit_override: "Delete",
  read_role: "Read",
  write_role: "Write",
  delete_role: "Delete",
  read_permission: "Read",
  write_permission: "Write",
  delete_permission: "Delete",
  read_github_app: "Read",
  write_github_app: "Write",
  delete_github_app: "Delete",
} satisfies Record<Action, string>;

export const WORKOS_PERMISSION_SLUGS = [
  "admin:*",
  "apps:delete",
  "apps:read",
  "apps:write",
  "deployment_logs:read",
  "deployments:delete",
  "deployments:read",
  "deployments:write",
  "domains:delete",
  "domains:read",
  "domains:write",
  "environment_variables:delete",
  "environment_variables:read",
  "environment_variables:write",
  "environments:delete",
  "environments:read",
  "environments:write",
  "gateway_logs:read",
  "gateway_policies:delete",
  "gateway_policies:read",
  "gateway_policies:write",
  "github_apps:delete",
  "github_apps:read",
  "github_apps:write",
  "identities:delete",
  "identities:read",
  "identities:write",
  "keys:decrypt",
  "keys:delete",
  "keys:read",
  "keys:verify",
  "keys:write",
  "keyspace_logs:read",
  "keyspaces:delete",
  "keyspaces:read",
  "keyspaces:write",
  "permissions:delete",
  "permissions:read",
  "permissions:write",
  "projects:delete",
  "projects:read",
  "projects:write",
  "ratelimit_logs:read",
  "ratelimit_namespaces:delete",
  "ratelimit_namespaces:limit",
  "ratelimit_namespaces:read",
  "ratelimit_namespaces:write",
  "ratelimit_overrides:delete",
  "ratelimit_overrides:read",
  "ratelimit_overrides:write",
  "roles:delete",
  "roles:read",
  "roles:write",
] as const;

export type WorkOSPermissionSlug = (typeof WORKOS_PERMISSION_SLUGS)[number];
export type GrantWorkOSPermissionSlug = Exclude<WorkOSPermissionSlug, "admin:*">;

export const RESOURCE_SCOPES = [
  "workspace",
  "projects",
  "apps",
  "environments",
  "keyspaces",
  "ratelimit-namespaces",
  "identities",
  "rbac",
] as const;

export type ResourceScope = (typeof RESOURCE_SCOPES)[number];

export const INSTANCE_TOKEN = "{instance}";

export type ActionGrant = {
  name: Action;
  slug: GrantWorkOSPermissionSlug;
  path?: string;
};

export function actionGrant(name: Action, slug: GrantWorkOSPermissionSlug): readonly ActionGrant[] {
  return [{ name, slug }];
}

export type PermissionRow = {
  id: string;
  label: string;
  path: string;
  allPath?: string;
  resource: string;
  actions?: Partial<Record<Action, readonly ActionGrant[]>>;
};

export type CatalogueGroup = {
  id: string;
  label: string;
  rows: PermissionRow[];
};

export type ScopeCatalogue = {
  scope: ResourceScope;
  label: string;
  allLabel: string;
  instanceNoun: string | null;
  groups: CatalogueGroup[];
};

export type PermissionSelection = Record<string, Action[]>;
