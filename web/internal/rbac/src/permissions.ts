/**
 * The database takes care of isolating roles between workspaces.
 *
 * role identifiers can look like this:
 * - `api_id.xxx`
 *
 */
import { z } from "zod";
import type { Flatten } from "./types";
export function buildIdSchema(prefix: string) {
  return z.string().refine((s) => {
    if (s === "*") {
      return true;
    }
    const regex = new RegExp(`^${prefix}_[0-9A-Za-z]{8,32}$`);
    return regex.test(s);
  });
}
const apiId = buildIdSchema("api");
const ratelimitNamespaceId = buildIdSchema("rlns");
const rbacId = buildIdSchema("rbac");
const identityEnvId = z.string();
const projectId = buildIdSchema("proj");
const appId = buildIdSchema("app");
const environmentId = buildIdSchema("env");
const workspaceId = buildIdSchema("ws");
const portalId = buildIdSchema("pc");
export const apiActions = z.enum([
  "read_api",
  "create_api",
  "delete_api",
  "update_api",
  "create_key",
  "update_key",
  "delete_key",
  "encrypt_key",
  "decrypt_key",
  "read_key",
  "verify_key",
  "read_analytics",
]);
export const ratelimitActions = z.enum([
  "limit",
  "read_analytics",
  "create_namespace",
  "read_namespace",
  "update_namespace",
  "delete_namespace",
  "set_override",
  "read_override",
  "delete_override",
]);
export const rbacActions = z.enum([
  "create_permission",
  "update_permission",
  "delete_permission",
  "read_permission",
  "create_role",
  "update_role",
  "delete_role",
  "read_role",
  "add_permission_to_key",
  "remove_permission_from_key",
  "add_role_to_key",
  "remove_role_from_key",
  "add_permission_to_role",
  "remove_permission_from_role",
]);
export const identityActions = z.enum([
  "create_identity",
  "read_identity",
  "update_identity",
  "delete_identity",
]);
export const projectActions = z.enum([
  "create_project",
  "read_project",
  "update_project",
  "delete_project",
  "create_app",
  "create_deployment",
  "read_deployment",
  "generate_upload_url",
  "read_gateway_requests",
  "read_runtime_logs",
]);
export const appActions = z.enum(["read_app", "update_app", "delete_app", "connect_repository"]);
export const workspaceActions = z.enum(["install_github"]);
export const portalActions = z.enum([
  "create_portal",
  "read_portal",
  "update_portal",
  "delete_portal",
  "create_portal_session",
]);
export const environmentActions = z.enum([
  "read_environment",
  "update_environment",
  "set_environment_variables",
  "remove_environment_variables",
  "read_environment_variables",
  "create_deployment",
  "read_deployment",
  "stop_deployment",
  "start_deployment",
  "promote_deployment",
  "rollback_deployment",
  "set_policies",
  "update_policy",
  "read_policies",
  "create_domain",
  "read_domain",
  "delete_domain",
  "verify_domain",
]);

// Resources that require an ID (resource.id.action format)
const scopedResources = {
  api: { idSchema: apiId, actionsSchema: apiActions },
  ratelimit: { idSchema: ratelimitNamespaceId, actionsSchema: ratelimitActions },
  rbac: { idSchema: rbacId, actionsSchema: rbacActions },
  identity: { idSchema: identityEnvId, actionsSchema: identityActions },
  project: { idSchema: projectId, actionsSchema: projectActions },
  app: { idSchema: appId, actionsSchema: appActions },
  environment: { idSchema: environmentId, actionsSchema: environmentActions },
  workspace: { idSchema: workspaceId, actionsSchema: workspaceActions },
  portal: { idSchema: portalId, actionsSchema: portalActions },
} as const;

export type Resources = {
  [resourceId in `api.${z.infer<typeof apiId>}`]: z.infer<typeof apiActions>;
} & {
  [resourceId in `ratelimit.${z.infer<typeof ratelimitNamespaceId>}`]: z.infer<
    typeof ratelimitActions
  >;
} & {
  [resourceId in `rbac.${z.infer<typeof rbacId>}`]: z.infer<typeof rbacActions>;
} & {
  [resourceId in `identity.${z.infer<typeof identityEnvId>}`]: z.infer<typeof identityActions>;
} & {
  [resourceId in `project.${z.infer<typeof projectId>}`]: z.infer<typeof projectActions>;
} & {
  [resourceId in `app.${z.infer<typeof appId>}`]: z.infer<typeof appActions>;
} & {
  [resourceId in `environment.${z.infer<typeof environmentId>}`]: z.infer<
    typeof environmentActions
  >;
} & {
  [resourceId in `workspace.${z.infer<typeof workspaceId>}`]: z.infer<typeof workspaceActions>;
} & {
  [resourceId in `portal.${z.infer<typeof portalId>}`]: z.infer<typeof portalActions>;
};

