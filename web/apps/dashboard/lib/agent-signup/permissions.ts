import { AgentSignupError } from "./errors";

export type PermissionGrant = {
  path: string;
  action: string;
};

// Key management only. decrypt and root-key permissions stay off this list
// even though a workspace admin could grant them in the dashboard.
export const AGENT_PERMISSION_ALLOWLIST: readonly PermissionGrant[] = [
  { path: "projects/*/keyspaces/*", action: "read" },
  { path: "projects/*/keyspaces/*", action: "write" },
  { path: "projects/*/keyspaces/*", action: "delete" },
  { path: "projects/*/keyspaces/*/logs", action: "read" },
  { path: "projects/*/keyspaces/*/keys/*", action: "read" },
  { path: "projects/*/keyspaces/*/keys/*", action: "write" },
  { path: "projects/*/keyspaces/*/keys/*", action: "delete" },
  { path: "projects/*/keyspaces/*/keys/*", action: "verify" },
];

export const DEFAULT_AGENT_PERMISSIONS: readonly PermissionGrant[] = [
  { path: "projects/*/keyspaces/*", action: "read" },
  { path: "projects/*/keyspaces/*", action: "write" },
  { path: "projects/*/keyspaces/*/keys/*", action: "read" },
  { path: "projects/*/keyspaces/*/keys/*", action: "write" },
  { path: "projects/*/keyspaces/*/keys/*", action: "verify" },
];

const ALLOWED = new Set(AGENT_PERMISSION_ALLOWLIST.map((grant) => `${grant.path}#${grant.action}`));

export function permissionUrn(workspaceId: string, grant: PermissionGrant): string {
  return `unkey:v1:${workspaceId}:${grant.path}#${grant.action}`;
}

export function resolveAgentPermissions(
  workspaceId: string,
  requested: readonly PermissionGrant[] | undefined,
): string[] {
  const grants = requested ?? DEFAULT_AGENT_PERMISSIONS;
  if (grants.length === 0) {
    throw new AgentSignupError(
      400,
      "permission_denied",
      "Request at least one permission, or omit permissions to use the default keyspace set.",
    );
  }

  const urns = new Set<string>();
  for (const grant of grants) {
    if (!ALLOWED.has(`${grant.path}#${grant.action}`)) {
      throw new AgentSignupError(
        400,
        "permission_denied",
        "That permission is outside the agent allowlist. Allowed resources are projects/*/keyspaces/*, projects/*/keyspaces/*/logs, and projects/*/keyspaces/*/keys/*. decrypt and root key permissions are not available.",
      );
    }
    urns.add(permissionUrn(workspaceId, grant));
  }
  return [...urns];
}
