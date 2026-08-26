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

const legacyWildcard = "*";

function legacyPermissionError(value: string): string | null {
  if (value === legacyWildcard) {
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
const urnFieldSeparator = ":";
const pathSeparator = "/";
const actionSeparator = "#";
const globalResourcePath = "**";
const segmentWildcard = "*";
const actionWildcard = "*";

type UrnPermissionParts = {
  workspaceId: string;
  resourcePath: string;
  action: string;
};

type UrnPermissionParseResult =
  | { ok: true; permission: UrnPermissionParts }
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
  const segments = resourcePath.split(pathSeparator);
  for (const [index, segment] of segments.entries()) {
    if (segment.length === 0) {
      return "Resource path must not contain empty segments.";
    }
    if (/[:#]/.test(segment)) {
      return 'Resource path segments must not contain ":" or "#".';
    }
    if (segment === segmentWildcard) {
      continue;
    }
    if (segment === globalResourcePath) {
      if (index !== segments.length - 1) {
        return '"**" must be the last resource path segment.';
      }
      continue;
    }
    if (segment.includes(segmentWildcard)) {
      return '"*" must be a whole resource path segment.';
    }
  }
  return null;
}

function validatePermissionAction(action: string): string | null {
  if (action === actionWildcard) {
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
  const [resource, action, ...extra] = value.split(actionSeparator);
  if (action === undefined || extra.length > 0) {
    return { ok: false, error: 'Permission must contain exactly one "#" action separator.' };
  }

  const fields = resource.split(urnFieldSeparator);
  if (fields.length < 4) {
    return {
      ok: false,
      error: 'Permission URN must read "unkey:v1:<workspace_id>:<resource_path>".',
    };
  }
  const [prefix, version, urnWorkspaceId] = fields;
  const resourcePath = fields.slice(3).join(urnFieldSeparator);

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
  if (action === actionWildcard && resourcePath !== globalResourcePath) {
    return {
      ok: false,
      error: `Action "*" requires the global resource path "${globalResourcePath}".`,
    };
  }

  return { ok: true, permission: { workspaceId: urnWorkspaceId, resourcePath, action } };
}

function urnPermissionError(value: string): string | null {
  const result = parseUrnPermission(value);
  return result.ok ? null : result.error;
}

export const unkeyUrnPermissionValidation = z
  .custom<UnkeyUrnPermission>()
  .refine((s) => typeof s === "string" && parseUrnPermission(s).ok);

export function urnPermissionWorkspaceId(value: string): string | null {
  const result = parseUrnPermission(value);
  return result.ok ? result.permission.workspaceId : null;
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