export type UnkeyPermission = Flatten<Resources> | "*";

export type UnkeyUrnPermission = `unkey:v1:${string}:${string}#${string}`;

export const PERMISSION_MAX_LENGTH = 512;

// The URN vocabulary: every resource path a root key can name, the action it
// carries there, and the WorkOS slug that stands for the same grant on a user.
// A URN permission is only meaningful if it appears here, so this list — not
// the grammar — decides what the write path accepts.
//
// Mutations collapse into `write_<resource>`: create and update are the same
// privilege, and a deployment start, stop, promote or rollback is a write of
// that deployment. Delete stays apart because it destroys, and keys keep
// `verify` and `decrypt` because both are narrower than reading a key.
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

export type WorkosPermissionSlug = (typeof workosPermissionDefinitions)[number]["slug"];

export type CatalogAction = (typeof workosPermissionDefinitions)[number]["action"];

function legacyPermissionError(value: string): string | null {
  if (value === "*") {
    return null;
  }
  const parts = value.split(".");
  if (parts.length !== 3) {
    return 'Permission must be a "unkey:v1:<workspace_id>:<resource_path>#<action>" URN or a legacy "resource.id.action" tuple.';
  }
  const [resource, id, action] = parts;
  const resourceConfig = scopedResources[resource as keyof typeof scopedResources];
  if (!resourceConfig) {
    return `Unknown resource "${resource}". Expected one of: ${Object.keys(scopedResources).join(", ")}.`;
  }
  if (!resourceConfig.idSchema.safeParse(id).success) {
    return `Invalid id "${id}" for resource "${resource}". Expected "*" or an id of the resource.`;
  }
  if (!resourceConfig.actionsSchema.safeParse(action).success) {
    return `Unknown action "${action}" for resource "${resource}". Expected one of: ${resourceConfig.actionsSchema.options.join(", ")}.`;
  }
  return null;
}

/**
 * Validation for roles used for our root keys
 */
export const unkeyPermissionValidation = z
  .custom<UnkeyPermission>()
  .refine((s) => typeof s === "string" && legacyPermissionError(s) === null);

const urnPrefix = "unkey";
const urnVersion = "v1";
const urnPermissionPrefix = `${urnPrefix}:${urnVersion}:`;
const globalResourcePath = "**";

type UrnPermissionParseResult =
  | { ok: true; workspaceId: string; resourcePath: string; action: string }
  | { ok: false; error: string };

