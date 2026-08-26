import { z } from "zod";

export function buildIdSchema(prefix: string) {
  return z.string().refine((s) => {
    if (s === "*") {
      return true;
    }
    const regex = new RegExp(`^${prefix}_[0-9A-Za-z]{8,32}$`);
    return regex.test(s);
  });
}

const workspaceId = buildIdSchema("ws");

export const workosPermissionDefinitions = [
  { slug: "admin:*", path: "**", action: "*" },
  { slug: "apps:delete", path: "projects/{project_id}/apps/{app_id}", action: "delete_app" },
  { slug: "apps:read", path: "projects/{project_id}/apps/{app_id}", action: "read_app" },
  { slug: "apps:write", path: "projects/{project_id}/apps/{app_id}", action: "write_app" },
  {
    slug: "deployment_logs:read",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/deployments/{deployment_id}/logs",
    action: "read_deployment_logs",
  },
  {
    slug: "deployments:delete",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/deployments/{deployment_id}",
    action: "delete_deployment",
  },
  {
    slug: "deployments:read",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/deployments/{deployment_id}",
    action: "read_deployment",
  },
  {
    slug: "deployments:start",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/deployments/{deployment_id}",
    action: "start_deployment",
  },
  {
    slug: "deployments:stop",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/deployments/{deployment_id}",
    action: "stop_deployment",
  },
  {
    slug: "deployments:write",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/deployments/{deployment_id}",
    action: "write_deployment",
  },
  {
    slug: "domains:delete",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/domains/{domain_id}",
    action: "delete_domain",
  },
  {
    slug: "domains:read",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/domains/{domain_id}",
    action: "read_domain",
  },
  {
    slug: "domains:write",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/domains/{domain_id}",
    action: "write_domain",
  },
  {
    slug: "environment_variables:delete",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/variables/{variable_id}",
    action: "delete_environment_variable",
  },
  {
    slug: "environment_variables:read",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/variables/{variable_id}",
    action: "read_environment_variable",
  },
  {
    slug: "environment_variables:write",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/variables/{variable_id}",
    action: "write_environment_variable",
  },
  {
    slug: "environments:delete",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}",
    action: "delete_environment",
  },
  {
    slug: "environments:read",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}",
    action: "read_environment",
  },
  {
    slug: "environments:write",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}",
    action: "write_environment",
  },
  {
    slug: "gateway_logs:read",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/gateway/logs",
    action: "read_gateway_logs",
  },
  {
    slug: "gateway_policies:delete",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/gateway/policies/{policy_id}",
    action: "delete_gateway_policy",
  },
  {
    slug: "gateway_policies:read",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/gateway/policies/{policy_id}",
    action: "read_gateway_policy",
  },
  {
    slug: "gateway_policies:write",
    path: "projects/{project_id}/apps/{app_id}/environments/{environment_id}/gateway/policies/{policy_id}",
    action: "write_gateway_policy",
  },
  { slug: "github_apps:delete", path: "github/apps/{github_app_id}", action: "delete_github_app" },
  { slug: "github_apps:read", path: "github/apps/{github_app_id}", action: "read_github_app" },
  { slug: "github_apps:write", path: "github/apps/{github_app_id}", action: "write_github_app" },
  {
    slug: "identities:delete",
    path: "projects/{project_id}/identities/{identity_id}",
    action: "delete_identity",
  },
  {
    slug: "identities:read",
    path: "projects/{project_id}/identities/{identity_id}",
    action: "read_identity",
  },
  {
    slug: "identities:write",
    path: "projects/{project_id}/identities/{identity_id}",
    action: "write_identity",
  },
  {
    slug: "keys:decrypt",
    path: "projects/{project_id}/keyspaces/{keyspace_id}/keys/{key_id}",
    action: "decrypt_key",
  },
  {
    slug: "keys:delete",
    path: "projects/{project_id}/keyspaces/{keyspace_id}/keys/{key_id}",
    action: "delete_key",
  },
  {
    slug: "keys:read",
    path: "projects/{project_id}/keyspaces/{keyspace_id}/keys/{key_id}",
    action: "read_key",
  },
  {
    slug: "keys:verify",
    path: "projects/{project_id}/keyspaces/{keyspace_id}/keys/{key_id}",
    action: "verify_key",
  },
  {
    slug: "keys:write",
    path: "projects/{project_id}/keyspaces/{keyspace_id}/keys/{key_id}",
    action: "write_key",
  },
  {
    slug: "keyspace_logs:read",
    path: "projects/{project_id}/keyspaces/{keyspace_id}/logs",
    action: "read_keyspace_logs",
  },
  {
    slug: "keyspaces:delete",
    path: "projects/{project_id}/keyspaces/{keyspace_id}",
    action: "delete_keyspace",
  },
  {
    slug: "keyspaces:read",
    path: "projects/{project_id}/keyspaces/{keyspace_id}",
    action: "read_keyspace",
  },
  {
    slug: "keyspaces:write",
    path: "projects/{project_id}/keyspaces/{keyspace_id}",
    action: "write_keyspace",
  },
  {
    slug: "permissions:delete",
    path: "projects/{project_id}/rbac/permissions/{permission_id}",
    action: "delete_permission",
  },
  {
    slug: "permissions:read",
    path: "projects/{project_id}/rbac/permissions/{permission_id}",
    action: "read_permission",
  },
  {
    slug: "permissions:write",
    path: "projects/{project_id}/rbac/permissions/{permission_id}",
    action: "write_permission",
  },
  { slug: "projects:delete", path: "projects/{project_id}", action: "delete_project" },
  { slug: "projects:read", path: "projects/{project_id}", action: "read_project" },
  { slug: "projects:write", path: "projects/{project_id}", action: "write_project" },
  {
    slug: "ratelimit_logs:read",
    path: "projects/{project_id}/ratelimits/namespaces/{namespace_id}/logs",
    action: "read_ratelimit_logs",
  },
  {
    slug: "ratelimit_namespaces:delete",
    path: "projects/{project_id}/ratelimits/namespaces/{namespace_id}",
    action: "delete_ratelimit_namespace",
  },
  {
    slug: "ratelimit_namespaces:limit",
    path: "projects/{project_id}/ratelimits/namespaces/{namespace_id}",
    action: "limit_ratelimit_namespace",
  },
  {
    slug: "ratelimit_namespaces:read",
    path: "projects/{project_id}/ratelimits/namespaces/{namespace_id}",
    action: "read_ratelimit_namespace",
  },
  {
    slug: "ratelimit_namespaces:write",
    path: "projects/{project_id}/ratelimits/namespaces/{namespace_id}",
    action: "write_ratelimit_namespace",
  },
  {
    slug: "ratelimit_overrides:delete",
    path: "projects/{project_id}/ratelimits/namespaces/{namespace_id}/overrides/{override_id}",
    action: "delete_ratelimit_override",
  },
  {
    slug: "ratelimit_overrides:read",
    path: "projects/{project_id}/ratelimits/namespaces/{namespace_id}/overrides/{override_id}",
    action: "read_ratelimit_override",
  },
  {
    slug: "ratelimit_overrides:write",
    path: "projects/{project_id}/ratelimits/namespaces/{namespace_id}/overrides/{override_id}",
    action: "write_ratelimit_override",
  },
  {
    slug: "roles:delete",
    path: "projects/{project_id}/rbac/roles/{role_id}",
    action: "delete_role",
  },
  { slug: "roles:read", path: "projects/{project_id}/rbac/roles/{role_id}", action: "read_role" },
  {
    slug: "roles:write",
    path: "projects/{project_id}/rbac/roles/{role_id}",
    action: "write_role",
  },
] as const;

