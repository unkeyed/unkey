/**
 * The database takes care of isolating roles between workspaces.
 * That's why we can assume the highest scope of a role is an `api` or later `sentinel`
 *
 * role identifiers can look like this:
 * - `api_id.xxx`
 * - `sentinel_id.xxx`
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
const ratelimitNamespaceId = buildIdSchema("rl");
const rbacId = buildIdSchema("rbac");
const identityEnvId = z.string();
const projectId = buildIdSchema("proj");
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
  "create_deployment",
  "read_deployment",
  "generate_upload_url",
]);

// Resources that require an ID (resource.id.action format)
const scopedResources = {
  api: { idSchema: apiId, actionsSchema: apiActions },
  ratelimit: { idSchema: ratelimitNamespaceId, actionsSchema: ratelimitActions },
  rbac: { idSchema: rbacId, actionsSchema: rbacActions },
  identity: { idSchema: identityEnvId, actionsSchema: identityActions },
  project: { idSchema: projectId, actionsSchema: projectActions },
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
};

export type UnkeyResourcePermission = `unkey:v1:${string}:${string}#${string}`;
export type UnkeyPermission = Flatten<Resources> | UnkeyResourcePermission | "*";

const urnPermission = z.string().refine((value) => {
  const [resource, action, ...rest] = value.split("#");
  if (rest.length > 0 || !resource || !action) {
    return false;
  }

  const [prefix, version, workspaceId, path, ...pathRest] = resource.split(":");
  if (
    prefix !== "unkey" ||
    version !== "v1" ||
    !workspaceId ||
    !path ||
    pathRest.length > 0 ||
    workspaceId.includes("/")
  ) {
    return false;
  }

  if (path.startsWith("/") || path.endsWith("/") || path.includes("//")) {
    return false;
  }

  const segments = path.split("/");
  let seenWildcardSelector = false;
  for (const [index, segment] of segments.entries()) {
    const isLastSegment = index === segments.length - 1;
    if (segment === "") {
      return false;
    }
    if (segment.includes("*") && segment !== "*" && segment !== "**") {
      return false;
    }
    if (segment === "*") {
      seenWildcardSelector = true;
      continue;
    }
    if (segment === "**" && !isLastSegment) {
      return false;
    }
    if (segment !== "**" && seenWildcardSelector) {
      const next = segments[index + 1];
      if (isLastSegment || next !== "*") {
        return false;
      }
    }
  }

  if (action === "*") {
    return path === "**";
  }
  if (action.startsWith("_") || action.endsWith("_")) {
    return false;
  }
  return !/[#:/*]/.test(action);
});

/**
 * Validation for roles used for our root keys
 */
export const unkeyPermissionValidation = z.custom<UnkeyPermission>().refine((s) => {
  z.string().parse(s);
  if (s === "*") {
    /**
     * This is a legacy role granting access to everything
     */
    return true;
  }
  if (urnPermission.safeParse(s).success) {
    return true;
  }
  const split = s.split(".");

  // Handle scoped resource.id.action format (3 parts)
  if (split.length !== 3) {
    return false;
  }
  const [resource, id, action] = split;
  const resourceConfig = scopedResources[resource as keyof typeof scopedResources];
  if (resourceConfig) {
    return (
      resourceConfig.idSchema.safeParse(id).success &&
      resourceConfig.actionsSchema.safeParse(action).success
    );
  }
  return false;
});