// validateWorkspaceId, validateResourcePath and validatePermissionAction mirror
// the helpers of the same name in pkg/urn/urn.go and pkg/rbac/urn.go. The
// reserved characters below cannot survive the field split, but the rule is
// stated so both grammars read the same.
function validateWorkspaceId(value: string): string | null {
  if (value.length === 0) {
    return "Workspace id must not be empty.";
  }
  if (/[:#/]/.test(value)) {
    return 'Workspace id must not contain ":", "#" or "/".';
  }
  return null;
}

function validateResourcePath(resourcePath: string): string | null {
  const segments = resourcePath.split("/");
  for (const [index, segment] of segments.entries()) {
    if (segment.length === 0) {
      return "Resource path must not contain empty segments.";
    }
    if (/[:#]/.test(segment)) {
      return 'Resource path segments must not contain ":" or "#".';
    }
    if (segment === "*") {
      continue;
    }
    if (segment === globalResourcePath) {
      if (index !== segments.length - 1) {
        return '"**" must be the last resource path segment.';
      }
      continue;
    }
    if (segment.includes("*")) {
      return '"*" must be a whole resource path segment.';
    }
  }
  return null;
}

function validatePermissionAction(action: string): string | null {
  if (action === "*") {
    return null;
  }
  if (action.length === 0) {
    return "Action must not be empty.";
  }
  if (/[:#/*]/.test(action)) {
    return 'Action must not contain ":", "#", "/" or "*".';
  }
  if (action.startsWith("_") || action.endsWith("_")) {
    return 'Action must not start or end with "_".';
  }
  return null;
}

function parseUrnPermission(value: string): UrnPermissionParseResult {
  const [resource, action, ...extra] = value.split("#");
  if (action === undefined || extra.length > 0) {
    return { ok: false, error: 'Permission must contain exactly one "#" action separator.' };
  }

  const fields = resource.split(":");
  if (fields.length < 4) {
    return {
      ok: false,
      error: 'Permission URN must read "unkey:v1:<workspace_id>:<resource_path>".',
    };
  }
  const [prefix, version, urnWorkspaceId] = fields;
  const resourcePath = fields.slice(3).join(":");

  if (prefix !== urnPrefix) {
    return { ok: false, error: `Permission URN prefix must be "${urnPrefix}".` };
  }
  if (version !== urnVersion) {
    return { ok: false, error: `Permission URN version must be "${urnVersion}".` };
  }

  const workspaceError = validateWorkspaceId(urnWorkspaceId);
  if (workspaceError) {
    return { ok: false, error: workspaceError };
  }
  const resourcePathError = validateResourcePath(resourcePath);
  if (resourcePathError) {
    return { ok: false, error: resourcePathError };
  }
  const actionError = validatePermissionAction(action);
  if (actionError) {
    return { ok: false, error: actionError };
  }
  if (action === "*" && resourcePath !== globalResourcePath) {
    return {
      ok: false,
      error: `Action "*" requires the global resource path "${globalResourcePath}".`,
    };
  }

  return { ok: true, workspaceId: urnWorkspaceId, resourcePath, action };
}

const isParameterSegment = (segment: string): boolean =>
  segment.startsWith("{") && segment.endsWith("}");

// A catalog path matches a resource path segment for segment. Literal segments
// must be spelled out; parameter segments take an id or "*". Wildcards are
// prefix-closed: once a parent id is "*" every id below it must be "*" too,
// because "the app in every project" names no app anyone can point at.
function catalogPathMatches(pattern: string, resourcePath: string): boolean {
  if (pattern === globalResourcePath) {
    return resourcePath === globalResourcePath;
  }
  const patternSegments = pattern.split("/");
  const resourceSegments = resourcePath.split("/");
  if (patternSegments.length !== resourceSegments.length) {
    return false;
  }
  let wildcardSeen = false;
  return patternSegments.every((patternSegment, index) => {
    const resourceSegment = resourceSegments[index];
    if (resourceSegment === undefined || resourceSegment === globalResourcePath) {
      return false;
    }
    if (!isParameterSegment(patternSegment)) {
      return patternSegment === resourceSegment;
    }
    if (resourceSegment === "*") {
      wildcardSeen = true;
      return true;
    }
    return !wildcardSeen;
  });
}

export function catalogPermissionError(resourcePath: string, action: string): string | null {
  const known = workosPermissionDefinitions.some(
    (definition) =>
      definition.action === action && catalogPathMatches(definition.path, resourcePath),
  );
  return known
    ? null
    : `"${action}" is not an action on "${resourcePath}". See the permission catalog in @unkey/rbac.`;
}

export function isCatalogPermission(value: string): boolean {
  const result = parseUrnPermission(value);
  return result.ok && catalogPermissionError(result.resourcePath, result.action) === null;
}

// Grammar first, then vocabulary: a URN can be well formed and still name a
// privilege that does not exist.
function urnPermissionError(value: string): string | null {
  const result = parseUrnPermission(value);
  return result.ok ? catalogPermissionError(result.resourcePath, result.action) : result.error;
}

export function urnPermissionWorkspaceId(value: string): string | null {
  const result = parseUrnPermission(value);
  return result.ok ? result.workspaceId : null;
}

// Branching on the URN prefix rather than unioning two schemas keeps the error
// specific to the grammar the caller was reaching for.
export const permissionValidation = z
  .string()
  .max(PERMISSION_MAX_LENGTH, {
    error: `Permission must be at most ${PERMISSION_MAX_LENGTH} characters.`,
  })
  .superRefine((value, ctx) => {
    const error = value.startsWith(urnPermissionPrefix)
      ? urnPermissionError(value)
      : legacyPermissionError(value);
    if (error) {
      ctx.addIssue({ code: "custom", message: error });
    }
  });