type CatalogAction = (typeof workosPermissionDefinitions)[number]["action"];

type CatalogEntry = {
  path: string;
  action: CatalogAction;
};

const catalog = workosPermissionDefinitions satisfies readonly CatalogEntry[];

type CatalogPermission = `unkey:v1:${string}:${string}#${CatalogAction}`;

export type UnkeyPermission = string;

const isParameterSegment = (segment: string): boolean =>
  segment.startsWith("{") && segment.endsWith("}");

const isValidResourceSegment = (segment: string): boolean => {
  if (segment.length === 0 || segment === "**") {
    return false;
  }
  return !segment.includes("*") || segment === "*";
};

const resourceMatches = (pattern: string, resource: string): boolean => {
  if (pattern === "**") {
    return resource === "**";
  }

  const patternSegments = pattern.split("/");
  const resourceSegments = resource.split("/");
  if (patternSegments.length !== resourceSegments.length) {
    return false;
  }
  let wildcardIDSeen = false;
  return patternSegments.every((patternSegment, index) => {
    const resourceSegment = resourceSegments[index];
    if (resourceSegment === undefined || !isValidResourceSegment(resourceSegment)) {
      return false;
    }
    if (!isParameterSegment(patternSegment)) {
      return patternSegment === resourceSegment;
    }
    if (resourceSegment === "*") {
      wildcardIDSeen = true;
      return true;
    }
    return !wildcardIDSeen;
  });
};

const urnPattern = /^unkey:v1:([^:]+):([^#]+)#([^#]+)$/;

function isUnkeyPermission(value: unknown): value is CatalogPermission {
  if (typeof value !== "string") {
    return false;
  }

  const match = urnPattern.exec(value);
  if (!match) {
    return false;
  }

  const [, workspace, resource, action] = match;
  if (
    workspace === undefined ||
    resource === undefined ||
    action === undefined ||
    !workspaceId.safeParse(workspace).success
  ) {
    return false;
  }

  return catalog.some((entry) => entry.action === action && resourceMatches(entry.path, resource));
}

/**
 * Validation for roles used for our root keys.
 */
export const unkeyPermissionValidation = z.custom<UnkeyPermission>(isUnkeyPermission);
