"use client";

import type { UnkeyPermission } from "@unkey/rbac";
import type { ActionDefinition, ActionGroupID } from "./types";

export const actionGroups = [
  { id: "keys", label: "Keys" },
  { id: "ratelimits", label: "Ratelimits" },
  { id: "authorization", label: "Authorization" },
  { id: "identities", label: "Identities" },
  { id: "deployments", label: "Deployments" },
  { id: "admin", label: "Admin" },
] as const satisfies ReadonlyArray<{ id: ActionGroupID; label: string }>;

const keyActions = [
  { id: "create_key", label: "Create key", description: "Create keys in the selected keyspace." },
  { id: "read_key", label: "Read key", description: "Read key metadata in the selected keyspace." },
  { id: "update_key", label: "Update key", description: "Update keys in the selected keyspace." },
  { id: "delete_key", label: "Delete key", description: "Delete keys in the selected keyspace." },
  { id: "verify_key", label: "Verify key", description: "Verify keys in the selected keyspace." },
  {
    id: "encrypt_key",
    label: "Encrypt key",
    description: "Create recoverable keys in the selected keyspace.",
  },
  {
    id: "decrypt_key",
    label: "Decrypt key",
    description: "Read recoverable key material in the selected keyspace.",
  },
] as const;

const namespaceActions = [
  { id: "limit", label: "Limit", description: "Consume rate limits in the selected namespace." },
  {
    id: "create_namespace",
    label: "Create namespace",
    description: "Create rate limit namespaces.",
  },
  {
    id: "read_namespace",
    label: "Read namespace",
    description: "Read rate limit namespace metadata.",
  },
  {
    id: "update_namespace",
    label: "Update namespace",
    description: "Update rate limit namespaces.",
  },
  {
    id: "delete_namespace",
    label: "Delete namespace",
    description: "Delete rate limit namespaces.",
  },
] as const;

const overrideActions = [
  {
    id: "set_override",
    label: "Set override",
    description: "Create or update rate limit overrides.",
  },
  { id: "read_override", label: "Read override", description: "Read rate limit overrides." },
  { id: "delete_override", label: "Delete override", description: "Delete rate limit overrides." },
] as const;

const roleActions = [
  { id: "create_role", label: "Create role", description: "Create RBAC roles." },
  { id: "read_role", label: "Read role", description: "Read RBAC roles." },
  { id: "delete_role", label: "Delete role", description: "Delete RBAC roles." },
] as const;

const permissionActions = [
  { id: "create_permission", label: "Create permission", description: "Create RBAC permissions." },
  { id: "read_permission", label: "Read permission", description: "Read RBAC permissions." },
  { id: "delete_permission", label: "Delete permission", description: "Delete RBAC permissions." },
] as const;

const identityActions = [
  { id: "create_identity", label: "Create identity", description: "Create identities." },
  { id: "read_identity", label: "Read identity", description: "Read identities." },
  { id: "update_identity", label: "Update identity", description: "Update identities." },
  { id: "delete_identity", label: "Delete identity", description: "Delete identities." },
] as const;

const deploymentActions = [
  { id: "create_deployment", label: "Create deployment", description: "Create deployments." },
  {
    id: "read_deployment",
    label: "Read deployment",
    description: "Read deployment details and status.",
  },
  {
    id: "generate_upload_url",
    label: "Generate upload URL",
    description: "Generate build upload URLs.",
  },
] as const;

function defineActions<T extends ReadonlyArray<{ id: string; label: string; description: string }>>(
  actions: T,
  metadata: Omit<ActionDefinition, "id" | "label" | "description">,
): ActionDefinition[] {
  return actions.map((action) => ({
    ...action,
    ...metadata,
  }));
}

export function buildActions(): ActionDefinition[] {
  return [
    ...defineActions(keyActions, {
      group: "keys",
      category: "keyspaces",
      categoryLabel: "Keyspaces",
      terminalLabel: "Keys",
    }),
    ...defineActions(namespaceActions, {
      group: "ratelimits",
      category: "ratelimits/namespaces",
      categoryLabel: "Ratelimit namespaces",
      terminalLabel: "Namespace",
    }),
    ...defineActions(overrideActions, {
      group: "ratelimits",
      category: "ratelimits/namespaces",
      categoryLabel: "Ratelimit namespaces",
      terminalLabel: "Overrides",
    }),
    ...defineActions(roleActions, {
      group: "authorization",
      category: "rbac/roles",
      categoryLabel: "Roles",
      terminalLabel: "Role",
    }),
    ...defineActions(permissionActions, {
      group: "authorization",
      category: "rbac/permissions",
      categoryLabel: "Permissions",
      terminalLabel: "Permission",
    }),
    ...defineActions(identityActions, {
      group: "identities",
      category: "identities",
      categoryLabel: "Identities",
      terminalLabel: "Identity",
    }),
    ...defineActions(deploymentActions, {
      group: "deployments",
      category: "projects",
      categoryLabel: "Projects",
      terminalLabel: "Deployment",
    }),
    {
      id: "*",
      label: "All actions",
      description: "Allow every action on every resource in this workspace.",
      group: "admin",
      category: "**",
      categoryLabel: "Workspace",
      terminalLabel: "Everything",
    },
  ];
}

export function permission(workspaceId: string, resource: string, action: string): UnkeyPermission {
  return `unkey:v1:${workspaceId}:${resource}#${action}` as UnkeyPermission;
}

export function togglePermission(
  selectedPermissions: UnkeyPermission[],
  nextPermission: UnkeyPermission,
): UnkeyPermission[] {
  if (selectedPermissions.includes(nextPermission)) {
    return selectedPermissions.filter(
      (selectedPermission) => selectedPermission !== nextPermission,
    );
  }
  return [...selectedPermissions, nextPermission];
}
